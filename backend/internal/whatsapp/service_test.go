package whatsapp

import (
	"testing"
	"time"

	"github.com/stratahq/backend/internal/audit"
)

func TestWhatsAppBroadcastCreatedAuditEvent(t *testing.T) {
	event := whatsAppBroadcastCreatedAuditEvent(whatsAppAuditInput{
		SchemeID:       "scheme-1",
		OrgID:          "org-1",
		ActorUserID:    "user-1",
		ActorRole:      "admin",
		BroadcastID:    "broadcast-1",
		Message:        "Hello residents",
		Type:           "general",
		RecipientCount: 12,
		SentAt:         time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
	})

	if event.Action != "whatsapp.broadcast_created" {
		t.Fatalf("action = %q, want whatsapp.broadcast_created", event.Action)
	}
	if event.ResourceType != "whatsapp_broadcast" {
		t.Fatalf("resource_type = %q, want whatsapp_broadcast", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["recipient_count"] != 12 {
		t.Fatalf("after.recipient_count = %v, want 12", after["recipient_count"])
	}
	if event.BeforeState != nil {
		t.Fatal("before state should be nil for create")
	}
}

func TestWhatsAppBroadcastSentAuditEvent(t *testing.T) {
	event := whatsAppBroadcastSentAuditEvent(whatsAppAuditInput{
		SchemeID:       "scheme-1",
		OrgID:          "org-1",
		ActorUserID:    "user-1",
		ActorRole:      "admin",
		BroadcastID:    "broadcast-1",
		Message:        "Hello residents",
		Type:           "general",
		RecipientCount: 12,
		SentAt:         time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
	}, 2)

	if event.Action != "whatsapp.broadcast_sent" {
		t.Fatalf("action = %q, want whatsapp.broadcast_sent", event.Action)
	}
	metadata, ok := event.Metadata.(map[string]any)
	if !ok {
		t.Fatal("metadata should be a map")
	}
	if metadata["send_errors"] != 2 {
		t.Fatalf("metadata.send_errors = %v, want 2", metadata["send_errors"])
	}
}

var _ audit.ResourceEvent
