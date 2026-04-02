package scheme

import (
	"context"
	"errors"
	"math"
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

type UnitInfo struct {
	ID              string  `json:"id"`
	Identifier      string  `json:"identifier"`
	OwnerName       string  `json:"owner_name"`
	Floor           int32   `json:"floor"`
	SectionValuePct float64 `json:"section_value_pct"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type MemberInfo struct {
	Phone          *string   `json:"phone"`
	UnitID         *string   `json:"unit_id"`
	UnitIdentifier *string   `json:"unit_identifier"`
	UserID         string    `json:"user_id"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type NoticeInfo struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Type   string    `json:"type"`
	SentAt time.Time `json:"sent_at"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type SchemeSummary struct {
	UnitID               *string `json:"unit_id"`
	UnitIdentifier       *string `json:"unit_identifier"`
	NextAgmDate          *string `json:"next_agm_date"`
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Address              string  `json:"address"`
	Role                 string  `json:"role"`
	Health               string  `json:"health"`
	UnitCount            int32   `json:"unit_count"`
	TotalMembers         int     `json:"total_members"`
	TrusteeCount         int     `json:"trustee_count"`
	ResidentCount        int     `json:"resident_count"`
	LevyCollectionPct    int     `json:"levy_collection_pct"`
	OpenMaintenanceCount int64   `json:"open_maintenance_count"`
	NoticeCount          int     `json:"notice_count"`
	DaysToAgm            *int    `json:"days_to_agm"`
}

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type SchemeDetail struct {
	Units         []UnitInfo   `json:"units"`
	RecentNotices []NoticeInfo `json:"recent_notices"`
	SchemeSummary
}

type CreateSchemeInput struct {
	Name      string
	Address   string
	UnitCount int32
}

type UpdateSchemeInput struct {
	Name      string
	Address   string
	UnitCount int32
}

type CreateUnitInput struct {
	Identifier      string
	OwnerName       string
	Floor           int32
	SectionValueBps int32
}

type UpdateUnitInput struct {
	Identifier      string
	OwnerName       string
	Floor           int32
	SectionValueBps int32
}

type UpdateMemberInput struct {
	UnitID *string
	Role   string
}

type Service struct {
	db *database.Pool
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, identity auth.Identity) ([]SchemeSummary, error) {
	if auth.IsAdminRole(identity.Role) {
		orgID, err := uuid.Parse(identity.OrgID)
		if err != nil {
			return nil, ErrInvalidInput
		}
		schemes, err := s.db.Q.ListSchemesByOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}

		summaries := make([]SchemeSummary, 0, len(schemes))
		for _, scheme := range schemes {
			summary, err := s.buildSummary(ctx, scheme, string(auth.RoleAdmin), nil, nil)
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, summary)
		}
		return summaries, nil
	}

	userID, err := uuid.Parse(identity.UserID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	memberships, err := s.db.Q.ListSchemeMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	summaries := make([]SchemeSummary, 0, len(memberships))
	for _, membership := range memberships {
		scheme, err := s.db.Q.GetScheme(ctx, membership.SchemeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}

		var unitID *string
		if membership.UnitID.Valid {
			id := uuid.UUID(membership.UnitID.Bytes).String()
			unitID = &id
		}
		summary, err := s.buildSummary(ctx, scheme, membership.Role, unitID, textPointer(membership.UnitIdentifier))
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *Service) Get(ctx context.Context, identity auth.Identity, schemeID string) (*SchemeDetail, error) {
	scheme, role, unitID, unitIdentifier, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	summary, err := s.buildSummary(ctx, scheme, role, unitID, unitIdentifier)
	if err != nil {
		return nil, err
	}

	units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	notices, err := s.db.Q.ListNoticesByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	detail := &SchemeDetail{
		SchemeSummary: summary,
		Units:         make([]UnitInfo, 0, len(units)),
		RecentNotices: make([]NoticeInfo, 0, min(3, len(notices))),
	}

	for _, unit := range units {
		detail.Units = append(detail.Units, mapUnit(unit))
	}

	for i, notice := range notices {
		if i == 3 {
			break
		}
		detail.RecentNotices = append(detail.RecentNotices, NoticeInfo{
			ID:     notice.ID.String(),
			Title:  notice.Title,
			Type:   string(notice.Type),
			SentAt: notice.SentAt,
		})
	}

	return detail, nil
}

func (s *Service) Create(ctx context.Context, identity auth.Identity, input CreateSchemeInput) (*SchemeSummary, error) {
	if !auth.IsAdminRole(identity.Role) {
		return nil, ErrForbidden
	}

	orgID, err := uuid.Parse(identity.OrgID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.CreateScheme(ctx, dbgen.CreateSchemeParams{
		OrgID:     orgID,
		Name:      input.Name,
		Address:   input.Address,
		UnitCount: input.UnitCount,
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.buildSummary(ctx, scheme, string(auth.RoleAdmin), nil, nil)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (s *Service) Update(ctx context.Context, identity auth.Identity, schemeID string, input UpdateSchemeInput) (*SchemeSummary, error) {
	scheme, role, unitID, unitIdentifier, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	updated, err := s.db.Q.UpdateScheme(ctx, dbgen.UpdateSchemeParams{
		ID:        scheme.ID,
		Name:      input.Name,
		Address:   input.Address,
		UnitCount: input.UnitCount,
	})
	if err != nil {
		return nil, err
	}

	summary, err := s.buildSummary(ctx, updated, role, unitID, unitIdentifier)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (s *Service) Delete(ctx context.Context, identity auth.Identity, schemeID string) error {
	scheme, role, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return err
	}
	if !auth.IsAdminRole(role) {
		return ErrForbidden
	}
	return s.db.Q.DeleteScheme(ctx, scheme.ID)
}

func (s *Service) ListUnits(ctx context.Context, identity auth.Identity, schemeID string) ([]UnitInfo, error) {
	scheme, _, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	result := make([]UnitInfo, 0, len(units))
	for _, unit := range units {
		result = append(result, mapUnit(unit))
	}
	return result, nil
}

func (s *Service) CreateUnit(ctx context.Context, identity auth.Identity, schemeID string, input CreateUnitInput) (*UnitInfo, error) {
	scheme, role, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	unit, err := s.db.Q.CreateUnit(ctx, dbgen.CreateUnitParams{
		SchemeID:        scheme.ID,
		Identifier:      input.Identifier,
		OwnerName:       input.OwnerName,
		Floor:           input.Floor,
		SectionValueBps: input.SectionValueBps,
	})
	if err != nil {
		return nil, err
	}

	return pointerToUnit(mapUnit(unit)), nil
}

func (s *Service) UpdateUnit(ctx context.Context, identity auth.Identity, schemeID, unitID string, input UpdateUnitInput) (*UnitInfo, error) {
	scheme, role, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}

	uid, err := uuid.Parse(unitID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	current, err := s.db.Q.GetUnit(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if current.SchemeID != scheme.ID {
		return nil, ErrForbidden
	}

	unit, err := s.db.Q.UpdateUnit(ctx, dbgen.UpdateUnitParams{
		ID:              uid,
		Identifier:      input.Identifier,
		OwnerName:       input.OwnerName,
		Floor:           input.Floor,
		SectionValueBps: input.SectionValueBps,
	})
	if err != nil {
		return nil, err
	}

	return pointerToUnit(mapUnit(unit)), nil
}

func (s *Service) ListMembers(ctx context.Context, identity auth.Identity, schemeID string) ([]MemberInfo, error) {
	scheme, role, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	members := make([]MemberInfo, 0, len(rows))
	for _, row := range rows {
		if auth.IsResidentRole(role) && row.Role != string(auth.RoleTrustee) {
			continue
		}
		members = append(members, mapMember(row))
	}

	return members, nil
}

func (s *Service) UpdateMember(ctx context.Context, identity auth.Identity, schemeID, userID string, input UpdateMemberInput) (*MemberInfo, error) {
	scheme, role, _, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if !auth.IsAdminRole(role) {
		return nil, ErrForbidden
	}
	if input.Role != string(auth.RoleTrustee) && input.Role != string(auth.RoleResident) {
		return nil, ErrInvalidInput
	}

	memberUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	_, err = s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   memberUserID,
		SchemeID: scheme.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	unitValue := pgtype.UUID{}
	if input.Role == string(auth.RoleResident) {
		if input.UnitID == nil || *input.UnitID == "" {
			return nil, ErrInvalidInput
		}
		parsedUnitID, parseErr := uuid.Parse(*input.UnitID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		unit, unitErr := s.db.Q.GetUnit(ctx, parsedUnitID)
		if unitErr != nil {
			if errors.Is(unitErr, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, unitErr
		}
		if unit.SchemeID != scheme.ID {
			return nil, ErrForbidden
		}
		unitValue = pgtype.UUID{Bytes: parsedUnitID, Valid: true}
	}

	_, err = s.db.Q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   memberUserID,
		SchemeID: scheme.ID,
		UnitID:   unitValue,
		Role:     input.Role,
	})
	if err != nil {
		return nil, err
	}

	_, err = s.db.Q.UpdateOrgMembershipRole(ctx, dbgen.UpdateOrgMembershipRoleParams{
		UserID: memberUserID,
		OrgID:  scheme.OrgID,
		Role:   input.Role,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	rows, err := s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.UserID == memberUserID {
			member := mapMember(row)
			return &member, nil
		}
	}

	return nil, ErrNotFound
}

func (s *Service) resolveSchemeAccess(ctx context.Context, identity auth.Identity, schemeID string) (dbgen.Scheme, string, *string, *string, error) {
	id, err := uuid.Parse(schemeID)
	if err != nil {
		return dbgen.Scheme{}, "", nil, nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", nil, nil, ErrNotFound
		}
		return dbgen.Scheme{}, "", nil, nil, err
	}

	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil {
			return dbgen.Scheme{}, "", nil, nil, ErrInvalidInput
		}
		if scheme.OrgID != orgID {
			return dbgen.Scheme{}, "", nil, nil, ErrForbidden
		}
		return scheme, string(auth.RoleAdmin), nil, nil, nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return dbgen.Scheme{}, "", nil, nil, ErrInvalidInput
	}

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Scheme{}, "", nil, nil, ErrForbidden
		}
		return dbgen.Scheme{}, "", nil, nil, err
	}

	var memberUnitID *string
	var memberUnitIdentifier *string
	if membership.UnitID.Valid {
		value := uuid.UUID(membership.UnitID.Bytes).String()
		memberUnitID = &value

		unit, unitErr := s.db.Q.GetUnit(ctx, uuid.UUID(membership.UnitID.Bytes))
		if unitErr == nil {
			memberUnitIdentifier = &unit.Identifier
		}
	}

	return scheme, membership.Role, memberUnitID, memberUnitIdentifier, nil
}

func (s *Service) buildSummary(ctx context.Context, scheme dbgen.Scheme, role string, unitID, unitIdentifier *string) (SchemeSummary, error) {
	members, err := s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
	if err != nil {
		return SchemeSummary{}, err
	}

	openMaintenanceCount, err := s.db.Q.CountOpenMaintenanceByScheme(ctx, scheme.ID)
	if err != nil {
		return SchemeSummary{}, err
	}

	notices, err := s.db.Q.ListNoticesByScheme(ctx, scheme.ID)
	if err != nil {
		return SchemeSummary{}, err
	}

	meetings, err := s.db.Q.ListAgmMeetingsByScheme(ctx, scheme.ID)
	if err != nil {
		return SchemeSummary{}, err
	}

	collectionPct, err := s.collectionPct(ctx, scheme.ID)
	if err != nil {
		return SchemeSummary{}, err
	}

	nextAgmDate, daysToAgm := nextAgm(meetings)
	health := healthFor(collectionPct, openMaintenanceCount)

	summary := SchemeSummary{
		UnitID:               unitID,
		UnitIdentifier:       unitIdentifier,
		NextAgmDate:          nextAgmDate,
		ID:                   scheme.ID.String(),
		Name:                 scheme.Name,
		Address:              scheme.Address,
		Role:                 role,
		Health:               health,
		UnitCount:            scheme.UnitCount,
		TotalMembers:         len(members),
		LevyCollectionPct:    collectionPct,
		OpenMaintenanceCount: openMaintenanceCount,
		NoticeCount:          len(notices),
		DaysToAgm:            daysToAgm,
	}

	for _, member := range members {
		switch member.Role {
		case string(auth.RoleTrustee):
			summary.TrusteeCount++
		case string(auth.RoleResident):
			summary.ResidentCount++
		}
	}

	return summary, nil
}

func (s *Service) collectionPct(ctx context.Context, schemeID uuid.UUID) (int, error) {
	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, schemeID)
	if err != nil {
		return 0, err
	}
	if len(periods) == 0 {
		return 100, nil
	}

	accounts, err := s.db.Q.ListLevyAccountsByPeriod(ctx, periods[0].ID)
	if err != nil {
		return 0, err
	}
	if len(accounts) == 0 {
		return 100, nil
	}

	var totalDue int64
	var totalPaid int64
	for _, account := range accounts {
		totalDue += account.AmountCents
		totalPaid += minInt64(account.PaidCents, account.AmountCents)
	}
	if totalDue == 0 {
		return 100, nil
	}

	return int(math.Round(float64(totalPaid) * 100 / float64(totalDue))), nil
}

func mapUnit(unit dbgen.Unit) UnitInfo {
	return UnitInfo{
		ID:              unit.ID.String(),
		Identifier:      unit.Identifier,
		OwnerName:       unit.OwnerName,
		Floor:           unit.Floor,
		SectionValuePct: float64(unit.SectionValueBps) / 100,
	}
}

func mapMember(row dbgen.ListSchemeMembersBySchemeRow) MemberInfo {
	member := MemberInfo{
		Phone:     textPointer(row.Phone),
		UserID:    row.UserID.String(),
		FullName:  row.FullName,
		Email:     row.Email,
		Role:      row.Role,
		CreatedAt: row.CreatedAt,
	}
	if row.UnitID.Valid {
		unitID := uuid.UUID(row.UnitID.Bytes).String()
		member.UnitID = &unitID
	}
	if row.UnitIdentifier.Valid {
		unitIdentifier := row.UnitIdentifier.String
		member.UnitIdentifier = &unitIdentifier
	}
	return member
}

func pointerToUnit(unit UnitInfo) *UnitInfo {
	return &unit
}

func nextAgm(meetings []dbgen.AgmMeeting) (*string, *int) {
	now := time.Now()
	var next *time.Time
	for _, meeting := range meetings {
		if !meeting.MeetingDate.Valid {
			continue
		}
		meetingTime := meeting.MeetingDate.Time
		if meetingTime.Before(now) {
			continue
		}
		if next == nil || meetingTime.Before(*next) {
			copy := meetingTime
			next = &copy
		}
	}
	if next == nil {
		return nil, nil
	}

	date := next.Format("2006-01-02")
	days := int(math.Ceil(next.Sub(now).Hours() / 24))
	return &date, &days
}

func healthFor(collectionPct int, openMaintenanceCount int64) string {
	switch {
	case collectionPct >= 95 && openMaintenanceCount <= 2:
		return "good"
	case collectionPct >= 75 && openMaintenanceCount <= 6:
		return "fair"
	default:
		return "poor"
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}
