package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	KindCollectionReminderEmail    = "collection_reminder_email"
	KindCollectionReminderWhatsApp = "collection_reminder_whatsapp"
	KindBankStatementImport        = "bank_statement_import"
)

var (
	ErrUnknownKind   = errors.New("unknown job kind")
	ErrBadPayload    = errors.New("bad job payload")
	ErrNonRetryable  = errors.New("non-retryable job error")
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

type Handler interface {
	Handle(ctx context.Context, payload json.RawMessage) error
}

type HandlerFunc func(ctx context.Context, payload json.RawMessage) error

func (fn HandlerFunc) Handle(ctx context.Context, payload json.RawMessage) error {
	return fn(ctx, payload)
}

type Registry map[string]Handler

type EnqueueInput struct {
	Kind           string
	Payload        any
	IdempotencyKey string
	MaxAttempts    int32
	RunAfter       time.Time
}

type CollectionReminderEmailPayload struct {
	CollectionEventID uuid.UUID `json:"collectionEventId"`
	To                string    `json:"to"`
	Subject           string    `json:"subject"`
	HTMLBody          string    `json:"htmlBody"`
}

type CollectionReminderWhatsAppPayload struct {
	CollectionEventID uuid.UUID `json:"collectionEventId"`
	To                string    `json:"to"`
	Body              string    `json:"body"`
}
