//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/stratahq/backend/internal/scheme"
)

func newSchemeHandler(t *testing.T) *scheme.Handler {
	t.Helper()
	return scheme.NewHandler(scheme.NewService(testPool))
}

func withRouteParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestScheme_AdminCoreFlow(t *testing.T) {
	h := newSchemeHandler(t)
	accessToken, _ := setupAgent(t)
	firstSchemeID := setupScheme(t, accessToken)

	req := httptest.NewRequest(http.MethodGet, "/schemes", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list schemes: status=%d body=%s", w.Code, w.Body)
	}
	listResp := decodeSuccess[[]scheme.SchemeSummary](t, w)
	if len(listResp) != 1 {
		t.Fatalf("expected 1 seeded scheme, got %d", len(listResp))
	}
	if listResp[0].Role != "admin" {
		t.Fatalf("expected admin role in summary, got %q", listResp[0].Role)
	}

	createBody, _ := json.Marshal(map[string]any{
		"name":       "Bluewater Gardens",
		"address":    "22 Beach Road",
		"unit_count": 18,
	})
	req = httptest.NewRequest(http.MethodPost, "/schemes", bytes.NewReader(createBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create scheme: status=%d body=%s", w.Code, w.Body)
	}
	createdScheme := decodeSuccess[scheme.SchemeSummary](t, w)
	if createdScheme.Name != "Bluewater Gardens" {
		t.Fatalf("created scheme name=%q", createdScheme.Name)
	}

	updateBody, _ := json.Marshal(map[string]any{
		"name":       "Bluewater Gardens North",
		"address":    "24 Beach Road",
		"unit_count": 20,
	})
	req = httptest.NewRequest(http.MethodPut, "/schemes/"+createdScheme.ID, bytes.NewReader(updateBody))
	req = withRouteParams(req, map[string]string{"id": createdScheme.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update scheme: status=%d body=%s", w.Code, w.Body)
	}
	updatedScheme := decodeSuccess[scheme.SchemeSummary](t, w)
	if updatedScheme.Name != "Bluewater Gardens North" || updatedScheme.UnitCount != 20 {
		t.Fatalf("unexpected updated scheme: %+v", updatedScheme)
	}

	unitBody, _ := json.Marshal(map[string]any{
		"identifier":        "1A",
		"owner_name":        "A. Adams",
		"floor":             1,
		"section_value_bps": 625,
	})
	req = httptest.NewRequest(http.MethodPost, "/schemes/"+firstSchemeID+"/units", bytes.NewReader(unitBody))
	req = withRouteParams(req, map[string]string{"id": firstSchemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.CreateUnit(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create unit: status=%d body=%s", w.Code, w.Body)
	}
	createdUnit := decodeSuccess[scheme.UnitInfo](t, w)
	if createdUnit.Identifier != "1A" {
		t.Fatalf("created unit identifier=%q", createdUnit.Identifier)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/schemes/"+firstSchemeID, nil)
	detailReq = withRouteParams(detailReq, map[string]string{"id": firstSchemeID})
	detailReq = withAuthContext(detailReq, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Get(w, detailReq)
	if w.Code != http.StatusOK {
		t.Fatalf("get scheme: status=%d body=%s", w.Code, w.Body)
	}
	detail := decodeSuccess[scheme.SchemeDetail](t, w)
	if len(detail.Units) != 1 {
		t.Fatalf("expected 1 unit in scheme detail, got %d", len(detail.Units))
	}

	unitUpdateBody, _ := json.Marshal(map[string]any{
		"identifier":        "1B",
		"owner_name":        "A. Adams",
		"floor":             2,
		"section_value_bps": 650,
	})
	req = httptest.NewRequest(http.MethodPut, "/schemes/"+firstSchemeID+"/units/"+createdUnit.ID, bytes.NewReader(unitUpdateBody))
	req = withRouteParams(req, map[string]string{"id": firstSchemeID, "unitId": createdUnit.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateUnit(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update unit: status=%d body=%s", w.Code, w.Body)
	}
	updatedUnit := decodeSuccess[scheme.UnitInfo](t, w)
	if updatedUnit.Identifier != "1B" || updatedUnit.Floor != 2 {
		t.Fatalf("unexpected updated unit: %+v", updatedUnit)
	}

	req = httptest.NewRequest(http.MethodDelete, "/schemes/"+createdScheme.ID, nil)
	req = withRouteParams(req, map[string]string{"id": createdScheme.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete scheme: status=%d body=%s", w.Code, w.Body)
	}
}
