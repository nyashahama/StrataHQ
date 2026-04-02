//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/maintenance"
)

func newMaintenanceHandler(t *testing.T) *maintenance.Handler {
	t.Helper()
	return maintenance.NewHandler(maintenance.NewService(testPool))
}

func TestMaintenance_AdminResidentAndTrusteeFlow(t *testing.T) {
	h := newMaintenanceHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "4B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	adminCreateBody, _ := json.Marshal(map[string]any{
		"title":       "Parking gate motor fault",
		"description": "The gate motor stops halfway through opening.",
		"category":    "electrical",
	})
	req := httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID, bytes.NewReader(adminCreateBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create: status=%d body=%s", w.Code, w.Body)
	}
	adminCreated := decodeSuccess[maintenance.RequestInfo](t, w)
	if adminCreated.Status != "open" || adminCreated.UnitID != nil {
		t.Fatalf("unexpected admin-created request: %+v", adminCreated)
	}

	residentCreateBody, _ := json.Marshal(map[string]any{
		"title":       "Kitchen tap leaking",
		"description": "The kitchen mixer tap drips continuously.",
		"category":    "plumbing",
	})
	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID, bytes.NewReader(residentCreateBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("resident create: status=%d body=%s", w.Code, w.Body)
	}
	residentCreated := decodeSuccess[maintenance.RequestInfo](t, w)
	if residentCreated.Status != "pending_approval" || residentCreated.UnitIdentifier == nil || *residentCreated.UnitIdentifier != "4B" {
		t.Fatalf("unexpected resident-created request: %+v", residentCreated)
	}

	req = httptest.NewRequest(http.MethodGet, "/maintenance/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin dashboard: status=%d body=%s", w.Code, w.Body)
	}
	adminDashboard := decodeSuccess[maintenance.DashboardResponse](t, w)
	if len(adminDashboard.Requests) != 2 {
		t.Fatalf("expected 2 maintenance requests, got %d", len(adminDashboard.Requests))
	}
	if adminDashboard.OpenCount != 2 || adminDashboard.PendingApprovalCount != 1 {
		t.Fatalf("unexpected admin dashboard counts: %+v", adminDashboard)
	}

	req = httptest.NewRequest(http.MethodGet, "/maintenance/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident dashboard: status=%d body=%s", w.Code, w.Body)
	}
	residentDashboard := decodeSuccess[maintenance.DashboardResponse](t, w)
	if len(residentDashboard.Requests) != 1 || residentDashboard.Requests[0].ID != residentCreated.ID {
		t.Fatalf("resident should only see own maintenance request, got %+v", residentDashboard.Requests)
	}

	assignBody, _ := json.Marshal(map[string]any{
		"contractor_name":  "Rapid Plumbing Co.",
		"contractor_phone": "021 555 0123",
	})
	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID+"/"+residentCreated.ID+"/assign", bytes.NewReader(assignBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": residentCreated.ID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w = httptest.NewRecorder()
	h.Assign(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee assign: status=%d body=%s", w.Code, w.Body)
	}
	assigned := decodeSuccess[maintenance.RequestInfo](t, w)
	if assigned.Status != "in_progress" || assigned.ContractorName == nil || *assigned.ContractorName != "Rapid Plumbing Co." {
		t.Fatalf("unexpected assigned request: %+v", assigned)
	}

	req = httptest.NewRequest(http.MethodPost, "/maintenance/"+schemeID+"/"+residentCreated.ID+"/resolve", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID, "id": residentCreated.ID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w = httptest.NewRecorder()
	h.Resolve(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("trustee resolve: status=%d body=%s", w.Code, w.Body)
	}
	resolved := decodeSuccess[maintenance.RequestInfo](t, w)
	if resolved.Status != "resolved" || resolved.ResolvedAt == nil {
		t.Fatalf("unexpected resolved request: %+v", resolved)
	}

	req = httptest.NewRequest(http.MethodGet, "/maintenance/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	adminDashboard = decodeSuccess[maintenance.DashboardResponse](t, w)
	if adminDashboard.OpenCount != 1 || adminDashboard.ResolvedThisMonth != 1 {
		t.Fatalf("unexpected post-resolve dashboard counts: %+v", adminDashboard)
	}
}
