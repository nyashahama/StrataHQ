//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/whatsapp"
)

type scriptedWhatsAppSender struct {
	failOnNth int
	failErr   error
	calls     int
}

func (s *scriptedWhatsAppSender) SendWhatsAppMessage(to, body string) error {
	s.calls++
	if s.failOnNth > 0 && s.calls == s.failOnNth {
		return s.failErr
	}
	return nil
}

func newWhatsAppHandlerWithSender(t *testing.T, sender whatsapp.MessageSender) *whatsapp.Handler {
	t.Helper()
	service := whatsapp.NewService(testPool, sender, slog.Default())
	return whatsapp.NewHandler(service)
}

func newWhatsAppHandler(t *testing.T) *whatsapp.Handler {
	return newWhatsAppHandlerWithSender(t, whatsapp.NewNoOpSender())
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
	if created.DeliveredRecipientCount != 1 || created.FailedRecipientCount != 0 {
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

func TestWhatsAppBroadcastReportsFailedRecipients(t *testing.T) {
	h := newWhatsAppHandlerWithSender(t, &scriptedWhatsAppSender{
		failOnNth: 1,
		failErr:   errors.New("whatsapp provider unavailable"),
	})

	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitResidentID := createUnitRecord(t, schemeID, "7A")
	unitOtherID := createUnitRecord(t, schemeID, "9B")
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	schemeUUID := mustParseUUID(schemeID)
	residentUnitUUID := mustParseUUID(unitResidentID)
	otherUnitUUID := mustParseUUID(unitOtherID)
	now := time.Now().UTC()

	if _, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         residentUnitUUID,
		ResidentUserID: pgtype.UUID{},
		PhoneNumber:    pgtype.Text{String: "+27715550404", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   now,
	}); err != nil {
		t.Fatalf("create connected thread 1: %v", err)
	}

	if _, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         otherUnitUUID,
		ResidentUserID: pgtype.UUID{},
		PhoneNumber:    pgtype.Text{String: "+27715550405", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("create connected thread 2: %v", err)
	}

	reqBody, _ := json.Marshal(map[string]any{
		"message": "AGM reminder with provider failure",
		"type":    "agm",
	})
	req := httptest.NewRequest(http.MethodPost, "/whatsapp/"+schemeID+"/broadcasts", bytes.NewReader(reqBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	w := httptest.NewRecorder()
	h.CreateBroadcast(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("trustee create whatsapp broadcast with failed sends: status=%d body=%s", w.Code, w.Body)
	}

	created := decodeSuccess[whatsapp.BroadcastInfo](t, w)
	if created.RecipientCount != 2 {
		t.Fatalf("expected recipient count 2, got %d: %+v", created.RecipientCount, created)
	}
	if created.DeliveredRecipientCount != 1 {
		t.Fatalf("expected 1 delivered recipient, got %d: %+v", created.DeliveredRecipientCount, created)
	}
	if created.FailedRecipientCount != 1 {
		t.Fatalf("expected 1 failed recipient, got %d: %+v", created.FailedRecipientCount, created)
	}
}

func newWhatsAppWebhookHandler(t *testing.T) *whatsapp.WebhookHandler {
	t.Helper()
	return newWhatsAppWebhookHandlerWithSender(t, whatsapp.NewNoOpSender())
}

func newWhatsAppWebhookHandlerWithSender(t *testing.T, sender whatsapp.MessageSender) *whatsapp.WebhookHandler {
	t.Helper()
	svc := whatsapp.NewService(testPool, whatsapp.NewNoOpSender(), slog.Default())
	bot := whatsapp.NewBot(testPool)
	h := whatsapp.NewWebhookHandler(testPool, sender, bot, svc, slog.Default(), "twilio-token")
	h.SetSkipSigVerify(true)
	return h
}

func TestWhatsAppWebhookCreatesMaintenanceTicketWithMedia(t *testing.T) {
	h := newWhatsAppWebhookHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "5A")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	schemeUUID := mustParseUUID(schemeID)
	residentUnitUUID := mustParseUUID(unitID)
	residentUserUUID := mustParseUUID(residentUserID)
	now := time.Now().UTC()

	if _, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         residentUnitUUID,
		ResidentUserID: pgtype.UUID{Bytes: residentUserUUID, Valid: true},
		PhoneNumber:    pgtype.Text{String: "+27715550404", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   now,
	}); err != nil {
		t.Fatalf("create connected thread: %v", err)
	}

	threads, err := testQ.GetConnectedWhatsAppThreadByPhone(t.Context(), pgtype.Text{String: "+27715550404", Valid: true})
	if err != nil || len(threads) == 0 {
		t.Fatalf("find thread by phone: err=%v count=%d", err, len(threads))
	}
	threadID := threads[0].ID

	form := url.Values{}
	form.Set("From", "whatsapp:+27715550404")
	form.Set("Body", "leaking tap in bathroom")
	form.Set("NumMedia", "1")
	form.Set("MediaUrl0", "https://api.twilio.com/media/ME123")
	form.Set("MediaContentType0", "image/jpeg")
	form.Set("MediaSid0", "ME123")

	r := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhooks", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Inbound(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("webhook: status=%d body=%s", w.Code, w.Body)
	}

	time.Sleep(200 * time.Millisecond)

	requests, err := testQ.ListMaintenanceRequestsByScheme(t.Context(), schemeUUID)
	if err != nil {
		t.Fatalf("list maintenance requests: %v", err)
	}
	var pending *dbgen.MaintenanceRequest
	for i := range requests {
		if requests[i].Status == dbgen.MaintenanceStatusPendingApproval {
			pending = &requests[i]
			break
		}
	}
	if pending == nil {
		t.Fatalf("expected pending maintenance request, got %d total requests", len(requests))
	}

	messages, err := testQ.ListWhatsAppMessagesByThread(t.Context(), threadID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	var linkedMsg *dbgen.WhatsappMessage
	for i := range messages {
		msg := &messages[i]
		if msg.MaintenanceRequestID.Valid {
			linkedMsg = msg
			break
		}
	}
	if linkedMsg == nil {
		t.Fatalf("expected message with maintenance_request_id, got %d messages", len(messages))
	}

	mediaCount, err := testQ.CountWhatsAppMessageMediaByMessage(t.Context(), linkedMsg.ID)
	if err != nil {
		t.Fatalf("count media: %v", err)
	}
	if mediaCount != 1 {
		t.Fatalf("expected 1 media row, got %d", mediaCount)
	}

	intakes, err := testQ.ListWhatsAppMaintenanceIntakesByScheme(t.Context(), schemeUUID)
	if err != nil {
		t.Fatalf("list intakes: %v", err)
	}
	var ticketIntake *dbgen.ListWhatsAppMaintenanceIntakesBySchemeRow
	for i := range intakes {
		if intakes[i].Status == "ticket_created" {
			ticketIntake = &intakes[i]
			break
		}
	}
	if ticketIntake == nil {
		t.Fatalf("expected intake with ticket_created status, got %d intakes", len(intakes))
	}
}

func TestWhatsAppWebhookDoesNotPersistBotReplyWhenProviderFails(t *testing.T) {
	h := newWhatsAppWebhookHandlerWithSender(t, &scriptedWhatsAppSender{
		failOnNth: 1,
		failErr:   errors.New("whatsapp provider unavailable"),
	})

	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "6B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitID)

	schemeUUID := mustParseUUID(schemeID)
	residentUnitUUID := mustParseUUID(unitID)
	residentUserUUID := mustParseUUID(residentUserID)
	now := time.Now().UTC()

	if _, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         residentUnitUUID,
		ResidentUserID: pgtype.UUID{Bytes: residentUserUUID, Valid: true},
		PhoneNumber:    pgtype.Text{String: "+27715550408", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   now,
	}); err != nil {
		t.Fatalf("create connected thread: %v", err)
	}

	threads, err := testQ.GetConnectedWhatsAppThreadByPhone(t.Context(), pgtype.Text{String: "+27715550408", Valid: true})
	if err != nil || len(threads) == 0 {
		t.Fatalf("find thread by phone: err=%v count=%d", err, len(threads))
	}
	threadID := threads[0].ID

	form := url.Values{}
	form.Set("From", "whatsapp:+27715550408")
	form.Set("Body", "how are the rates")

	r := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhooks", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Inbound(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("webhook: status=%d body=%s", w.Code, w.Body)
	}

	time.Sleep(200 * time.Millisecond)

	messages, err := testQ.ListWhatsAppMessagesByThread(t.Context(), threadID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected only resident inbound message when send fails, got %d messages", len(messages))
	}
	if messages[0].Sender != dbgen.WhatsappMessageSenderResident {
		t.Fatalf("expected inbound message sender resident, got %q", messages[0].Sender)
	}
}

func TestWhatsAppMaintenanceInboxManualCreateAndDismiss(t *testing.T) {
	h := newWhatsAppHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitResidentID := createUnitRecord(t, schemeID, "3C")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident User", string(auth.RoleResident), &unitResidentID)
	trusteeEmail := uniqueEmail(t)
	trusteeUserID := createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee User", string(auth.RoleTrustee), nil)

	schemeUUID := mustParseUUID(schemeID)
	residentUnitUUID := mustParseUUID(unitResidentID)
	residentUserUUID := mustParseUUID(residentUserID)
	now := time.Now().UTC()

	thread, err := testQ.CreateWhatsAppThread(t.Context(), dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         residentUnitUUID,
		ResidentUserID: pgtype.UUID{Bytes: residentUserUUID, Valid: true},
		PhoneNumber:    pgtype.Text{String: "+27715550405", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true},
		UnreadCount:    1,
		LastActiveAt:   now,
	})
	if err != nil {
		t.Fatalf("create resident whatsapp thread: %v", err)
	}

	msg, err := testQ.CreateWhatsAppMessage(t.Context(), dbgen.CreateWhatsAppMessageParams{
		ThreadID:             thread.ID,
		Sender:               dbgen.WhatsappMessageSenderResident,
		Body:                 "my tap is leaking in the kitchen",
		MaintenanceRequestID: pgtype.UUID{},
		NoticeID:             pgtype.UUID{},
	})
	if err != nil {
		t.Fatalf("create resident whatsapp message: %v", err)
	}
	msgID := msg.ID.String()

	adminDashboardReq := httptest.NewRequest(http.MethodGet, "/whatsapp/"+schemeID, nil)
	adminDashboardReq = withRouteParams(adminDashboardReq, map[string]string{"schemeId": schemeID})
	adminDashboardReq = withAuthContext(adminDashboardReq, accessToken, testJWTSigningKey)
	adminW := httptest.NewRecorder()
	h.Dashboard(adminW, adminDashboardReq)
	if adminW.Code != http.StatusOK {
		t.Fatalf("admin dashboard: status=%d body=%s", adminW.Code, adminW.Body)
	}

	adminDashboard := decodeSuccess[whatsapp.DashboardResponse](t, adminW)
	if len(adminDashboard.Threads) != 1 {
		t.Fatalf("expected 1 thread in admin dashboard, got %d", len(adminDashboard.Threads))
	}

	createBody, _ := json.Marshal(map[string]string{
		"title":       "Fix leaking tap",
		"description": "Kitchen tap is leaking water continuously",
		"category":    "plumbing",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/whatsapp/"+schemeID+"/messages/"+msgID+"/maintenance-request", bytes.NewReader(createBody))
	createReq = withRouteParams(createReq, map[string]string{"schemeId": schemeID, "messageId": msgID})
	createReq = createReq.WithContext(auth.ContextWithIdentity(createReq.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	createW := httptest.NewRecorder()
	h.CreateMaintenanceFromMessage(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("trustee create maintenance from message: status=%d body=%s", createW.Code, createW.Body)
	}

	created := decodeSuccess[whatsapp.MaintenanceIntakeInfo](t, createW)
	if created.Status != "ticket_created" {
		t.Fatalf("expected ticket_created status, got %q", created.Status)
	}
	if created.Category != "plumbing" {
		t.Fatalf("expected plumbing category, got %q", created.Category)
	}
	if created.MaintenanceRequestID == nil {
		t.Fatalf("expected maintenance_request_id to be set: %+v", created)
	}

	adminDashboardReq = httptest.NewRequest(http.MethodGet, "/whatsapp/"+schemeID, nil)
	adminDashboardReq = withRouteParams(adminDashboardReq, map[string]string{"schemeId": schemeID})
	adminDashboardReq = withAuthContext(adminDashboardReq, accessToken, testJWTSigningKey)
	adminW = httptest.NewRecorder()
	h.Dashboard(adminW, adminDashboardReq)
	if adminW.Code != http.StatusOK {
		t.Fatalf("admin dashboard after create: status=%d body=%s", adminW.Code, adminW.Body)
	}

	adminDashboard = decodeSuccess[whatsapp.DashboardResponse](t, adminW)
	if len(adminDashboard.MaintenanceIntakes) != 1 {
		t.Fatalf("expected 1 maintenance intake in dashboard, got %d", len(adminDashboard.MaintenanceIntakes))
	}
	if adminDashboard.MaintenanceIntakes[0].ID != created.ID {
		t.Fatalf("dashboard intake id mismatch: %q vs %q", adminDashboard.MaintenanceIntakes[0].ID, created.ID)
	}
	if adminDashboard.Threads[0].Messages[0].MaintenanceRequestID == nil {
		t.Fatalf("expected message to have maintenance_request_id set: %+v", adminDashboard.Threads[0].Messages[0])
	}

	dismissReq := httptest.NewRequest(http.MethodPatch, "/whatsapp/"+schemeID+"/maintenance-intakes/"+created.ID, nil)
	dismissReq = withRouteParams(dismissReq, map[string]string{"schemeId": schemeID, "intakeId": created.ID})
	dismissReq = dismissReq.WithContext(auth.ContextWithIdentity(dismissReq.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	dismissW := httptest.NewRecorder()
	h.DismissMaintenanceIntake(dismissW, dismissReq)
	if dismissW.Code != http.StatusBadRequest {
		t.Fatalf("trustee dismiss ticket_created intake should fail: status=%d body=%s", dismissW.Code, dismissW.Body)
	}

	createReq = httptest.NewRequest(http.MethodPost, "/whatsapp/"+schemeID+"/messages/"+msgID+"/maintenance-request", bytes.NewReader(createBody))
	createReq = withRouteParams(createReq, map[string]string{"schemeId": schemeID, "messageId": msgID})
	createReq = createReq.WithContext(auth.ContextWithIdentity(createReq.Context(), trusteeUserID, orgID, string(auth.RoleTrustee)))
	createW = httptest.NewRecorder()
	h.CreateMaintenanceFromMessage(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("trustee re-create maintenance from same message should return existing intake: status=%d body=%s", createW.Code, createW.Body)
	}

	dismissReq = httptest.NewRequest(http.MethodPatch, "/whatsapp/"+schemeID+"/maintenance-intakes/"+created.ID, nil)
	dismissReq = withRouteParams(dismissReq, map[string]string{"schemeId": schemeID, "intakeId": created.ID})
	dismissReq = dismissReq.WithContext(auth.ContextWithIdentity(dismissReq.Context(), residentUserID, orgID, string(auth.RoleResident)))
	dismissW = httptest.NewRecorder()
	h.DismissMaintenanceIntake(dismissW, dismissReq)
	if dismissW.Code != http.StatusForbidden {
		t.Fatalf("resident dismiss maintenance intake should be forbidden: status=%d body=%s", dismissW.Code, dismissW.Body)
	}
}
