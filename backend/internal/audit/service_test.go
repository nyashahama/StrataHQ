package audit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
)

type fakeResourceAuditQueries struct {
	params dbgen.CreateResourceAuditEventParams
	err    error
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

func (f *fakeResourceAuditQueries) ListResourceAuditEventsByScheme(context.Context, dbgen.ListResourceAuditEventsBySchemeParams) ([]dbgen.ResourceAuditEvent, error) {
	return nil, nil
}

func (f *fakeResourceAuditQueries) ListResourceAuditEventsBySchemeAndAction(context.Context, dbgen.ListResourceAuditEventsBySchemeAndActionParams) ([]dbgen.ResourceAuditEvent, error) {
	return nil, nil
}

func (f *fakeResourceAuditQueries) CountResourceAuditEventsByScheme(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

func (f *fakeResourceAuditQueries) GetScheme(context.Context, uuid.UUID) (dbgen.Scheme, error) {
	return dbgen.Scheme{}, nil
}

func (f *fakeResourceAuditQueries) GetSchemeMembership(context.Context, dbgen.GetSchemeMembershipParams) (dbgen.SchemeMembership, error) {
	return dbgen.SchemeMembership{}, nil
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
