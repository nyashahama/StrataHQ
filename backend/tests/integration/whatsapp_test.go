//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/whatsapp"
)

func newWhatsAppHandler(t *testing.T) *whatsapp.Handler {
	t.Helper()
	return whatsapp.NewHandler(whatsapp.NewService(testPool))
}

func TestWhatsAppDashboardAndBroadcast(t *testing.T) {
	h := newWhatsAppHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitResidentID := createUnitRecord(t, schemeID, "4B")
	unitOtherID := createUnitRecord(t, schemeID, "2B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitResidentID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	schemeUUID := mustParseUUID(schemeID)
	residentUnitUUID := mustParseUUID(unitResidentID)
	otherUnitUUID := mustParseUUID(unitOtherID)
	residentUserUUID := mustParseUUID(residentUserID)
	now := time.Now().UTC()

	thread, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         residentUnitUUID,
		ResidentUserID: pgtype.UUID{Bytes: residentUserUUID, Valid: true},
		PhoneNumber:    pgtype.Text{String: "+27715550404", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    1,
		LastActiveAt:   now,
	})
	if err != nil {
		t.Fatalf("create resident whatsapp thread: %v", err)
	}

	if _, err := testQ.CreateWhatsAppMessage(t.Context(), dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderResident,
		Body:                 "hello from resident",
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	}); err != nil {
		t.Fatalf("create resident whatsapp message: %v", err)
	}

	if _, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         otherUnitUUID,
		ResidentUserID: pgtype.UUID{},
		PhoneNumber:    pgtype.Text{},
		Connected:      false,
		ConsentedAt:    pgtype.Timestamptz{},
		UnreadCount:    0,
		LastActiveAt:   now.Add(-48 * time.Hour),
	}); err != nil {
		t.Fatalf("create other whatsapp thread: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/whatsapp/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w := httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident whatsapp dashboard: status=%d body=%s", w.Code, w.Body)
	}

	residentDashboard := decodeSuccess[whatsapp.DashboardResponse](t, w)
	if residentDashboard.ResidentThread == nil {
		t.Fatalf("expected resident thread in dashboard: %+v", residentDashboard)
	}
	if residentDashboard.ResidentThread.UnitIdentifier != "4B" {
		t.Fatalf("unexpected resident unit identifier: %+v", residentDashboard.ResidentThread)
	}

	req = httptest.NewRequest(http.MethodGet, "/whatsapp/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin whatsapp dashboard: status=%d body=%s", w.Code, w.Body)
	}

	adminDashboard := decodeSuccess[whatsapp.DashboardResponse](t, w)
	if adminDashboard.TotalResidents != 2 || len(adminDashboard.Threads) != 2 {
		t.Fatalf("unexpected admin whatsapp dashboard: %+v", adminDashboard)
	}

	createBody, _ := json.Marshal(map[string]any{
		"message": "AGM reminder via WhatsApp",
		"type":    "agm",
	})
	req = httptest.NewRequest(http.MethodPost, "/whatsapp/"+schemeID+"/broadcasts", bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w = httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("trustee create whatsapp broadcast: status=%d body=%s", w.Code, w.Body)
	}

	created := decodeSuccess[whatsapp.BroadcastInfo](t, w)
	if created.Type != "agm" || created.RecipientCount != 1 {
		t.Fatalf("unexpected whatsapp broadcast: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/whatsapp/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident reload whatsapp dashboard: status=%d body=%s", w.Code, w.Body)
	}

	updatedResidentDashboard := decodeSuccess[whatsapp.DashboardResponse](t, w)
	if updatedResidentDashboard.ResidentThread == nil || len(updatedResidentDashboard.ResidentThread.Messages) != 2 {
		t.Fatalf("expected broadcast delivery in resident thread: %+v", updatedResidentDashboard)
	}
	if updatedResidentDashboard.ResidentThread.Messages[1].From != "bot" {
		t.Fatalf("expected bot delivery message: %+v", updatedResidentDashboard.ResidentThread.Messages[1])
	}

	req = httptest.NewRequest(http.MethodPost, "/whatsapp/"+schemeID+"/broadcasts", bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident create broadcast should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}
