//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/communications"
)

func newCommunicationsHandler(t *testing.T) *communications.Handler {
	t.Helper()
	return communications.NewHandler(communications.NewService(testPool))
}

func TestCommunications_CreateListAndFilter(t *testing.T) {
	h := newCommunicationsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "2B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	adminCreateBody, _ := json.Marshal(map[string]any{
		"title": "Annual General Meeting",
		"body":  "The AGM will be held on 20 April at 18:00.",
		"type":  "agm",
	})
	req := httptest.NewRequest(http.MethodPost, "/communications/"+schemeID, bytes.NewReader(adminCreateBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create notice: status=%d body=%s", w.Code, w.Body)
	}
	adminNotice := decodeSuccess[communications.NoticeInfo](t, w)
	if adminNotice.Type != "agm" || adminNotice.SentByName == nil {
		t.Fatalf("unexpected admin notice: %+v", adminNotice)
	}

	trusteeCreateBody, _ := json.Marshal(map[string]any{
		"title": "Water outage",
		"body":  "Water will be off from 09:00 to 12:00 tomorrow.",
		"type":  "urgent",
	})
	req = httptest.NewRequest(http.MethodPost, "/communications/"+schemeID, bytes.NewReader(trusteeCreateBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w = httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("trustee create notice: status=%d body=%s", w.Code, w.Body)
	}
	trusteeNotice := decodeSuccess[communications.NoticeInfo](t, w)
	if trusteeNotice.Type != "urgent" {
		t.Fatalf("unexpected trustee notice: %+v", trusteeNotice)
	}

	req = httptest.NewRequest(http.MethodGet, "/communications/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident list notices: status=%d body=%s", w.Code, w.Body)
	}
	residentList := decodeSuccess[communications.DashboardResponse](t, w)
	if residentList.Total != 2 || len(residentList.Notices) != 2 {
		t.Fatalf("unexpected resident communications list: %+v", residentList)
	}

	req = httptest.NewRequest(http.MethodGet, "/communications/"+schemeID+"?type=urgent", nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("filter notices: status=%d body=%s", w.Code, w.Body)
	}
	filtered := decodeSuccess[communications.DashboardResponse](t, w)
	if filtered.Total != 1 || filtered.Notices[0].Type != "urgent" {
		t.Fatalf("unexpected filtered notices: %+v", filtered)
	}

	req = httptest.NewRequest(http.MethodPost, "/communications/"+schemeID, bytes.NewReader(adminCreateBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident create should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}
