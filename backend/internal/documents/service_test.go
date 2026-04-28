package documents

import (
	"testing"

	"github.com/stratahq/backend/internal/audit"
)

func TestDocumentAuditEventForCreate(t *testing.T) {
	event := documentCreatedAuditEvent(documentAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		DocumentID:  "doc-1",
		Name:        "rules.pdf",
		Category:    "rules",
		FileType:    "pdf",
		SizeBytes:   1234,
		StorageKey:  "scheme/doc.pdf",
	})

	if event.Action != "document.uploaded" {
		t.Fatalf("action = %q, want document.uploaded", event.Action)
	}
	if event.ResourceType != "document" {
		t.Fatalf("resource type = %q, want document", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["name"] != "rules.pdf" {
		t.Fatalf("after name = %v, want rules.pdf", after["name"])
	}
}

func TestDocumentAuditEventForDeleteIncludesBeforeState(t *testing.T) {
	event := documentDeletedAuditEvent(documentAuditInput{
		SchemeID:   "scheme-1",
		OrgID:      "org-1",
		DocumentID: "doc-1",
		Name:       "rules.pdf",
		Category:   "rules",
		FileType:   "pdf",
		SizeBytes:  1234,
		StorageKey: "scheme/doc.pdf",
	})

	if event.Action != "document.deleted" {
		t.Fatalf("action = %q, want document.deleted", event.Action)
	}
	if event.BeforeState == nil {
		t.Fatal("before state should be present for delete")
	}
}

var _ audit.ResourceEvent
