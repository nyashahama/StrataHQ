package maintenance

import (
	"context"
	"errors"
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

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type RequestInfo struct {
	ContractorName  *string    `json:"contractor_name"`
	ContractorPhone *string    `json:"contractor_phone"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	UnitID          *string    `json:"unit_id"`
	UnitIdentifier  *string    `json:"unit_identifier"`
	OwnerName       *string    `json:"owner_name"`
	ID              string     `json:"id"`
	SchemeID        string     `json:"scheme_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Category        string     `json:"category"`
	Status          string     `json:"status"`
	SubmittedByUnit *string    `json:"submitted_by_unit"`
	SlaHours        int32      `json:"sla_hours"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	SlaBreached     bool       `json:"sla_breached"`
}

type DashboardResponse struct {
	Requests             []RequestInfo `json:"requests"`
	Role                 string        `json:"role"`
	OpenCount            int           `json:"open_count"`
	SlaBreachedCount     int           `json:"sla_breached_count"`
	PendingApprovalCount int           `json:"pending_approval_count"`
	ResolvedThisMonth    int           `json:"resolved_this_month"`
}

type CreateInput struct {
	Title       string
	Description string
	Category    string
}

type AssignInput struct {
	ContractorName  string
	ContractorPhone *string
}

type accessInfo struct {
	scheme         dbgen.Scheme
	role           string
	memberUnitID   *uuid.UUID
	memberUnitName *string
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

	rows, err := s.db.Q.ListMaintenanceRequestsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	response := &DashboardResponse{
		Requests: []RequestInfo{},
		Role:     access.role,
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for _, row := range rows {
		if auth.IsResidentRole(access.role) && !sameUnit(row.UnitID, access.memberUnitID) {
			continue
		}

		item := mapRequestRow(row, now)
		response.Requests = append(response.Requests, item)
		if item.Status != string(dbgen.MaintenanceStatusResolved) {
			response.OpenCount++
		}
		if item.SlaBreached {
			response.SlaBreachedCount++
		}
		if item.Status == string(dbgen.MaintenanceStatusPendingApproval) {
			response.PendingApprovalCount++
		}
		if item.ResolvedAt != nil && item.ResolvedAt.After(monthStart) {
			response.ResolvedThisMonth++
		}
	}

	return response, nil
}

func (s *Service) Create(ctx context.Context, identity auth.Identity, schemeID string, input CreateInput) (*RequestInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if input.Title == "" || input.Description == "" || !validCategory(input.Category) {
		return nil, ErrInvalidInput
	}

	params := dbgen.CreateMaintenanceRequestParams{
		SchemeID:    access.scheme.ID,
		Title:       input.Title,
		Description: input.Description,
		Category:    dbgen.MaintenanceCategory(input.Category),
		SlaHours:    defaultSLAHours(input.Category, access.role),
	}

	if auth.IsResidentRole(access.role) {
		if access.memberUnitID == nil || access.memberUnitName == nil {
			return nil, ErrForbidden
		}
		params.UnitID = pgtype.UUID{Bytes: *access.memberUnitID, Valid: true}
		params.SubmittedByUnit = pgtype.Text{String: *access.memberUnitName, Valid: true}
	} else {
		params.UnitID = pgtype.UUID{}
		params.SubmittedByUnit = pgtype.Text{}
	}

	created, err := s.db.Q.CreateMaintenanceRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	if auth.IsResidentRole(access.role) {
		if _, err := s.db.Q.UpdateMaintenanceStatus(ctx, dbgen.UpdateMaintenanceStatusParams{
			ID:     created.ID,
			Status: dbgen.MaintenanceStatusPendingApproval,
		}); err != nil {
			return nil, err
		}
		created.Status = dbgen.MaintenanceStatusPendingApproval
	}

	return s.enrichRequest(ctx, created)
}

func (s *Service) Assign(ctx context.Context, identity auth.Identity, schemeID, requestID string, input AssignInput) (*RequestInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) || input.ContractorName == "" {
		return nil, ErrForbidden
	}

	request, err := s.mustGetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if request.SchemeID != access.scheme.ID {
		return nil, ErrForbidden
	}

	phone := pgtype.Text{}
	if input.ContractorPhone != nil && *input.ContractorPhone != "" {
		phone = pgtype.Text{String: *input.ContractorPhone, Valid: true}
	}

	updated, err := s.db.Q.AssignMaintenanceContractor(ctx, dbgen.AssignMaintenanceContractorParams{
		ID:              request.ID,
		ContractorName:  pgtype.Text{String: input.ContractorName, Valid: true},
		ContractorPhone: phone,
	})
	if err != nil {
		return nil, err
	}

	return s.enrichRequest(ctx, updated)
}

func (s *Service) Resolve(ctx context.Context, identity auth.Identity, schemeID, requestID string) (*RequestInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}

	request, err := s.mustGetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if request.SchemeID != access.scheme.ID {
		return nil, ErrForbidden
	}

	resolved, err := s.db.Q.ResolveMaintenanceRequest(ctx, request.ID)
	if err != nil {
		return nil, err
	}

	return s.enrichRequest(ctx, resolved)
}

func (s *Service) resolveAccess(ctx context.Context, identity auth.Identity, schemeID string) (*accessInfo, error) {
	sid, err := uuid.Parse(schemeID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	scheme, err := s.db.Q.GetScheme(ctx, sid)
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
		return &accessInfo{scheme: scheme, role: string(auth.RoleAdmin)}, nil
	}

	userID, parseErr := uuid.Parse(identity.UserID)
	if parseErr != nil {
		return nil, ErrInvalidInput
	}

	membership, err := s.db.Q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
		UserID:   userID,
		SchemeID: sid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	info := &accessInfo{
		scheme: scheme,
		role:   membership.Role,
	}
	if membership.UnitID.Valid {
		unitID := uuid.UUID(membership.UnitID.Bytes)
		info.memberUnitID = &unitID
		unit, unitErr := s.db.Q.GetUnit(ctx, unitID)
		if unitErr == nil {
			unitIdentifier := unit.Identifier
			info.memberUnitName = &unitIdentifier
		}
	}

	return info, nil
}

func (s *Service) mustGetRequest(ctx context.Context, requestID string) (dbgen.MaintenanceRequest, error) {
	id, err := uuid.Parse(requestID)
	if err != nil {
		return dbgen.MaintenanceRequest{}, ErrInvalidInput
	}

	request, err := s.db.Q.GetMaintenanceRequest(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dbgen.MaintenanceRequest{}, ErrNotFound
		}
		return dbgen.MaintenanceRequest{}, err
	}
	return request, nil
}

func (s *Service) enrichRequest(ctx context.Context, request dbgen.MaintenanceRequest) (*RequestInfo, error) {
	var unitIdentifier *string
	var ownerName *string
	var unitID *string
	if request.UnitID.Valid {
		value := uuid.UUID(request.UnitID.Bytes)
		stringValue := value.String()
		unitID = &stringValue
		unit, err := s.db.Q.GetUnit(ctx, value)
		if err == nil {
			unitIdentifier = &unit.Identifier
			ownerName = &unit.OwnerName
		}
	}

	now := time.Now()
	return &RequestInfo{
		ContractorName:  textPointer(request.ContractorName),
		ContractorPhone: textPointer(request.ContractorPhone),
		ResolvedAt:      timePointer(request.ResolvedAt),
		UnitID:          unitID,
		UnitIdentifier:  unitIdentifier,
		OwnerName:       ownerName,
		ID:              request.ID.String(),
		SchemeID:        request.SchemeID.String(),
		Title:           request.Title,
		Description:     request.Description,
		Category:        string(request.Category),
		Status:          string(request.Status),
		SubmittedByUnit: textPointer(request.SubmittedByUnit),
		SlaHours:        request.SlaHours,
		CreatedAt:       request.CreatedAt,
		UpdatedAt:       request.UpdatedAt,
		SlaBreached:     isSlaBreached(request.CreatedAt, request.SlaHours, request.Status, now),
	}, nil
}

func mapRequestRow(row dbgen.ListMaintenanceRequestsDetailedBySchemeRow, now time.Time) RequestInfo {
	var unitID *string
	if row.UnitID.Valid {
		value := uuid.UUID(row.UnitID.Bytes).String()
		unitID = &value
	}

	return RequestInfo{
		ContractorName:  textPointer(row.ContractorName),
		ContractorPhone: textPointer(row.ContractorPhone),
		ResolvedAt:      timestamptzPointer(row.ResolvedAt),
		UnitID:          unitID,
		UnitIdentifier:  textPointer(row.UnitIdentifier),
		OwnerName:       textPointer(row.OwnerName),
		ID:              row.ID.String(),
		SchemeID:        row.SchemeID.String(),
		Title:           row.Title,
		Description:     row.Description,
		Category:        string(row.Category),
		Status:          string(row.Status),
		SubmittedByUnit: textPointer(row.SubmittedByUnit),
		SlaHours:        row.SlaHours,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		SlaBreached:     isSlaBreached(row.CreatedAt, row.SlaHours, row.Status, now),
	}
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	copy := value.Time
	return &copy
}

func timestamptzPointer(value pgtype.Timestamptz) *time.Time {
	return timePointer(value)
}

func isSlaBreached(createdAt time.Time, slaHours int32, status dbgen.MaintenanceStatus, now time.Time) bool {
	if status == dbgen.MaintenanceStatusResolved {
		return false
	}
	return createdAt.Add(time.Duration(slaHours) * time.Hour).Before(now)
}

func validCategory(category string) bool {
	switch category {
	case string(dbgen.MaintenanceCategoryPlumbing),
		string(dbgen.MaintenanceCategoryElectrical),
		string(dbgen.MaintenanceCategoryStructural),
		string(dbgen.MaintenanceCategoryGarden),
		string(dbgen.MaintenanceCategoryPool),
		string(dbgen.MaintenanceCategoryOther):
		return true
	default:
		return false
	}
}

func defaultSLAHours(category, role string) int32 {
	if auth.IsResidentRole(role) {
		return 72
	}
	switch category {
	case string(dbgen.MaintenanceCategoryElectrical):
		return 24
	case string(dbgen.MaintenanceCategoryGarden):
		return 8
	case string(dbgen.MaintenanceCategoryPool):
		return 72
	case string(dbgen.MaintenanceCategoryStructural):
		return 96
	default:
		return 48
	}
}

func sameUnit(value pgtype.UUID, expected *uuid.UUID) bool {
	if expected == nil || !value.Valid {
		return false
	}
	return uuid.UUID(value.Bytes) == *expected
}
