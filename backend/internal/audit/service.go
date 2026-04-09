package audit

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/stratahq/backend/internal/platform/database"
)

type Event struct {
	OccurredAt   time.Time
	StatusCode   int
	ActorUserID  string
	OrgID        string
	ActorRole    string
	Method       string
	Path         string
	RoutePattern string
	IPAddress    string
	UserAgent    string
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

	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
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
