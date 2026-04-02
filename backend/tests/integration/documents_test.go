//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/documents"
)

func newDocumentsHandler(t *testing.T) *documents.Handler {
	t.Helper()
	return documents.NewHandler(documents.NewService(testPool))
}

func TestDocuments_CreateListFilterAndDelete(t *testing.T) {
	h := newDocumentsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "3A")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	createBody, _ := json.Marshal(map[string]any{
		"name":        "Conduct Rules",
		"storage_key": "data:application/pdf;base64,VEVTVA==",
		"file_type":   "pdf",
		"category":    "rules",
		"size_bytes":  4,
	})
	req := httptest.NewRequest(http.MethodPost, "/documents/"+schemeID, bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create document: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[documents.DocumentInfo](t, w)
	if created.Category != "rules" || created.StorageKey == "" {
		t.Fatalf("unexpected created document: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list documents: status=%d body=%s", w.Code, w.Body)
	}
	listResp := decodeSuccess[documents.DashboardResponse](t, w)
	if listResp.Total != 1 || len(listResp.Documents) != 1 {
		t.Fatalf("unexpected documents list: %+v", listResp)
	}

	req = httptest.NewRequest(http.MethodGet, "/documents/"+schemeID+"?category=rules", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("filter documents: status=%d body=%s", w.Code, w.Body)
	}
	filtered := decodeSuccess[documents.DashboardResponse](t, w)
	if filtered.Total != 1 || filtered.Documents[0].Category != "rules" {
		t.Fatalf("unexpected filtered documents: %+v", filtered)
	}

	req = httptest.NewRequest(http.MethodDelete, "/documents/"+schemeID+"/"+created.ID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete document: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodDelete, "/documents/"+schemeID+"/"+created.ID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident delete should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}
