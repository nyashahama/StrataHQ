//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/integrations"
)

func newIntegrationsHandler(t *testing.T) *integrations.Handler {
	t.Helper()
	return integrations.NewHandler(integrations.NewService(testPool))
}

func TestOpenAPI_ReadsSchemeLevyAndFinancialDataWithAPIKey(t *testing.T) {
	h := newIntegrationsHandler(t)
	accessToken, _ := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	body, _ := json.Marshal(map[string]any{
		"name":       "QuickBooks Reader",
		"scheme_ids": []string{schemeID},
	})
	req := httptest.NewRequest(http.MethodPost, "/integrations/api-clients", bytes.NewReader(body))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.CreateAPIClient(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create client: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[integrations.APIClientCreateResponse](t, w)

	req = httptest.NewRequest(http.MethodGet, "/open/v1/schemes", nil)
	req.Header.Set("Authorization", "Bearer "+created.APIKey)
	w = httptest.NewRecorder()
	h.OpenRoutes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("open schemes: status=%d body=%s", w.Code, w.Body)
	}
}

func TestIntegrations_AdminCreatesAndRevokesAPIClient(t *testing.T) {
	h := newIntegrationsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	unitID := createUnitRecord(t, schemeID, "1A")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	body, _ := json.Marshal(map[string]any{
		"name":       "Pastel Export",
		"scheme_ids": []string{schemeID},
		"scopes":     []string{"read:schemes", "read:levies", "read:financials"},
	})
	req := httptest.NewRequest(http.MethodPost, "/integrations/api-clients", bytes.NewReader(body))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.CreateAPIClient(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create client: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[integrations.APIClientCreateResponse](t, w)
	if created.APIKey == "" || created.KeyPrefix == "" || len(created.SchemeIDs) != 1 {
		t.Fatalf("unexpected created client: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/integrations/api-clients", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.ListAPIClients(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list clients: status=%d body=%s", w.Code, w.Body)
	}
	listed := decodeSuccess[[]integrations.APIClientInfo](t, w)
	if len(listed) != 1 || listed[0].KeyPrefix != created.KeyPrefix {
		t.Fatalf("unexpected clients: %+v", listed)
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/integrations/api-clients/"+created.ID, nil)
	revokeReq = withAuthContext(revokeReq, accessToken, testJWTSigningKey)
	revokeReq = withRouteParams(revokeReq, map[string]string{"clientId": created.ID})
	w = httptest.NewRecorder()
	h.RevokeAPIClient(w, revokeReq)
	if w.Code != http.StatusOK {
		t.Fatalf("revoke client: status=%d body=%s", w.Code, w.Body)
	}

	req = httptest.NewRequest(http.MethodPost, "/integrations/api-clients", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.CreateAPIClient(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident create should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}
