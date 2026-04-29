package maintenance

import (
	"testing"

	"github.com/stratahq/backend/internal/audit"
)

func ptr(s string) *string {
	return &s
}

func TestMaintenanceRequestCreatedAuditEvent(t *testing.T) {
	event := maintenanceRequestCreatedAuditEvent(maintenanceAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		RequestID:   "req-1",
		Title:       "Leaky faucet",
		Description: "Kitchen faucet is leaking",
		Category:    "plumbing",
		Status:      "open",
		SlaHours:    48,
	})

	if event.Action != "maintenance.request_created" {
		t.Fatalf("action = %q, want maintenance.request_created", event.Action)
	}
	if event.ResourceType != "maintenance_request" {
		t.Fatalf("resource_type = %q, want maintenance_request", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["title"] != "Leaky faucet" {
		t.Fatalf("after.title = %v, want Leaky faucet", after["title"])
	}
	if event.BeforeState != nil {
		t.Fatal("before state should be nil for create")
	}
}

func TestMaintenanceRequestAssignedAuditEvent(t *testing.T) {
	event := maintenanceRequestAssignedAuditEvent(maintenanceAuditInput{
		SchemeID:       "scheme-1",
		OrgID:          "org-1",
		ActorUserID:    "user-1",
		ActorRole:      "admin",
		RequestID:      "req-1",
		Title:          "Leaky faucet",
		Status:         "open",
		ContractorName: ptr("Plumber Joe"),
	}, "", nil)

	if event.Action != "maintenance.request_assigned" {
		t.Fatalf("action = %q, want maintenance.request_assigned", event.Action)
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["contractor_name"] != "" {
		t.Fatalf("before.contractor_name = %v, want empty", before["contractor_name"])
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["contractor_name"] != "Plumber Joe" {
		t.Fatalf("after.contractor_name = %v, want Plumber Joe", after["contractor_name"])
	}
}

func TestMaintenanceRequestResolvedAuditEvent(t *testing.T) {
	event := maintenanceRequestResolvedAuditEvent(maintenanceAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		RequestID:   "req-1",
		Title:       "Leaky faucet",
		Status:      "resolved",
	}, "open")

	if event.Action != "maintenance.request_resolved" {
		t.Fatalf("action = %q, want maintenance.request_resolved", event.Action)
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["status"] != "open" {
		t.Fatalf("before.status = %v, want open", before["status"])
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["status"] != "resolved" {
		t.Fatalf("after.status = %v, want resolved", after["status"])
	}
}

var _ audit.ResourceEvent
