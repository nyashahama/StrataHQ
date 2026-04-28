//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/audit"
)

func TestAudit_ListSchemeResourceEvents(t *testing.T) {
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	recorder := audit.NewResourceService(testPool.Q)
	if err := recorder.RecordResourceEvent(context.Background(), audit.ResourceEvent{
		SchemeID:     schemeID,
		OrgID:        orgID,
		ActorRole:    "admin",
		ResourceType: "document",
		Action:       "document.uploaded",
		AfterState: map[string]any{
			"name": "rules.pdf",
		},
	}); err != nil {
		t.Fatalf("seed resource audit event: %v", err)
	}

	h := audit.NewHandler(recorder)
	req := httptest.NewRequest(http.MethodGet, "/audit/schemes/"+schemeID+"/events", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()

	h.ListSchemeEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list audit events: status=%d body=%s", w.Code, w.Body)
	}
	var resp struct {
		Data audit.ListResourceEventsResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data.Events) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.Data.Events))
	}
	if resp.Data.Events[0].Action != "document.uploaded" {
		t.Fatalf("action = %q, want document.uploaded", resp.Data.Events[0].Action)
	}
}
