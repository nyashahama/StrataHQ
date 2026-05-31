//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/contractors"
	"github.com/stratahq/backend/internal/maintenance"
)

func newContractorsHandler(t *testing.T) *contractors.Handler {
	t.Helper()
	return contractors.NewHandler(contractors.NewService(testPool))
}

func TestContractors_AdminCreatesListsAndResidentDenied(t *testing.T) {
	h := newContractorsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	unitID := createUnitRecord(t, schemeID, "2B")
	residentID := createMemberRecord(t, orgID, schemeID, uniqueEmail(t), "Resident", string(auth.RoleResident), &unitID)

	body, _ := json.Marshal(map[string]any{
		"name":       "SparkPro Electrical",
		"trade":      "electrical",
		"phone":      "+27 21 555 0200",
		"suburb":     "Woodstock",
		"active":     true,
		"vetted":     true,
		"scheme_ids": []string{schemeID},
	})
	req := httptest.NewRequest(http.MethodPost, "/contractors", bytes.NewReader(body))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create contractor: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[contractors.ContractorInfo](t, w)
	if created.Name != "SparkPro Electrical" || len(created.SchemeIDs) != 1 {
		t.Fatalf("unexpected contractor: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/contractors?scheme_id="+schemeID, nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list contractors: status=%d body=%s", w.Code, w.Body)
	}
	listed := decodeSuccess[[]contractors.ContractorInfo](t, w)
	if len(listed) == 0 {
		t.Fatal("expected contractor list")
	}

	req = httptest.NewRequest(http.MethodGet, "/contractors", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident list should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func TestContractors_ReviewResolvedMaintenanceRequest(t *testing.T) {
	ch := newContractorsHandler(t)
	mh := newMaintenanceHandler(t)
	accessToken, _ := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	contractorBody, _ := json.Marshal(map[string]any{
		"name": "AquaFix Plumbing", "trade": "plumbing", "suburb": "Sea Point",
		"phone": "+27 21 555 0199", "active": true, "vetted": true, "scheme_ids": []string{schemeID},
	})
	req := httptest.NewRequest(http.MethodPost, "/contractors", bytes.NewReader(contractorBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	ch.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create contractor: status=%d body=%s", w.Code, w.Body)
	}
	contractor := decodeSuccess[contractors.ContractorInfo](t, w)

	requestBody, _ := json.Marshal(map[string]string{"title": "Leak", "description": "Leaking pipe", "category": "plumbing"})
	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID, bytes.NewReader(requestBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	w = httptest.NewRecorder()
	mh.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create request: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[maintenance.RequestInfo](t, w)

	assignBody, _ := json.Marshal(map[string]string{"contractor_id": contractor.ID})
	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID+"/"+created.ID+"/assign", bytes.NewReader(assignBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	w = httptest.NewRecorder()
	mh.Assign(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("assign request: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID+"/"+created.ID+"/resolve", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": created.ID})
	w = httptest.NewRecorder()
	mh.Resolve(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resolve request: status=%d body=%s", w.Code, w.Body)
	}

	for _, rating := range []int{-1, 0, 6} {
		reviewBody, _ := json.Marshal(map[string]any{
			"scheme_id":              schemeID,
			"maintenance_request_id": created.ID,
			"rating":                 rating,
			"comment":                "Invalid rating should be rejected.",
		})
		req = httptest.NewRequest(http.MethodPost, "/contractors/"+contractor.ID+"/reviews", bytes.NewReader(reviewBody))
		req = withAuthContext(req, accessToken, testJWTSigningKey)
		req = withRouteParams(req, map[string]string{"contractorId": contractor.ID})
		w = httptest.NewRecorder()
		ch.CreateReview(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("create review with rating %d: status=%d body=%s, want 400", rating, w.Code, w.Body)
		}
	}

	reviewBody, _ := json.Marshal(map[string]any{
		"scheme_id":              schemeID,
		"maintenance_request_id": created.ID,
		"rating":                 5,
		"comment":                "Fast and clean.",
	})
	req = httptest.NewRequest(http.MethodPost, "/contractors/"+contractor.ID+"/reviews", bytes.NewReader(reviewBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	req = withRouteParams(req, map[string]string{"contractorId": contractor.ID})
	w = httptest.NewRecorder()
	ch.CreateReview(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create review: status=%d body=%s", w.Code, w.Body)
	}
}
