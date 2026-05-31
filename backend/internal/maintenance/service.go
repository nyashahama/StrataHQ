package maintenance

import (
	"context"
	"errors"
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

//nolint:govet // Keep API response fields grouped by meaning rather than field packing.
type RequestInfo struct {
	ContractorID    *string    `json:"contractor_id"`
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

//nolint:govet // Keep response DTO fields grouped by meaning rather than field packing.
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

//nolint:govet // Keep input DTO fields grouped by domain meaning rather than field packing.
type AssignInput struct {
	ContractorID    *string
	ContractorName  string
	ContractorPhone *string
}

//nolint:govet // Keep access helper fields grouped by access meaning rather than field packing.
type accessInfo struct {
	scheme         dbgen.Scheme
	role           string
	memberUnitID   *uuid.UUID
	memberUnitName *string
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

	rows, err := s.db.Q.ListMaintenanceRequestsDetailedByScheme(ctx, access.scheme.ID)
	if err != nil {
		return nil, err
	}

	response := &DashboardResponse{
		Requests: []RequestInfo{},
		Role:     access.role,
	}

	now := time.Now().UTC()
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
		if item.Status == string(dbgen.MaintenanceStatusResolved) && item.ResolvedAt != nil && item.ResolvedAt.After(monthStart) {
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

	var created dbgen.MaintenanceRequest
	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		c, txErr := q.CreateMaintenanceRequest(ctx, params)
		if txErr != nil {
			return txErr
		}
		created = c

		if auth.IsResidentRole(access.role) {
			if _, updateErr := q.UpdateMaintenanceStatus(ctx, dbgen.UpdateMaintenanceStatusParams{
				ID:     created.ID,
				Status: dbgen.MaintenanceStatusPendingApproval,
			}); updateErr != nil {
				return updateErr
			}
			created.Status = dbgen.MaintenanceStatusPendingApproval
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	info, err := s.enrichRequest(ctx, created)
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, maintenanceRequestCreatedAuditEvent(maintenanceAuditInput{
			SchemeID:    access.scheme.ID.String(),
			OrgID:       access.scheme.OrgID.String(),
			ActorUserID: identity.UserID,
			ActorRole:   access.role,
			RequestID:   created.ID.String(),
			Title:       info.Title,
			Description: info.Description,
			Category:    info.Category,
			Status:      info.Status,
			SlaHours:    info.SlaHours,
		}))
	}

	return info, nil
}

func (s *Service) Assign(ctx context.Context, identity auth.Identity, schemeID, requestID string, input AssignInput) (*RequestInfo, error) {
	access, err := s.resolveAccess(ctx, identity, schemeID)
	if err != nil {
		return nil, err
	}
	if auth.IsResidentRole(access.role) {
		return nil, ErrForbidden
	}
	if input.ContractorID == nil && input.ContractorName == "" {
		return nil, ErrInvalidInput
	}

	request, err := s.mustGetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if request.SchemeID != access.scheme.ID {
		return nil, ErrForbidden
	}

	beforeInfo, err := s.enrichRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	if input.ContractorID != nil {
		contractorUUID, parseErr := uuid.Parse(*input.ContractorID)
		if parseErr != nil {
			return nil, ErrInvalidInput
		}
		contractor, lookupErr := s.db.Q.ContractorAssignableToScheme(ctx, dbgen.ContractorAssignableToSchemeParams{
			ContractorID: contractorUUID,
			SchemeID:     access.scheme.ID,
		})
		if lookupErr != nil {
			if errors.Is(lookupErr, pgx.ErrNoRows) {
				return nil, ErrForbidden
			}
			return nil, lookupErr
		}
		updated, updateErr := s.db.Q.AssignMaintenanceContractorProfile(ctx, dbgen.AssignMaintenanceContractorProfileParams{
			ID:              request.ID,
			ContractorID:    pgtype.UUID{Bytes: contractor.ID, Valid: true},
			ContractorName:  pgtype.Text{String: contractor.Name, Valid: true},
			ContractorPhone: contractor.Phone,
		})
		if updateErr != nil {
			return nil, updateErr
		}
		afterInfo, enrichErr := s.enrichRequest(ctx, updated)
		if enrichErr != nil {
			return nil, enrichErr
		}
		if s.auditor != nil {
			beforeName := ""
			if beforeInfo.ContractorName != nil {
				beforeName = *beforeInfo.ContractorName
			}
			beforePhone := beforeInfo.ContractorPhone
			_ = s.auditor.RecordResourceEvent(ctx, maintenanceRequestAssignedAuditEvent(maintenanceAuditInput{
				SchemeID:       access.scheme.ID.String(),
				OrgID:          access.scheme.OrgID.String(),
				ActorUserID:    identity.UserID,
				ActorRole:      access.role,
				RequestID:      updated.ID.String(),
				Title:          afterInfo.Title,
				Status:         afterInfo.Status,
				ContractorID:   afterInfo.ContractorID,
				ContractorName: afterInfo.ContractorName,
			}, beforeName, beforePhone))
		}
		return afterInfo, nil
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

	afterInfo, err := s.enrichRequest(ctx, updated)
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		beforeContractorName := ""
		if beforeInfo.ContractorName != nil {
			beforeContractorName = *beforeInfo.ContractorName
		}
		_ = s.auditor.RecordResourceEvent(ctx, maintenanceRequestAssignedAuditEvent(maintenanceAuditInput{
			SchemeID:       access.scheme.ID.String(),
			OrgID:          access.scheme.OrgID.String(),
			ActorUserID:    identity.UserID,
			ActorRole:      access.role,
			RequestID:      updated.ID.String(),
			Title:          afterInfo.Title,
			Status:         afterInfo.Status,
			ContractorID:   afterInfo.ContractorID,
			ContractorName: afterInfo.ContractorName,
		}, beforeContractorName, beforeInfo.ContractorPhone))
	}

	return afterInfo, nil
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
	if request.Status != dbgen.MaintenanceStatusInProgress {
		if request.Status == dbgen.MaintenanceStatusResolved {
			return s.enrichRequest(ctx, request)
		}
		return nil, ErrInvalidInput
	}

	beforeInfo, err := s.enrichRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	resolved, err := s.db.Q.ResolveMaintenanceRequest(ctx, request.ID)
	if err != nil {
		return nil, err
	}

	afterInfo, err := s.enrichRequest(ctx, resolved)
	if err != nil {
		return nil, err
	}

	if s.auditor != nil {
		_ = s.auditor.RecordResourceEvent(ctx, maintenanceRequestResolvedAuditEvent(maintenanceAuditInput{
			SchemeID:    access.scheme.ID.String(),
			OrgID:       access.scheme.OrgID.String(),
			ActorUserID: identity.UserID,
			ActorRole:   access.role,
			RequestID:   resolved.ID.String(),
			Title:       afterInfo.Title,
			Status:      afterInfo.Status,
		}, beforeInfo.Status))
	}

	return afterInfo, nil
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

	now := time.Now().UTC()
	return &RequestInfo{
		ContractorID:    uuidTextPointer(request.ContractorID),
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
		ContractorID:    uuidTextPointer(row.ContractorID),
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

func uuidTextPointer(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	s := uuid.UUID(value.Bytes).String()
	return &s
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

type maintenanceAuditInput struct {
	SchemeID       string
	OrgID          string
	ActorUserID    string
	ActorRole      string
	RequestID      string
	Title          string
	Description    string
	Category       string
	Status         string
	SlaHours       int32
	ContractorID   *string
	ContractorName *string
}

func maintenanceRequestCreatedAuditEvent(input maintenanceAuditInput) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "maintenance_request",
		ResourceID:   input.RequestID,
		Action:       "maintenance.request_created",
		AfterState: map[string]any{
			"title":       input.Title,
			"description": input.Description,
			"category":    input.Category,
			"status":      input.Status,
			"sla_hours":   input.SlaHours,
		},
	}
}

func maintenanceRequestAssignedAuditEvent(input maintenanceAuditInput, beforeContractorName string, beforeContractorPhone *string) audit.ResourceEvent {
	beforeState := map[string]any{
		"title":           input.Title,
		"status":          input.Status,
		"contractor_name": beforeContractorName,
	}
	if beforeContractorPhone != nil {
		beforeState["contractor_phone"] = *beforeContractorPhone
	}
	afterState := map[string]any{
		"title":  input.Title,
		"status": input.Status,
	}
	if input.ContractorID != nil {
		afterState["contractor_id"] = *input.ContractorID
	}
	if input.ContractorName != nil {
		afterState["contractor_name"] = *input.ContractorName
	}
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "maintenance_request",
		ResourceID:   input.RequestID,
		Action:       "maintenance.request_assigned",
		BeforeState:  beforeState,
		AfterState:   afterState,
	}
}

func maintenanceRequestResolvedAuditEvent(input maintenanceAuditInput, beforeStatus string) audit.ResourceEvent {
	return audit.ResourceEvent{
		SchemeID:     input.SchemeID,
		OrgID:        input.OrgID,
		ActorUserID:  input.ActorUserID,
		ActorRole:    input.ActorRole,
		ResourceType: "maintenance_request",
		ResourceID:   input.RequestID,
		Action:       "maintenance.request_resolved",
		BeforeState: map[string]any{
			"title":  input.Title,
			"status": beforeStatus,
		},
		AfterState: map[string]any{
			"title":  input.Title,
			"status": input.Status,
		},
	}
}
