package agm

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type ResolutionInfo struct {
	UserVote      *string   `json:"user_vote"`
	ID            string    `json:"id"`
	MeetingID     string    `json:"meeting_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	VotesFor      int32     `json:"votes_for"`
	VotesAgainst  int32     `json:"votes_against"`
	TotalEligible int32     `json:"total_eligible"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
type MeetingInfo struct {
	UserProxyGranteeID *string          `json:"user_proxy_grantee_id"`
	Resolutions        []ResolutionInfo `json:"resolutions"`
	ID                 string           `json:"id"`
	SchemeID           string           `json:"scheme_id"`
	MeetingDate        string           `json:"date"`
	Status             string           `json:"status"`
	QuorumRequired     int32            `json:"quorum_required"`
	QuorumPresent      int32            `json:"quorum_present"`
}

type DashboardResponse struct {
	Latest   *MeetingInfo `json:"latest"`
	Upcoming *MeetingInfo `json:"upcoming"`
	Role     string       `json:"role"`
}

//nolint:govet // Keep input DTO fields grouped by meaning rather than field packing.
type ScheduleMeetingInput struct {
	MeetingDate    time.Time
	QuorumRequired int32
	Resolutions    []ScheduleResolutionInput
}

type ScheduleResolutionInput struct {
	Title       string
	Description string
}

type CastVoteInput struct {
	Choice string
}

type AssignProxyInput struct {
	GranteeUserID string
}

type accessInfo struct {
	scheme dbgen.Scheme
	role   string
	userID string
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Dashboard(ctx context.Context, identity auth.Identity, schemeID string) (*DashboardResponse, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	meetings, err := s.db.Q.ListAgmMeetingsByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}
	if len(meetings) == 0 {
		return &DashboardResponse{Role: access.role}, nil
	}

	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].MeetingDate.Time.After(meetings[j].MeetingDate.Time)
	})

	response := &DashboardResponse{Role: access.role}
	now := startOfDay(time.Now())

	for _, meeting := range meetings {
		meetingDate := meeting.MeetingDate.Time
		switch {
		case meetingDate.Before(now) || meeting.Status == dbgen.AgmStatusClosed:
			if response.Latest == nil {
				item, err := s.buildMeeting(ctx, access, meeting)
				if err != nil {
					return nil, err
				}
				response.Latest = item
			}
		default:
			if response.Upcoming == nil || meetingDate.Before(parseDate(response.Upcoming.MeetingDate)) {
				item, err := s.buildMeeting(ctx, access, meeting)
				if err != nil {
					return nil, err
				}
				response.Upcoming = item
			}
		}
	}

	return response, nil
}

func (s *Service) ScheduleMeeting(ctx context.Context, identity auth.Identity, schemeID string, input ScheduleMeetingInput) (*MeetingInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(access.role) {
		return nil, ErrForbidden
	}
	if input.MeetingDate.IsZero() || input.QuorumRequired <= 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.db.Q.WithTx(tx)
	meeting, err := q.CreateAgmMeeting(ctx, dbgen.CreateAgmMeetingParams{
		SchemeID:       access.scheme.ID,
		MeetingDate:    dateValue(input.MeetingDate),
		QuorumRequired: input.QuorumRequired,
	})
	if err != nil {
		return nil, err
	}

	totalEligible, err := s.totalEligibleVoters(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	for _, resolution := range input.Resolutions {
		if strings.TrimSpace(resolution.Title) == "" || strings.TrimSpace(resolution.Description) == "" {
			return nil, ErrInvalidInput
		}
		if _, createErr := q.CreateAgmResolution(ctx, dbgen.CreateAgmResolutionParams{
			MeetingID:     meeting.ID,
			Title:         strings.TrimSpace(resolution.Title),
			Description:   strings.TrimSpace(resolution.Description),
			TotalEligible: totalEligible,
		}); createErr != nil {
			return nil, createErr
		}
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, commitErr
	}

	createdMeeting, err := s.db.Q.GetAgmMeeting(ctx, meeting.ID)
	if err != nil {
		return nil, err
	}
	return s.buildMeeting(ctx, access, createdMeeting)
}

func (s *Service) CastVote(ctx context.Context, identity auth.Identity, schemeID, resolutionID string, input CastVoteInput) (*ResolutionInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsAdminRole(access.role) {
		return nil, ErrForbidden
	}
	if input.Choice != string(dbgen.VoteChoiceFor) && input.Choice != string(dbgen.VoteChoiceAgainst) {
		return nil, ErrInvalidInput
	}

	resUUID, err := uuid.Parse(resolutionID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	resolution, err := s.db.Q.GetAgmResolution(ctx, resUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	meeting, err := s.db.Q.GetAgmMeeting(ctx, resolution.MeetingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if meeting.SchemeID != access.scheme.ID || resolution.Status != dbgen.ResolutionStatusOpen {
		return nil, ErrForbidden
	}

	userID, err := uuid.Parse(access.userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if _, voteErr := s.db.Q.GetAgmVote(ctx, dbgen.GetAgmVoteParams{ResolutionID: resolution.ID, VoterUserID: userID}); voteErr == nil {
		return nil, ErrForbidden
	} else if !errors.Is(voteErr, pgx.ErrNoRows) {
		return nil, voteErr
	}

	if _, createErr := s.db.Q.CreateAgmVote(ctx, dbgen.CreateAgmVoteParams{
		ResolutionID: resolution.ID,
		VoterUserID:  userID,
		Vote:         dbgen.VoteChoice(input.Choice),
	}); createErr != nil {
		return nil, createErr
	}

	votes, err := s.db.Q.ListAgmVotesByResolution(ctx, resolution.ID)
	if err != nil {
		return nil, err
	}
	var votesFor int32
	var votesAgainst int32
	for _, vote := range votes {
		switch vote.Vote {
		case dbgen.VoteChoiceFor:
			votesFor++
		case dbgen.VoteChoiceAgainst:
			votesAgainst++
		}
	}

	updated, err := s.db.Q.UpdateAgmResolutionVotes(ctx, dbgen.UpdateAgmResolutionVotesParams{
		ID:           resolution.ID,
		VotesFor:     votesFor,
		VotesAgainst: votesAgainst,
	})
	if err != nil {
		return nil, err
	}

	if votesFor+votesAgainst >= updated.TotalEligible {
		nextStatus := dbgen.ResolutionStatusFailed
		if votesFor > votesAgainst {
			nextStatus = dbgen.ResolutionStatusPassed
		}
		updated, err = s.db.Q.UpdateAgmResolutionStatus(ctx, dbgen.UpdateAgmResolutionStatusParams{
			ID:     updated.ID,
			Status: nextStatus,
		})
		if err != nil {
			return nil, err
		}
	}

	item := mapResolution(updated)
	choice := input.Choice
	item.UserVote = &choice
	return &item, nil
}

func (s *Service) AssignProxy(ctx context.Context, identity auth.Identity, schemeID, meetingID string, input AssignProxyInput) error {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return err
	}
	if auth.IsAdminRole(access.role) {
		return ErrForbidden
	}

	meetUUID, err := uuid.Parse(meetingID)
	if err != nil {
		return ErrInvalidInput
	}
	meeting, err := s.db.Q.GetAgmMeeting(ctx, meetUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if meeting.SchemeID != access.scheme.ID {
		return ErrForbidden
	}

	grantorID, err := uuid.Parse(access.userID)
	if err != nil {
		return ErrInvalidInput
	}
	granteeID, err := uuid.Parse(input.GranteeUserID)
	if err != nil || grantorID == granteeID {
		return ErrInvalidInput
	}

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   granteeID,
		SchemeID: access.scheme.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		return err
	}
	if membership.Role != string(auth.RoleTrustee) && membership.Role != string(auth.RoleResident) {
		return ErrForbidden
	}

	if _, proxyErr := s.db.Q.GetProxyAssignment(ctx, dbgen.GetProxyAssignmentParams{
		MeetingID:     meeting.ID,
		GrantorUserID: grantorID,
	}); proxyErr == nil {
		return ErrForbidden
	} else if !errors.Is(proxyErr, pgx.ErrNoRows) {
		return proxyErr
	}

	_, err = s.db.Q.CreateProxyAssignment(ctx, dbgen.CreateProxyAssignmentParams{
		MeetingID:     meeting.ID,
		GrantorUserID: grantorID,
		GranteeUserID: granteeID,
	})
	return err
}

func (s *Service) buildMeeting(ctx context.Context, access *accessInfo, meeting dbgen.AgmMeeting) (*MeetingInfo, error) {
	resolutions, err := s.db.Q.ListAgmResolutionsByMeeting(ctx, meeting.ID)
	if err != nil {
		return nil, err
	}

	var proxyGranteeID *string
	if access.userID != "" {
		userID, parseErr := uuid.Parse(access.userID)
		if parseErr == nil {
			proxy, proxyErr := s.db.Q.GetProxyAssignment(ctx, dbgen.GetProxyAssignmentParams{
				MeetingID:     meeting.ID,
				GrantorUserID: userID,
			})
			if proxyErr == nil {
				value := proxy.GranteeUserID.String()
				proxyGranteeID = &value
			}
		}
	}

	items := make([]ResolutionInfo, 0, len(resolutions))
	for _, resolution := range resolutions {
		item := mapResolution(resolution)
		if access.userID != "" {
			userID, parseErr := uuid.Parse(access.userID)
			if parseErr == nil {
				vote, voteErr := s.db.Q.GetAgmVote(ctx, dbgen.GetAgmVoteParams{
					ResolutionID: resolution.ID,
					VoterUserID:  userID,
				})
				if voteErr == nil {
					choice := string(vote.Vote)
					item.UserVote = &choice
				}
			}
		}
		items = append(items, item)
	}

	return &MeetingInfo{
		UserProxyGranteeID: proxyGranteeID,
		Resolutions:        items,
		ID:                 meeting.ID.String(),
		SchemeID:           meeting.SchemeID.String(),
		MeetingDate:        meeting.MeetingDate.Time.Format("2006-01-02"),
		Status:             string(meeting.Status),
		QuorumRequired:     meeting.QuorumRequired,
		QuorumPresent:      meeting.QuorumPresent,
	}, nil
}

func (s *Service) totalEligibleVoters(ctx context.Context, schemeID uuid.UUID) (int32, error) {
	members, err := s.db.Q.ListSchemeMembersByScheme(ctx, schemeID)
	if err != nil {
		return 0, err
	}
	return int32(len(members)), nil
}

func (s *Service) resolveAccess(ctx context.Context, identity auth.Identity, schemeID string) (*accessInfo, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return nil, ErrForbidden
		}
		return &accessInfo{scheme: scheme, role: string(auth.RoleAdmin), userID: identity.UserID}, nil
	}
	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return nil, ErrInvalidInput
	}
	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}
	return &accessInfo{scheme: scheme, role: membership.Role, userID: identity.UserID}, nil
}

func mapResolution(resolution dbgen.AgmResolution) ResolutionInfo {
	return ResolutionInfo{
		ID:            resolution.ID.String(),
		MeetingID:     resolution.MeetingID.String(),
		Title:         resolution.Title,
		Description:   resolution.Description,
		VotesFor:      resolution.VotesFor,
		VotesAgainst:  resolution.VotesAgainst,
		TotalEligible: resolution.TotalEligible,
		Status:        string(resolution.Status),
		CreatedAt:     resolution.CreatedAt,
	}
}

func dateValue(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func parseDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
