package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
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
