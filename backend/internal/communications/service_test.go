package communications

import (
	"testing"
	"time"

	"github.com/stratahq/backend/internal/audit"
)

func TestNoticeCreatedAuditEvent(t *testing.T) {
	event := noticeCreatedAuditEvent(noticeAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		NoticeID:    "notice-1",
		Title:       "Monthly Meeting",
		Body:        "The monthly meeting will be held on Friday.",
		Type:        "general",
		SentAt:      time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
	})

	if event.Action != "notice.created" {
		t.Fatalf("action = %q, want notice.created", event.Action)
	}
	if event.ResourceType != "notice" {
		t.Fatalf("resource_type = %q, want notice", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["title"] != "Monthly Meeting" {
		t.Fatalf("after.title = %v, want Monthly Meeting", after["title"])
	}
	if event.BeforeState != nil {
		t.Fatal("before state should be nil for create")
	}
}

var _ audit.ResourceEvent
