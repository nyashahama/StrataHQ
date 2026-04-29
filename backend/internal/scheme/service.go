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
	"github.com/stratahq/backend/internal/audit"
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
	UnitID               *string        `json:"unit_id"`
	UnitIdentifier       *string        `json:"unit_identifier"`
	NextAgmDate          *string        `json:"next_agm_date"`
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Address              string         `json:"address"`
	Role                 string         `json:"role"`
	Health               string         `json:"health"`
	HealthScore          int            `json:"health_score"`
	HealthBreakdown      map[string]int `json:"health_breakdown"`
	UnitCount            int32          `json:"unit_count"`
	TotalMembers         int            `json:"total_members"`
	TrusteeCount         int            `json:"trustee_count"`
	ResidentCount        int            `json:"resident_count"`
	LevyCollectionPct    int            `json:"levy_collection_pct"`
	OpenMaintenanceCount int64          `json:"open_maintenance_count"`
	NoticeCount          int            `json:"notice_count"`
	DaysToAgm            *int           `json:"days_to_agm"`
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

func (s *Service) List(ctx context.Context, identity auth.Identity) ([]SchemeSummary, error) {
	if auth.IsAdminRole(identity.Role) {
		orgID, err := uuid.Parse(identity.OrgID)
		if err != nil {
			return nil, ErrInvalidInput
		}
		rows, err := s.db.Q.ListSchemeSummariesByOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		summaries := make([]SchemeSummary, 0, len(rows))
		for _, row := range rows {
			var nextAgmDate *string
			var daysToAgm *int
			if row.NextAgmDate.Valid {
				meetingTime := row.NextAgmDate.Time
				date := meetingTime.Format("2006-01-02")
				nextAgmDate = &date
				days := int(math.Ceil(meetingTime.Sub(now).Hours() / 24))
				if days < 0 {
					days = 0
				}
				daysToAgm = &days
			}

			openMaintenanceCount := row.OpenMaintenanceCount
			levyCollectionPct := int(row.LevyCollectionPct)
			score, breakdown := s.computeHealthScore(ctx, row.ID)
			summaries = append(summaries, SchemeSummary{
				ID:                   row.ID.String(),
				Name:                 row.Name,
				Address:              row.Address,
				Role:                 string(auth.RoleAdmin),
				Health:               healthFor(score),
				HealthScore:          score,
				HealthBreakdown:      breakdown,
				UnitCount:            row.UnitCount,
				TotalMembers:         int(row.TotalMembers),
				TrusteeCount:         int(row.TrusteeCount),
				ResidentCount:        int(row.ResidentCount),
				LevyCollectionPct:    levyCollectionPct,
				OpenMaintenanceCount: openMaintenanceCount,
				NoticeCount:          int(row.NoticeCount),
				NextAgmDate:          nextAgmDate,
				DaysToAgm:            daysToAgm,
			})
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

	if auth.IsResidentRole(role) && unitID != nil {
		filtered := make([]dbgen.Unit, 0, 1)
		for _, u := range units {
			if u.ID.String() == *unitID {
				filtered = append(filtered, u)
				break
			}
		}
		units = filtered
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

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, schemeCreatedAuditEvent(schemeAuditInput{
			SchemeID:    scheme.ID.String(),
			OrgID:       scheme.OrgID.String(),
			ActorUserID: identity.UserID,
			ActorRole:   string(auth.RoleAdmin),
			Name:        scheme.Name,
			Address:     scheme.Address,
			UnitCount:   scheme.UnitCount,
		}))
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

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, schemeUpdatedAuditEvent(schemeAuditInput{
			SchemeID:    updated.ID.String(),
			OrgID:       updated.OrgID.String(),
			ActorUserID: identity.UserID,
			ActorRole:   role,
			Name:        updated.Name,
			Address:     updated.Address,
			UnitCount:   updated.UnitCount,
		}, scheme.Name, scheme.Address, scheme.UnitCount))
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

	if err := s.db.Q.DeleteScheme(ctx, scheme.ID); err != nil {
		return err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, schemeDeletedAuditEvent(schemeAuditInput{
			SchemeID:    scheme.ID.String(),
			OrgID:       scheme.OrgID.String(),
			ActorUserID: identity.UserID,
			ActorRole:   role,
			Name:        scheme.Name,
			Address:     scheme.Address,
			UnitCount:   scheme.UnitCount,
		}))
	}

	return nil
}

func (s *Service) ListUnits(ctx context.Context, identity auth.Identity, schemeID string) ([]UnitInfo, error) {
	scheme, role, unitID, _, err := s.resolveSchemeAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}

	units, err := s.db.Q.ListUnitsByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}

	if auth.IsResidentRole(role) && unitID != nil {
		filtered := make([]dbgen.Unit, 0, 1)
		for _, u := range units {
			if u.ID.String() == *unitID {
				filtered = append(filtered, u)
				break
			}
		}
		units = filtered
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

	info := mapUnit(unit)

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, unitCreatedAuditEvent(unitAuditInput{
			SchemeID:        scheme.ID.String(),
			OrgID:           scheme.OrgID.String(),
			ActorUserID:     identity.UserID,
			ActorRole:       role,
			UnitID:          unit.ID.String(),
			Identifier:      info.Identifier,
			OwnerName:       info.OwnerName,
			Floor:           info.Floor,
			SectionValuePct: info.SectionValuePct,
		}))
	}

	return pointerToUnit(info), nil
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

	beforeInfo := mapUnit(current)

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

	afterInfo := mapUnit(unit)

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, unitUpdatedAuditEvent(unitAuditInput{
			SchemeID:        scheme.ID.String(),
			OrgID:           scheme.OrgID.String(),
			ActorUserID:     identity.UserID,
			ActorRole:       role,
			UnitID:          unit.ID.String(),
			Identifier:      afterInfo.Identifier,
			OwnerName:       afterInfo.OwnerName,
			Floor:           afterInfo.Floor,
			SectionValuePct: afterInfo.SectionValuePct,
		}, beforeInfo.Identifier, beforeInfo.OwnerName, beforeInfo.Floor, beforeInfo.SectionValuePct))
	}

	return pointerToUnit(afterInfo), nil
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

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   memberUserID,
		SchemeID: scheme.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	beforeUnitID := ""
	if membership.UnitID.Valid {
		beforeUnitID = uuid.UUID(membership.UnitID.Bytes).String()
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

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if _, txErr := q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
			UserID:   memberUserID,
			SchemeID: scheme.ID,
			UnitID:   unitValue,
			Role:     input.Role,
		}); txErr != nil {
			return txErr
		}

		if _, txErr := q.UpdateOrgMembershipRole(ctx, dbgen.UpdateOrgMembershipRoleParams{
			UserID: memberUserID,
			OrgID:  scheme.OrgID,
			Role:   input.Role,
		}); txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return txErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Q.ListSchemeMembersByScheme(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.UserID == memberUserID {
			member := mapMember(row)

			if s.auditor != nil {
				afterUnitID := ""
				if member.UnitID != nil {
					afterUnitID = *member.UnitID
				}
				_ = s.auditor.RecordResourceEvent(ctx, memberUpdatedAuditEvent(memberAuditInput{
					SchemeID:     scheme.ID.String(),
					OrgID:        scheme.OrgID.String(),
					ActorUserID:  identity.UserID,
					ActorRole:    role,
					UserID:       member.UserID,
					Role:         member.Role,
					UnitID:       afterUnitID,
					BeforeRole:   membership.Role,
					BeforeUnitID: beforeUnitID,
				}))
			}

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
	healthScore, healthBreakdown := s.computeHealthScore(ctx, scheme.ID)
	health := healthFor(healthScore)

	summary := SchemeSummary{
		UnitID:               unitID,
		UnitIdentifier:       unitIdentifier,
		NextAgmDate:          nextAgmDate,
		ID:                   scheme.ID.String(),
		Name:                 scheme.Name,
		Address:              scheme.Address,
		Role:                 role,
		Health:               health,
		HealthScore:          healthScore,
		HealthBreakdown:      healthBreakdown,
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

func healthFor(score int) string {
	switch {
	case score >= 80:
		return "good"
	case score >= 60:
		return "fair"
	default:
		return "poor"
	}
}

type healthFactors struct {
	levyCollection int
	maintenanceSLA int
	compliance     int
	reserveFund    int
	agmRecency     int
}

func (s *Service) computeHealthScore(ctx context.Context, schemeID uuid.UUID) (int, map[string]int) {
	factors := healthFactors{
		levyCollection: s.computeLevyFactor(ctx, schemeID),
		maintenanceSLA: s.computeMaintenanceFactor(ctx, schemeID),
		compliance:     s.complianceFactor(ctx, schemeID),
		reserveFund:    s.reserveFundFactor(ctx, schemeID),
		agmRecency:     s.agmRecencyFactor(ctx, schemeID),
	}

	score := int(math.Round(
		float64(factors.levyCollection)*0.35 +
			float64(factors.maintenanceSLA)*0.25 +
			float64(factors.compliance)*0.20 +
			float64(factors.reserveFund)*0.15 +
			float64(factors.agmRecency)*0.05,
	))

	breakdown := map[string]int{
		"levy_collection": factors.levyCollection,
		"maintenance_sla": factors.maintenanceSLA,
		"compliance":      factors.compliance,
		"reserve_fund":    factors.reserveFund,
		"agm_recency":     factors.agmRecency,
	}

	return score, breakdown
}

func (s *Service) computeLevyFactor(ctx context.Context, schemeID uuid.UUID) int {
	periods, err := s.db.Q.ListLevyPeriodsByScheme(ctx, schemeID)
	if err != nil || len(periods) == 0 {
		return 0
	}
	accounts, err := s.db.Q.ListLevyAccountsByPeriod(ctx, periods[0].ID)
	if err != nil {
		return 0
	}
	var totalDue, totalPaid int64
	for _, a := range accounts {
		totalDue += a.AmountCents
		totalPaid += minInt64(a.PaidCents, a.AmountCents)
	}
	if totalDue == 0 {
		return 100
	}
	return int(math.Round(float64(totalPaid) * 100 / float64(totalDue)))
}

func (s *Service) computeMaintenanceFactor(ctx context.Context, schemeID uuid.UUID) int {
	openCount, err := s.db.Q.CountOpenMaintenanceByScheme(ctx, schemeID)
	if err != nil || openCount == 0 {
		return 100
	}
	slaBreached, err := s.db.Q.CountSlaBreachedMaintenanceByScheme(ctx, schemeID)
	if err != nil {
		return 0
	}
	breachRate := float64(slaBreached) / float64(openCount)
	score := int(math.Round((1 - breachRate) * 100))
	if score < 0 {
		score = 0
	}
	return score
}

func (s *Service) complianceFactor(ctx context.Context, schemeID uuid.UUID) int {
	items, err := s.db.Q.ListComplianceItemsByScheme(ctx, schemeID)
	if err != nil || len(items) == 0 {
		return 100
	}
	var totalPoints, earnedPoints int
	for _, item := range items {
		totalPoints += 10
		switch item.Status {
		case dbgen.ComplianceStatusCompliant:
			earnedPoints += 10
		case dbgen.ComplianceStatusAtRisk:
			earnedPoints += 5
		}
	}
	if totalPoints == 0 {
		return 100
	}
	return (earnedPoints * 100) / totalPoints
}

func (s *Service) reserveFundFactor(ctx context.Context, schemeID uuid.UUID) int {
	fund, err := s.db.Q.GetReserveFund(ctx, schemeID)
	if err != nil || fund.TargetCents == 0 {
		return 100
	}
	pct := int(math.Round(float64(fund.BalanceCents) * 100 / float64(fund.TargetCents)))
	if pct > 100 {
		pct = 100
	}
	return pct
}

func (s *Service) agmRecencyFactor(ctx context.Context, schemeID uuid.UUID) int {
	meetings, err := s.db.Q.ListAgmMeetingsByScheme(ctx, schemeID)
	if err != nil || len(meetings) == 0 {
		return 0
	}
	var latestDate time.Time
	for _, m := range meetings {
		if m.MeetingDate.Valid && m.MeetingDate.Time.After(latestDate) {
			latestDate = m.MeetingDate.Time
		}
	}
	if latestDate.IsZero() {
		return 0
	}
	monthsSince := int(time.Since(latestDate).Hours() / (24 * 30))
	if monthsSince <= 12 {
		return 100
	}
	score := 100 - (monthsSince-12)*5
	if score < 0 {
		score = 0
	}
	return score
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

type schemeAuditInput struct {
	SchemeID    string
	OrgID       string
	ActorUserID string
	ActorRole   string
	Name        string
	Address     string
	UnitCount   int32
}

func schemeCreatedAuditEvent(input schemeAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "scheme",
		ResourceID:   input.SchemeID,
		Action:       "scheme.created",
		AfterState: map[string]any{
			"name":       input.Name,
			"address":    input.Address,
			"unit_count": input.UnitCount,
		},
	}
}

func schemeUpdatedAuditEvent(input schemeAuditInput, beforeName, beforeAddress string, beforeUnitCount int32) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "scheme",
		ResourceID:   input.SchemeID,
		Action:       "scheme.updated",
		BeforeState: map[string]any{
			"name":       beforeName,
			"address":    beforeAddress,
			"unit_count": beforeUnitCount,
		},
		AfterState: map[string]any{
			"name":       input.Name,
			"address":    input.Address,
			"unit_count": input.UnitCount,
		},
	}
}

func schemeDeletedAuditEvent(input schemeAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "scheme",
		ResourceID:   input.SchemeID,
		Action:       "scheme.deleted",
		BeforeState: map[string]any{
			"name":       input.Name,
			"address":    input.Address,
			"unit_count": input.UnitCount,
		},
	}
}

type unitAuditInput struct {
	SchemeID        string
	OrgID           string
	ActorUserID     string
	ActorRole       string
	UnitID          string
	Identifier      string
	OwnerName       string
	Floor           int32
	SectionValuePct float64
}

func unitCreatedAuditEvent(input unitAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "unit",
		ResourceID:   input.UnitID,
		Action:       "unit.created",
		AfterState: map[string]any{
			"identifier":        input.Identifier,
			"owner_name":        input.OwnerName,
			"floor":             input.Floor,
			"section_value_pct": input.SectionValuePct,
		},
	}
}

func unitUpdatedAuditEvent(input unitAuditInput, beforeIdentifier, beforeOwnerName string, beforeFloor int32, beforeSectionValuePct float64) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "unit",
		ResourceID:   input.UnitID,
		Action:       "unit.updated",
		BeforeState: map[string]any{
			"identifier":        beforeIdentifier,
			"owner_name":        beforeOwnerName,
			"floor":             beforeFloor,
			"section_value_pct": beforeSectionValuePct,
		},
		AfterState: map[string]any{
			"identifier":        input.Identifier,
			"owner_name":        input.OwnerName,
			"floor":             input.Floor,
			"section_value_pct": input.SectionValuePct,
		},
	}
}

type memberAuditInput struct {
	SchemeID     string
	OrgID        string
	ActorUserID  string
	ActorRole    string
	UserID       string
	Role         string
	UnitID       string
	BeforeRole   string
	BeforeUnitID string
}

func memberUpdatedAuditEvent(input memberAuditInput) audit.ResourceEvent {
	beforeState := map[string]any{
		"role": input.BeforeRole,
	}
	if input.BeforeUnitID != "" {
		beforeState["unit_id"] = input.BeforeUnitID
	}
	afterState := map[string]any{
		"role": input.Role,
	}
	if input.UnitID != "" {
		afterState["unit_id"] = input.UnitID
	}
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "member",
		ResourceID:   input.UserID,
		Action:       "member.updated",
		BeforeState:  beforeState,
		AfterState:   afterState,
	}
}
