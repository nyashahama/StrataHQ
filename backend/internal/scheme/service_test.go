package scheme

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

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

func TestComputeHealthScorePropagatesFactorError(t *testing.T) {
	t.Helper()

	schemeID := uuid.New()
	factorErr := errors.New("reserve fund query failed")

	svc := &Service{
		healthFactorFns: []func(context.Context, uuid.UUID) (int, error){
			func(_ context.Context, _ uuid.UUID) (int, error) { return 100, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 100, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 100, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 0, factorErr },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 100, nil },
		},
	}

	score, breakdown, err := svc.computeHealthScore(context.Background(), schemeID)
	if !errors.Is(err, factorErr) {
		t.Fatalf("expected reserve fund factor error, got score=%d breakdown=%+v err=%v", score, breakdown, err)
	}
	if score != 0 {
		t.Fatalf("expected zero score on factor error, got %d", score)
	}
	if breakdown != nil {
		t.Fatalf("expected nil breakdown on factor error, got %+v", breakdown)
	}
}

func TestComputeHealthScoreAggregatesFactors(t *testing.T) {
	t.Helper()

	schemeID := uuid.New()
	svc := &Service{
		healthFactorFns: []func(context.Context, uuid.UUID) (int, error){
			func(_ context.Context, _ uuid.UUID) (int, error) { return 100, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 80, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 60, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 40, nil },
			func(_ context.Context, _ uuid.UUID) (int, error) { return 20, nil },
		},
	}

	score, breakdown, err := svc.computeHealthScore(context.Background(), schemeID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := int(100*0.35 + 80*0.25 + 60*0.20 + 40*0.15 + 20*0.05)
	if score != want {
		t.Fatalf("expected weighted score %d, got %d", want, score)
	}
	if got := breakdown["levy_collection"]; got != 100 {
		t.Fatalf("expected levy_collection 100, got %d", got)
	}
	if got := breakdown["agm_recency"]; got != 20 {
		t.Fatalf("expected agm_recency 20, got %d", got)
	}
}
