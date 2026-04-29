package scheme

import (
	"testing"

	"github.com/stratahq/backend/internal/audit"
)

func TestSchemeCreatedAuditEvent(t *testing.T) {
	event := schemeCreatedAuditEvent(schemeAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		Name:        "Test Scheme",
		Address:     "123 Main St",
		UnitCount:   10,
	})

	if event.Action != "scheme.created" {
		t.Fatalf("action = %q, want scheme.created", event.Action)
	}
	if event.ResourceType != "scheme" {
		t.Fatalf("resource_type = %q, want scheme", event.ResourceType)
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["name"] != "Test Scheme" {
		t.Fatalf("after.name = %v, want Test Scheme", after["name"])
	}
	if event.BeforeState != nil {
		t.Fatal("before state should be nil for create")
	}
}

func TestSchemeUpdatedAuditEvent(t *testing.T) {
	event := schemeUpdatedAuditEvent(schemeAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		Name:        "Updated Scheme",
		Address:     "456 Oak St",
		UnitCount:   12,
	}, "Old Scheme", "123 Main St", 10)

	if event.Action != "scheme.updated" {
		t.Fatalf("action = %q, want scheme.updated", event.Action)
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["name"] != "Old Scheme" {
		t.Fatalf("before.name = %v, want Old Scheme", before["name"])
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["name"] != "Updated Scheme" {
		t.Fatalf("after.name = %v, want Updated Scheme", after["name"])
	}
}

func TestSchemeDeletedAuditEvent(t *testing.T) {
	event := schemeDeletedAuditEvent(schemeAuditInput{
		SchemeID:    "scheme-1",
		OrgID:       "org-1",
		ActorUserID: "user-1",
		ActorRole:   "admin",
		Name:        "Deleted Scheme",
		Address:     "123 Main St",
		UnitCount:   10,
	})

	if event.Action != "scheme.deleted" {
		t.Fatalf("action = %q, want scheme.deleted", event.Action)
	}
	if event.AfterState != nil {
		t.Fatal("after state should be nil for delete")
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["name"] != "Deleted Scheme" {
		t.Fatalf("before.name = %v, want Deleted Scheme", before["name"])
	}
}

func TestUnitCreatedAuditEvent(t *testing.T) {
	event := unitCreatedAuditEvent(unitAuditInput{
		SchemeID:        "scheme-1",
		OrgID:           "org-1",
		ActorUserID:     "user-1",
		ActorRole:       "admin",
		UnitID:          "unit-1",
		Identifier:      "A101",
		OwnerName:       "John Doe",
		Floor:           1,
		SectionValuePct: 25.0,
	})

	if event.Action != "unit.created" {
		t.Fatalf("action = %q, want unit.created", event.Action)
	}
	if event.ResourceType != "unit" {
		t.Fatalf("resource_type = %q, want unit", event.ResourceType)
	}
}

func TestUnitUpdatedAuditEvent(t *testing.T) {
	event := unitUpdatedAuditEvent(unitAuditInput{
		SchemeID:        "scheme-1",
		OrgID:           "org-1",
		ActorUserID:     "user-1",
		ActorRole:       "admin",
		UnitID:          "unit-1",
		Identifier:      "A102",
		OwnerName:       "Jane Doe",
		Floor:           2,
		SectionValuePct: 30.0,
	}, "A101", "John Doe", 1, 25.0)

	if event.Action != "unit.updated" {
		t.Fatalf("action = %q, want unit.updated", event.Action)
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["identifier"] != "A101" {
		t.Fatalf("before.identifier = %v, want A101", before["identifier"])
	}
}

func TestMemberUpdatedAuditEvent(t *testing.T) {
	event := memberUpdatedAuditEvent(memberAuditInput{
		SchemeID:     "scheme-1",
		OrgID:        "org-1",
		ActorUserID:  "user-1",
		ActorRole:    "admin",
		UserID:       "member-1",
		Role:         "resident",
		UnitID:       "unit-1",
		BeforeRole:   "trustee",
		BeforeUnitID: "",
	})

	if event.Action != "member.updated" {
		t.Fatalf("action = %q, want member.updated", event.Action)
	}
	if event.ResourceType != "member" {
		t.Fatalf("resource_type = %q, want member", event.ResourceType)
	}
	before, ok := event.BeforeState.(map[string]any)
	if !ok {
		t.Fatal("before state should be a map")
	}
	if before["role"] != "trustee" {
		t.Fatalf("before.role = %v, want trustee", before["role"])
	}
	if _, exists := before["unit_id"]; exists {
		t.Fatal("before state should not have unit_id when empty")
	}
	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if after["unit_id"] != "unit-1" {
		t.Fatalf("after.unit_id = %v, want unit-1", after["unit_id"])
	}
}

func TestMemberUpdatedAuditEventOmitsEmptyUnitID(t *testing.T) {
	event := memberUpdatedAuditEvent(memberAuditInput{
		SchemeID:     "scheme-1",
		OrgID:        "org-1",
		ActorUserID:  "user-1",
		ActorRole:    "admin",
		UserID:       "member-1",
		Role:         "trustee",
		UnitID:       "",
		BeforeRole:   "resident",
		BeforeUnitID: "unit-1",
	})

	after, ok := event.AfterState.(map[string]any)
	if !ok {
		t.Fatal("after state should be a map")
	}
	if _, exists := after["unit_id"]; exists {
		t.Fatal("after state should not have unit_id when empty")
	}
}

var _ audit.ResourceEvent
