//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/audit"
	"github.com/stratahq/backend/internal/auth"
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

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("schemeId", schemeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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

func TestAudit_ListSchemeEvents_ReturnsNewestFirst(t *testing.T) {
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	recorder := audit.NewResourceService(testPool.Q)
	for i, action := range []string{"document.uploaded", "scheme.updated", "unit.created"} {
		if err := recorder.RecordResourceEvent(context.Background(), audit.ResourceEvent{
			SchemeID:     schemeID,
			OrgID:        orgID,
			ActorRole:    "admin",
			ResourceType: "document",
			Action:       action,
			AfterState: map[string]any{
				"index": i,
			},
		}); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	h := audit.NewHandler(recorder)
	req := httptest.NewRequest(http.MethodGet, "/audit/schemes/"+schemeID+"/events", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("schemeId", schemeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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
	if len(resp.Data.Events) != 3 {
		t.Fatalf("events len = %d, want 3", len(resp.Data.Events))
	}
	wantOrder := []string{"unit.created", "scheme.updated", "document.uploaded"}
	for i, want := range wantOrder {
		if resp.Data.Events[i].Action != want {
			t.Fatalf("event[%d].action = %q, want %q", i, resp.Data.Events[i].Action, want)
		}
	}
}

func TestAudit_ListSchemeEvents_RejectsResidents(t *testing.T) {
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	_ = accessToken

	// Build a token with resident role for the scheme
	residentToken, err := auth.GenerateAccessToken("00000000-0000-0000-0000-000000000001", orgID, "resident", "http://localhost:3000", "stratahq-api", testJWTSigningKey, 15*time.Minute)
	if err != nil {
		t.Fatalf("generate resident token: %v", err)
	}

	recorder := audit.NewResourceService(testPool.Q)
	h := audit.NewHandler(recorder)
	req := httptest.NewRequest(http.MethodGet, "/audit/schemes/"+schemeID+"/events", nil)
	req = withAuthContext(req, residentToken, testJWTSigningKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("schemeId", schemeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListSchemeEvents(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAudit_ListSchemeEvents_UsesDefaultLimit(t *testing.T) {
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	recorder := audit.NewResourceService(testPool.Q)
	for i := 0; i < 55; i++ {
		if err := recorder.RecordResourceEvent(context.Background(), audit.ResourceEvent{
			SchemeID:     schemeID,
			OrgID:        orgID,
			ActorRole:    "admin",
			ResourceType: "document",
			Action:       "document.uploaded",
			AfterState: map[string]any{
				"index": i,
			},
		}); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}

	h := audit.NewHandler(recorder)
	req := httptest.NewRequest(http.MethodGet, "/audit/schemes/"+schemeID+"/events", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("schemeId", schemeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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
	if resp.Data.Limit != 50 {
		t.Fatalf("limit = %d, want 50", resp.Data.Limit)
	}
	if len(resp.Data.Events) > 50 {
		t.Fatalf("events len = %d, want <= 50", len(resp.Data.Events))
	}
}
