package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/database"
)

type Event struct {
	ActorUserID  string
	OrgID        string
	ActorRole    string
	Method       string
	Path         string
	RoutePattern string
	IPAddress    string
	UserAgent    string
	OccurredAtNS int64
	StatusCode   int
}

type Recorder interface {
	Record(ctx context.Context, event Event) error
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type Service struct {
	db execer
}

func NewService(db *database.Pool) *Service {
	return &Service{db: db}
}

const insertAuditEvent = `
INSERT INTO audit_events (
	actor_user_id,
	org_id,
	actor_role,
	method,
	path,
	route_pattern,
	status_code,
	ip_address,
	user_agent,
	occurred_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`

func (s *Service) Record(ctx context.Context, event Event) error {
	if s == nil || s.db == nil {
		return nil
	}

	occurredAt := time.Unix(0, event.OccurredAtNS).UTC()
	if event.OccurredAtNS == 0 {
		occurredAt = time.Now().UTC()
	}

	routePattern := strings.TrimSpace(event.RoutePattern)
	if routePattern == "" {
		routePattern = strings.TrimSpace(event.Path)
	}

	_, err := s.db.Exec(ctx, insertAuditEvent,
		parseOptionalUUID(event.ActorUserID),
		parseOptionalUUID(event.OrgID),
		strings.TrimSpace(event.ActorRole),
		strings.TrimSpace(event.Method),
		strings.TrimSpace(event.Path),
		routePattern,
		event.StatusCode,
		strings.TrimSpace(event.IPAddress),
		strings.TrimSpace(event.UserAgent),
		occurredAt,
	)
	return err
}

func parseOptionalUUID(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil
	}
	return parsed
}

var ErrInvalidResourceEvent = errors.New("invalid resource audit event")
var ErrForbidden = errors.New("forbidden")
var ErrNotFound = errors.New("not found")

type ResourceEvent struct {
	SchemeID     string
	OrgID        string
	ActorUserID  string
	ActorRole    string
	ResourceType string
	ResourceID   string
	Action       string
	BeforeState  any
	AfterState   any
	Metadata     any
}

type ResourceRecorder interface {
	RecordResourceEvent(ctx context.Context, event ResourceEvent) error
}

type ResourceAuditQueries interface {
	CreateResourceAuditEvent(ctx context.Context, arg dbgen.CreateResourceAuditEventParams) (dbgen.ResourceAuditEvent, error)
	ListResourceAuditEventsByScheme(ctx context.Context, arg dbgen.ListResourceAuditEventsBySchemeParams) ([]dbgen.ResourceAuditEvent, error)
	ListResourceAuditEventsBySchemeAndAction(ctx context.Context, arg dbgen.ListResourceAuditEventsBySchemeAndActionParams) ([]dbgen.ResourceAuditEvent, error)
	CountResourceAuditEventsByScheme(ctx context.Context, schemeID uuid.UUID) (int64, error)
	GetScheme(ctx context.Context, id uuid.UUID) (dbgen.Scheme, error)
	GetSchemeMembership(ctx context.Context, arg dbgen.GetSchemeMembershipParams) (dbgen.SchemeMembership, error)
}

type ResourceService struct {
	q ResourceAuditQueries
}

func NewResourceService(q ResourceAuditQueries) *ResourceService {
	return &ResourceService{q: q}
}

func (s *ResourceService) RecordResourceEvent(ctx context.Context, event ResourceEvent) error {
	if s == nil || s.q == nil {
		return nil
	}
	if strings.TrimSpace(event.SchemeID) == "" ||
		strings.TrimSpace(event.OrgID) == "" ||
		strings.TrimSpace(event.ResourceType) == "" ||
		strings.TrimSpace(event.Action) == "" {
		return ErrInvalidResourceEvent
	}

	schemeID, err := uuid.Parse(event.SchemeID)
	if err != nil {
		return ErrInvalidResourceEvent
	}
	orgID, err := uuid.Parse(event.OrgID)
	if err != nil {
		return ErrInvalidResourceEvent
	}

	params := dbgen.CreateResourceAuditEventParams{
		SchemeID:     schemeID,
		OrgID:        orgID,
		ActorUserID:  optionalUUID(event.ActorUserID),
		ActorRole:    strings.TrimSpace(event.ActorRole),
		ResourceType: strings.TrimSpace(event.ResourceType),
		ResourceID:   optionalUUID(event.ResourceID),
		Action:       strings.TrimSpace(event.Action),
		BeforeState:  jsonbValue(event.BeforeState),
		AfterState:   jsonbValue(event.AfterState),
		Metadata:     jsonbValue(event.Metadata),
	}

	_, err = s.q.CreateResourceAuditEvent(ctx, params)
	return err
}

func optionalUUID(value string) pgtype.UUID {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return pgtype.UUID{}
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func jsonbValue(value any) []byte {
	if value == nil {
		return []byte(`{}`)
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return bytes
}

type ResourceEventInfo struct {
	BeforeState  json.RawMessage `json:"before_state,omitempty"`
	AfterState   json.RawMessage `json:"after_state,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	OccurredAt   time.Time       `json:"occurred_at"`
	ID           string          `json:"id"`
	SchemeID     string          `json:"scheme_id"`
	OrgID        string          `json:"org_id"`
	ActorUserID  *string         `json:"actor_user_id"`
	ActorRole    string          `json:"actor_role"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *string         `json:"resource_id"`
	Action       string          `json:"action"`
}

type ListResourceEventsResponse struct {
	Events []ResourceEventInfo `json:"events"`
	Total  int64               `json:"total"`
	Limit  int32               `json:"limit"`
}

func (s *ResourceService) ListSchemeEvents(ctx context.Context, identity auth.Identity, schemeID string, limit int32) (*ListResourceEventsResponse, error) {
	if s == nil || s.q == nil {
		return &ListResourceEventsResponse{Events: []ResourceEventInfo{}, Limit: limit}, nil
	}
	parsedSchemeID, err := uuid.Parse(strings.TrimSpace(schemeID))
	if err != nil {
		return nil, ErrInvalidResourceEvent
	}
	scheme, err := s.q.GetScheme(ctx, parsedSchemeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if auth.IsAdminRole(identity.Role) {
		orgID, parseErr := uuid.Parse(identity.OrgID)
		if parseErr != nil || scheme.OrgID != orgID {
			return nil, ErrForbidden
		}
	} else {
		userID, parseErr := uuid.Parse(identity.UserID)
		if parseErr != nil {
			return nil, ErrForbidden
		}
		membership, membershipErr := s.q.GetSchemeMembership(ctx, dbgen.GetSchemeMembershipParams{
			UserID:   userID,
			SchemeID: parsedSchemeID,
		})
		if membershipErr != nil {
			if errors.Is(membershipErr, pgx.ErrNoRows) {
				return nil, ErrForbidden
			}
			return nil, membershipErr
		}
		if membership.Role != string(auth.RoleTrustee) {
			return nil, ErrForbidden
		}
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.q.ListResourceAuditEventsByScheme(ctx, dbgen.ListResourceAuditEventsBySchemeParams{
		SchemeID: parsedSchemeID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	total, err := s.q.CountResourceAuditEventsByScheme(ctx, parsedSchemeID)
	if err != nil {
		return nil, err
	}
	events := make([]ResourceEventInfo, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapResourceAuditEvent(row))
	}
	return &ListResourceEventsResponse{Events: events, Total: total, Limit: limit}, nil
}

func mapResourceAuditEvent(row dbgen.ResourceAuditEvent) ResourceEventInfo {
	return ResourceEventInfo{
		ID:           row.ID.String(),
		SchemeID:     row.SchemeID.String(),
		OrgID:        row.OrgID.String(),
		ActorUserID:  optionalUUIDString(row.ActorUserID),
		ActorRole:    row.ActorRole,
		ResourceType: row.ResourceType,
		ResourceID:   optionalUUIDString(row.ResourceID),
		Action:       row.Action,
		BeforeState:  rawJSON(row.BeforeState),
		AfterState:   rawJSON(row.AfterState),
		Metadata:     rawJSON(row.Metadata),
		OccurredAt:   row.OccurredAt,
	}
}

func optionalUUIDString(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	str := uuid.UUID(value.Bytes).String()
	return &str
}

func rawJSON(value []byte) json.RawMessage {
	if len(value) == 0 || string(value) == "{}" {
		return nil
	}
	return json.RawMessage(value)
}
