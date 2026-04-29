package audit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
)

type fakeResourceAuditQueries struct {
	params               dbgen.CreateResourceAuditEventParams
	err                  error
	scheme               dbgen.Scheme
	schemeErr            error
	membership           dbgen.SchemeMembership
	membershipErr        error
	events               []dbgen.ResourceAuditEvent
	count                int64
}

func (f *fakeResourceAuditQueries) CreateResourceAuditEvent(_ context.Context, params dbgen.CreateResourceAuditEventParams) (dbgen.ResourceAuditEvent, error) {
	f.params = params
	return dbgen.ResourceAuditEvent{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000999"),
		SchemeID:     params.SchemeID,
		OrgID:        params.OrgID,
		ActorUserID:  params.ActorUserID,
		ActorRole:    params.ActorRole,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		Action:       params.Action,
		BeforeState:  params.BeforeState,
		AfterState:   params.AfterState,
		Metadata:     params.Metadata,
	}, f.err
}

func (f *fakeResourceAuditQueries) ListResourceAuditEventsByScheme(_ context.Context, arg dbgen.ListResourceAuditEventsBySchemeParams) ([]dbgen.ResourceAuditEvent, error) {
	limit := int(arg.Limit)
	if limit <= 0 || limit > len(f.events) {
		limit = len(f.events)
	}
	return f.events[:limit], nil
}

func (f *fakeResourceAuditQueries) ListResourceAuditEventsBySchemeAndAction(context.Context, dbgen.ListResourceAuditEventsBySchemeAndActionParams) ([]dbgen.ResourceAuditEvent, error) {
	return nil, nil
}

func (f *fakeResourceAuditQueries) CountResourceAuditEventsByScheme(context.Context, uuid.UUID) (int64, error) {
	return f.count, nil
}

func (f *fakeResourceAuditQueries) GetScheme(context.Context, uuid.UUID) (dbgen.Scheme, error) {
	return f.scheme, f.schemeErr
}

func (f *fakeResourceAuditQueries) GetSchemeMembership(context.Context, dbgen.GetSchemeMembershipParams) (dbgen.SchemeMembership, error) {
	return f.membership, f.membershipErr
}

func TestResourceRecorderNoopsWhenNil(t *testing.T) {
	var recorder *ResourceService
	if err := recorder.RecordResourceEvent(context.Background(), ResourceEvent{}); err != nil {
		t.Fatalf("nil recorder error = %v, want nil", err)
	}
}

func TestRecordResourceEventSerializesSnapshots(t *testing.T) {
	queries := &fakeResourceAuditQueries{}
	service := NewResourceService(queries)

	event := ResourceEvent{
		SchemeID:     "00000000-0000-0000-0000-000000000001",
		OrgID:        "00000000-0000-0000-0000-000000000002",
		ActorUserID:  "00000000-0000-0000-0000-000000000003",
		ActorRole:    "admin",
		ResourceType: "document",
		ResourceID:   "00000000-0000-0000-0000-000000000004",
		Action:       "document.uploaded",
		BeforeState:  nil,
		AfterState: map[string]any{
			"name": "rules.pdf",
		},
		Metadata: map[string]any{
			"category": "rules",
		},
	}

	if err := service.RecordResourceEvent(context.Background(), event); err != nil {
		t.Fatalf("record resource event: %v", err)
	}

	if queries.params.SchemeID.String() != event.SchemeID {
		t.Fatalf("scheme id = %s, want %s", queries.params.SchemeID, event.SchemeID)
	}
	if !queries.params.ActorUserID.Valid {
		t.Fatal("actor user id should be valid")
	}
	if string(queries.params.BeforeState) != "{}" {
		t.Fatalf("before state = %s, want {}", string(queries.params.BeforeState))
	}
	if len(queries.params.AfterState) == 0 {
		t.Fatal("after state should not be empty")
	}
	var after map[string]string
	if err := json.Unmarshal(queries.params.AfterState, &after); err != nil {
		t.Fatalf("decode after state: %v", err)
	}
	if after["name"] != "rules.pdf" {
		t.Fatalf("after.name = %q, want rules.pdf", after["name"])
	}
}

func TestRecordResourceEventRejectsMissingRequiredFields(t *testing.T) {
	service := NewResourceService(&fakeResourceAuditQueries{})

	err := service.RecordResourceEvent(context.Background(), ResourceEvent{
		SchemeID: "00000000-0000-0000-0000-000000000001",
		OrgID:    "00000000-0000-0000-0000-000000000002",
		Action:   "document.uploaded",
	})
	if err != ErrInvalidResourceEvent {
		t.Fatalf("error = %v, want ErrInvalidResourceEvent", err)
	}
}

func TestListSchemeEventsDefaultsLimitTo50(t *testing.T) {
	schemeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	queries := &fakeResourceAuditQueries{
		scheme: dbgen.Scheme{ID: schemeID, OrgID: orgID},
		events: make([]dbgen.ResourceAuditEvent, 60),
		count:  60,
	}
	service := NewResourceService(queries)

	resp, err := service.ListSchemeEvents(context.Background(), auth.Identity{
		UserID: "00000000-0000-0000-0000-000000000003",
		OrgID:  orgID.String(),
		Role:   "admin",
	}, schemeID.String(), 0)
	if err != nil {
		t.Fatalf("list scheme events: %v", err)
	}
	if resp.Limit != 50 {
		t.Fatalf("limit = %d, want 50", resp.Limit)
	}
	if len(resp.Events) != 50 {
		t.Fatalf("events len = %d, want 50", len(resp.Events))
	}
}

func TestListSchemeEventsRejectsResidents(t *testing.T) {
	schemeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	queries := &fakeResourceAuditQueries{
		scheme: dbgen.Scheme{ID: schemeID, OrgID: orgID},
		membership: dbgen.SchemeMembership{
			UserID:   userID,
			SchemeID: schemeID,
			Role:     "resident",
		},
	}
	service := NewResourceService(queries)

	_, err := service.ListSchemeEvents(context.Background(), auth.Identity{
		UserID: userID.String(),
		Role:   "resident",
	}, schemeID.String(), 50)
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestListSchemeEventsRejectsNonMatchingOrg(t *testing.T) {
	schemeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	queries := &fakeResourceAuditQueries{
		scheme: dbgen.Scheme{ID: schemeID, OrgID: orgID},
	}
	service := NewResourceService(queries)

	_, err := service.ListSchemeEvents(context.Background(), auth.Identity{
		UserID: "00000000-0000-0000-0000-000000000003",
		OrgID:  "00000000-0000-0000-0000-000000000099",
		Role:   "admin",
	}, schemeID.String(), 50)
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}
