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
	"github.com/stratahq/backend/internal/audit"
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

type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	db      *database.Pool
	auditor resourceAuditor
}

func NewService(db *database.Pool) *Service {
	return NewServiceWithAudit(db, nil)
}

func NewServiceWithAudit(db *database.Pool, auditor resourceAuditor) *Service {
	return &Service{db: db, auditor: auditor}
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

	_ = s.auditor.RecordResourceEvent(ctx, audit.ResourceEvent{
		SchemeID:     access.scheme.ID.String(),
		OrgID:        access.scheme.OrgID.String(),
		ActorUserID:  access.userID,
		ActorRole:    access.role,
		ResourceType: "agm_meeting",
		ResourceID:   meeting.ID.String(),
		Action:       "agm_meeting.scheduled",
		AfterState: map[string]any{
			"meeting_date":     meeting.MeetingDate.Time.Format("2006-01-02"),
			"quorum_required":  meeting.QuorumRequired,
			"resolution_count": len(input.Resolutions),
		},
	})

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
	assignments, err := s.db.Q.ListProxyAssignmentsByMeeting(ctx, meeting.ID)
	if err != nil {
		return nil, err
	}
	if hasOutgoingProxyAssignment(assignments, userID) {
		return nil, ErrForbidden
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
	votesFor, votesAgainst := calculateVoteTotals(votes, assignments)

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

	_ = s.auditor.RecordResourceEvent(ctx, agmVoteAuditEvent(agmVoteAuditInput{
		SchemeID:     access.scheme.ID.String(),
		OrgID:        access.scheme.OrgID.String(),
		ActorUserID:  access.userID,
		ActorRole:    access.role,
		ResolutionID: resolution.ID.String(),
		Choice:       input.Choice,
	}))

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

	assignments, err := s.db.Q.ListProxyAssignmentsByMeeting(ctx, meeting.ID)
	if err != nil {
		return err
	}
	if hasOutgoingProxyAssignment(assignments, grantorID) {
		return ErrForbidden
	}
	if hasIncomingProxyAssignment(assignments, grantorID) {
		return ErrForbidden
	}
	if hasOutgoingProxyAssignment(assignments, granteeID) {
		return ErrForbidden
	}
	if voted, voteErr := s.userHasMeetingVote(ctx, meeting.ID, grantorID); voteErr != nil {
		return voteErr
	} else if voted {
		return ErrForbidden
	}
	if voted, voteErr := s.userHasMeetingVote(ctx, meeting.ID, granteeID); voteErr != nil {
		return voteErr
	} else if voted {
		return ErrForbidden
	}

	_, err = s.db.Q.CreateProxyAssignment(ctx, dbgen.CreateProxyAssignmentParams{
		MeetingID:     meeting.ID,
		GrantorUserID: grantorID,
		GranteeUserID: granteeID,
	})
	if err != nil {
		return err
	}

	_ = s.auditor.RecordResourceEvent(ctx, audit.ResourceEvent{
		SchemeID:     access.scheme.ID.String(),
		OrgID:        access.scheme.OrgID.String(),
		ActorUserID:  access.userID,
		ActorRole:    access.role,
		ResourceType: "agm_meeting",
		ResourceID:   meeting.ID.String(),
		Action:       "agm.proxy_assigned",
		AfterState: map[string]any{
			"grantor_user_id": grantorID.String(),
			"grantee_user_id": granteeID.String(),
		},
	})

	return nil
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

func hasOutgoingProxyAssignment(assignments []dbgen.ProxyAssignment, userID uuid.UUID) bool {
	for _, assignment := range assignments {
		if assignment.GrantorUserID == userID {
			return true
		}
	}
	return false
}

func hasIncomingProxyAssignment(assignments []dbgen.ProxyAssignment, userID uuid.UUID) bool {
	for _, assignment := range assignments {
		if assignment.GranteeUserID == userID {
			return true
		}
	}
	return false
}

func calculateVoteTotals(votes []dbgen.AgmVote, assignments []dbgen.ProxyAssignment) (int32, int32) {
	directVotes := make(map[uuid.UUID]struct{}, len(votes))
	for _, vote := range votes {
		directVotes[vote.VoterUserID] = struct{}{}
	}

	incomingWeights := make(map[uuid.UUID]int32)
	for _, assignment := range assignments {
		if _, grantorVotedDirectly := directVotes[assignment.GrantorUserID]; grantorVotedDirectly {
			continue
		}
		incomingWeights[assignment.GranteeUserID]++
	}

	var votesFor int32
	var votesAgainst int32
	for _, vote := range votes {
		weight := int32(1) + incomingWeights[vote.VoterUserID]
		switch vote.Vote {
		case dbgen.VoteChoiceFor:
			votesFor += weight
		case dbgen.VoteChoiceAgainst:
			votesAgainst += weight
		}
	}

	return votesFor, votesAgainst
}

func (s *Service) userHasMeetingVote(ctx context.Context, meetingID, userID uuid.UUID) (bool, error) {
	resolutions, err := s.db.Q.ListAgmResolutionsByMeeting(ctx, meetingID)
	if err != nil {
		return false, err
	}
	for _, resolution := range resolutions {
		_, voteErr := s.db.Q.GetAgmVote(ctx, dbgen.GetAgmVoteParams{
			ResolutionID: resolution.ID,
			VoterUserID:  userID,
		})
		if voteErr == nil {
			return true, nil
		}
		if !errors.Is(voteErr, pgx.ErrNoRows) {
			return false, voteErr
		}
	}
	return false, nil
}

type agmVoteAuditInput struct {
	SchemeID     string
	OrgID        string
	ActorUserID  string
	ActorRole    string
	ResolutionID string
	Choice       string
}

func agmVoteAuditEvent(input agmVoteAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "agm_resolution",
		ResourceID:   input.ResolutionID,
		Action:       "agm.vote_cast",
		AfterState: map[string]any{
			"choice": input.Choice,
		},
	}
}
