//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/jobs"
	"github.com/stratahq/backend/internal/levy"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/whatsapp"
)

func newLevyHandler(t *testing.T) *levy.Handler {
	t.Helper()
	emailSender := &notification.NoopSender{}
	whatsAppSender := &whatsapp.NoOpSender{}
	svc := levy.NewService(testPool, emailSender, whatsAppSender)
	return levy.NewHandler(svc)
}

func TestLevy_AdminDashboardCreateAndReconcile(t *testing.T) {
	h := newLevyHandler(t)
	accessToken, _ := setupAgent(t)
	schemeID := setupScheme(t, accessToken)
	ctx := context.Background()

	unitA, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "1A", "A. Adams"))
	if err != nil {
		t.Fatalf("create unit A: %v", err)
	}
	_, err = testQ.CreateUnit(ctx, createUnitParams(schemeID, "2B", "B. Brown"))
	if err != nil {
		t.Fatalf("create unit B: %v", err)
	}

	createBody, _ := json.Marshal(map[string]any{
		"label":        "April 2026",
		"amount_cents": 245000,
		"due_date":     "2026-04-10",
	})
	req := httptest.NewRequest(http.MethodPost, "/levies/"+schemeID+"/periods", bytes.NewReader(createBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.CreatePeriod(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create period: status=%d body=%s", w.Code, w.Body)
	}
	created := decodeSuccess[levy.PeriodInfo](t, w)
	if created.Label != "April 2026" {
		t.Fatalf("unexpected period label=%q", created.Label)
	}

	req = httptest.NewRequest(http.MethodGet, "/levies/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard before reconcile: status=%d body=%s", w.Code, w.Body)
	}
	dashboard := decodeSuccess[levy.DashboardResponse](t, w)
	if dashboard.CurrentPeriod == nil || dashboard.CurrentPeriod.Label != "April 2026" {
		t.Fatalf("unexpected current period: %+v", dashboard.CurrentPeriod)
	}
	if len(dashboard.LevyRoll) != 2 {
		t.Fatalf("expected 2 levy accounts, got %d", len(dashboard.LevyRoll))
	}
	if dashboard.CollectionRatePct != 0 {
		t.Fatalf("expected 0%% collection before reconcile, got %d", dashboard.CollectionRatePct)
	}

	reconcileBody, _ := json.Marshal(map[string]any{
		"payments": []map[string]any{
			{
				"account_id":   dashboard.LevyRoll[0].ID,
				"amount_cents": 245000,
				"payment_date": "2026-04-05",
				"reference":    uuid.NewString(),
				"bank_ref":     "FNB-001",
			},
			{
				"account_id":   dashboard.LevyRoll[1].ID,
				"amount_cents": 120000,
				"payment_date": "2026-04-06",
				"reference":    uuid.NewString(),
				"bank_ref":     "FNB-002",
			},
		},
	})
	req = httptest.NewRequest(http.MethodPost, "/levies/"+schemeID+"/reconcile", bytes.NewReader(reconcileBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Reconcile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("reconcile: status=%d body=%s", w.Code, w.Body)
	}
	result := decodeSuccess[levy.ReconcileResult](t, w)
	if result.AppliedCount != 2 || len(result.UpdatedAccountIDs) != 2 {
		t.Fatalf("unexpected reconcile result: %+v", result)
	}

	req = httptest.NewRequest(http.MethodGet, "/levies/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard after reconcile: status=%d body=%s", w.Code, w.Body)
	}
	dashboard = decodeSuccess[levy.DashboardResponse](t, w)
	if dashboard.CollectionRatePct != 74 {
		t.Fatalf("expected 74%% collection after reconcile, got %d", dashboard.CollectionRatePct)
	}
	if dashboard.TotalCollectedCents != 365000 {
		t.Fatalf("unexpected collected amount=%d", dashboard.TotalCollectedCents)
	}

	var paidCount, partialCount int
	for _, account := range dashboard.LevyRoll {
		switch account.Status {
		case "paid":
			paidCount++
		case "partial":
			partialCount++
		}
	}
	if paidCount != 1 || partialCount != 1 {
		t.Fatalf("unexpected levy roll statuses: %+v", dashboard.LevyRoll)
	}

	accountRows, err := testQ.ListLevyAccountsByUnit(ctx, unitA.ID)
	if err != nil {
		t.Fatalf("list levy accounts by unit: %v", err)
	}
	if len(accountRows) != 1 {
		t.Fatalf("expected 1 levy account for unit A, got %d", len(accountRows))
	}
	payments, err := testQ.ListLevyPaymentsByUnit(ctx, unitA.ID)
	if err != nil {
		t.Fatalf("list levy payments by unit: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("expected 1 levy payment for unit A, got %d", len(payments))
	}
}

func createUnitParams(schemeID, identifier, ownerName string) dbgen.CreateUnitParams {
	return dbgen.CreateUnitParams{
		SchemeID:        mustParseUUID(schemeID),
		Identifier:      identifier,
		OwnerName:       ownerName,
		Floor:           1,
		SectionValueBps: 500,
	}
}

func mustParseUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}

type failFirstWhatsAppJobQueue struct {
	delegate           *jobs.Service
	failedWhatsAppOnce bool
}

func (q *failFirstWhatsAppJobQueue) Enqueue(ctx context.Context, input jobs.EnqueueInput) (dbgen.BackgroundJob, error) {
	if input.Kind == jobs.KindCollectionReminderWhatsApp && !q.failedWhatsAppOnce {
		q.failedWhatsAppOnce = true
		return dbgen.BackgroundJob{}, errors.New("whatsapp enqueue unavailable")
	}
	return q.delegate.Enqueue(ctx, input)
}

func TestSendReminderRecordsQueuedEventAndEnqueuesJobs(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	claims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	identity := auth.Identity{UserID: claims.Subject, OrgID: orgID, Role: claims.Role}
	schemeID := setupScheme(t, accessToken)
	schemeUUID := mustParseUUID(schemeID)

	unit, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "9R", "Reminder Owner"))
	if err != nil {
		t.Fatalf("create reminder unit: %v", err)
	}
	owner, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        uniqueEmail(t),
		PasswordHash: "test-hash",
		FullName:     "Reminder Owner",
	})
	if err != nil {
		t.Fatalf("create owner user: %v", err)
	}
	if _, err := testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   owner.ID,
		SchemeID: schemeUUID,
		UnitID:   pgtype.UUID{Bytes: unit.ID, Valid: true},
		Role:     string(auth.RoleOwner),
	}); err != nil {
		t.Fatalf("create owner scheme membership: %v", err)
	}
	phone := "+27715550123"
	if _, err := testQ.CreateWhatsAppThread(ctx, dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         unit.ID,
		ResidentUserID: pgtype.UUID{Bytes: owner.ID, Valid: true},
		PhoneNumber:    pgtype.Text{String: phone, Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: time.Now().UTC().Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create whatsapp thread: %v", err)
	}

	jobService := jobs.NewService(testQ, jobs.Registry{}, nil, jobs.RealClock{}, jobs.Config{WorkerID: "integration-enqueuer"})
	service := levy.NewServiceWithAuditAndJobs(testPool, &notification.NoopSender{}, whatsapp.NewNoOpSender(), nil, jobService)
	service.SetMaxJobAttempts(3)
	_, err = service.CreatePeriod(ctx, identity, schemeID, levy.CreatePeriodInput{
		Label:       "Reminder Test",
		AmountCents: 245000,
		DueDate:     time.Now().UTC().AddDate(0, 0, -1),
	})
	if err != nil {
		t.Fatalf("create levy period: %v", err)
	}

	accounts, err := testQ.ListAttentionAccountsByScheme(ctx, schemeUUID)
	if err != nil {
		t.Fatalf("list attention accounts: %v", err)
	}
	var accountID uuid.UUID
	for _, account := range accounts {
		if account.UnitID == unit.ID {
			accountID = account.LevyAccountID
			break
		}
	}
	if accountID == uuid.Nil {
		t.Fatalf("expected reminder unit in attention accounts: %+v", accounts)
	}

	event, err := service.SendReminder(ctx, identity, schemeID, accountID.String(), levy.SendReminderInput{
		Email:    levy.ReminderChannelInput{Enabled: true, Subject: "Reminder subject", Body: "Email body"},
		WhatsApp: levy.ReminderChannelInput{Enabled: true, Body: "WhatsApp body"},
	})
	if err != nil {
		t.Fatalf("send reminder: %v", err)
	}
	if event.EventType != "reminder_sent" {
		t.Fatalf("event type = %q, want reminder_sent", event.EventType)
	}

	stored, err := testQ.GetCollectionEventByID(ctx, mustParseUUID(event.ID))
	if err != nil {
		t.Fatalf("get collection event: %v", err)
	}
	if stored.EmailStatus.String != "queued" || stored.WhatsappStatus.String != "queued" {
		t.Fatalf("delivery statuses = email:%+v whatsapp:%+v, want queued", stored.EmailStatus, stored.WhatsappStatus)
	}
	if stored.EmailTo.String != owner.Email || stored.WhatsappTo.String != phone {
		t.Fatalf("delivery recipients = email:%q whatsapp:%q, want %q and %q", stored.EmailTo.String, stored.WhatsappTo.String, owner.Email, phone)
	}

	var emailAttempts int32
	if err := testPool.QueryRow(ctx, `
		SELECT max_attempts
		FROM background_jobs
		WHERE kind = $1 AND idempotency_key = $2
	`, jobs.KindCollectionReminderEmail, accountID.String()+":reminder_email").Scan(&emailAttempts); err != nil {
		t.Fatalf("get email job: %v", err)
	}
	var whatsappAttempts int32
	if err := testPool.QueryRow(ctx, `
		SELECT max_attempts
		FROM background_jobs
		WHERE kind = $1 AND idempotency_key = $2
	`, jobs.KindCollectionReminderWhatsApp, accountID.String()+":reminder_whatsapp").Scan(&whatsappAttempts); err != nil {
		t.Fatalf("get whatsapp job: %v", err)
	}
	if emailAttempts != 3 || whatsappAttempts != 3 {
		t.Fatalf("max attempts = email:%d whatsapp:%d, want 3", emailAttempts, whatsappAttempts)
	}
}

func TestSendReminderRetryAfterWhatsAppEnqueueFailureDoesNotDuplicateEmailJob(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	claims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	identity := auth.Identity{UserID: claims.Subject, OrgID: orgID, Role: claims.Role}
	schemeID := setupScheme(t, accessToken)
	schemeUUID := mustParseUUID(schemeID)

	unit, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "9F", "Partial Failure Owner"))
	if err != nil {
		t.Fatalf("create reminder unit: %v", err)
	}
	owner, err := testQ.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        uniqueEmail(t),
		PasswordHash: "test-hash",
		FullName:     "Partial Failure Owner",
	})
	if err != nil {
		t.Fatalf("create owner user: %v", err)
	}
	if _, err := testQ.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
		UserID:   owner.ID,
		SchemeID: schemeUUID,
		UnitID:   pgtype.UUID{Bytes: unit.ID, Valid: true},
		Role:     string(auth.RoleOwner),
	}); err != nil {
		t.Fatalf("create owner scheme membership: %v", err)
	}
	if _, err := testQ.CreateWhatsAppThread(ctx, dbgen.CreateWhatsAppThreadParams{
		SchemeID:       schemeUUID,
		UnitID:         unit.ID,
		ResidentUserID: pgtype.UUID{Bytes: owner.ID, Valid: true},
		PhoneNumber:    pgtype.Text{String: "+27715550124", Valid: true},
		Connected:      true,
		ConsentedAt:    pgtype.Timestamptz{Time: time.Now().UTC().Add(-24 * time.Hour), Valid: true},
		UnreadCount:    0,
		LastActiveAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create whatsapp thread: %v", err)
	}

	jobQueue := &failFirstWhatsAppJobQueue{
		delegate: jobs.NewService(testQ, jobs.Registry{}, nil, jobs.RealClock{}, jobs.Config{WorkerID: "integration-partial-failure"}),
	}
	service := levy.NewServiceWithAuditAndJobs(testPool, &notification.NoopSender{}, whatsapp.NewNoOpSender(), nil, jobQueue)
	_, err = service.CreatePeriod(ctx, identity, schemeID, levy.CreatePeriodInput{
		Label:       "Partial Failure Reminder Test",
		AmountCents: 245000,
		DueDate:     time.Now().UTC().AddDate(0, 0, -1),
	})
	if err != nil {
		t.Fatalf("create levy period: %v", err)
	}

	accounts, err := testQ.ListAttentionAccountsByScheme(ctx, schemeUUID)
	if err != nil {
		t.Fatalf("list attention accounts: %v", err)
	}
	var accountID uuid.UUID
	for _, account := range accounts {
		if account.UnitID == unit.ID {
			accountID = account.LevyAccountID
			break
		}
	}
	if accountID == uuid.Nil {
		t.Fatalf("expected reminder unit in attention accounts: %+v", accounts)
	}

	input := levy.SendReminderInput{
		Email:    levy.ReminderChannelInput{Enabled: true, Subject: "Reminder subject", Body: "Email body"},
		WhatsApp: levy.ReminderChannelInput{Enabled: true, Body: "WhatsApp body"},
	}
	firstEvent, err := service.SendReminder(ctx, identity, schemeID, accountID.String(), input)
	if err != nil {
		t.Fatalf("send reminder with partial failure: %v", err)
	}
	storedFirst, err := testQ.GetCollectionEventByID(ctx, mustParseUUID(firstEvent.ID))
	if err != nil {
		t.Fatalf("get first collection event: %v", err)
	}
	if storedFirst.EmailStatus.String != "queued" || storedFirst.WhatsappStatus.String != "failed" {
		t.Fatalf("first delivery statuses = email:%+v whatsapp:%+v, want email queued and whatsapp failed", storedFirst.EmailStatus, storedFirst.WhatsappStatus)
	}

	if _, err := service.SendReminder(ctx, identity, schemeID, accountID.String(), input); err != nil {
		t.Fatalf("retry send reminder: %v", err)
	}

	var emailJobs int
	if err := testPool.QueryRow(ctx, `
		SELECT count(*)
		FROM background_jobs
		WHERE kind = $1 AND idempotency_key = $2
	`, jobs.KindCollectionReminderEmail, accountID.String()+":reminder_email").Scan(&emailJobs); err != nil {
		t.Fatalf("count email jobs: %v", err)
	}
	var whatsappJobs int
	if err := testPool.QueryRow(ctx, `
		SELECT count(*)
		FROM background_jobs
		WHERE kind = $1 AND idempotency_key = $2
	`, jobs.KindCollectionReminderWhatsApp, accountID.String()+":reminder_whatsapp").Scan(&whatsappJobs); err != nil {
		t.Fatalf("count whatsapp jobs: %v", err)
	}
	if emailJobs != 1 || whatsappJobs != 1 {
		t.Fatalf("queued job counts = email:%d whatsapp:%d, want one job per channel after retry", emailJobs, whatsappJobs)
	}
}

func TestLevy_BankStatementImportLifecycle(t *testing.T) {
	ctx := context.Background()
	accessToken, orgID := setupAgent(t)
	claims, err := auth.ValidateAccessToken(accessToken, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	identity := auth.Identity{UserID: claims.Subject, OrgID: orgID, Role: claims.Role}
	schemeID := setupScheme(t, accessToken)

	unitA, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "1A", "A. Adams"))
	if err != nil {
		t.Fatalf("create unit A: %v", err)
	}
	unitB, err := testQ.CreateUnit(ctx, createUnitParams(schemeID, "2B", "B. Brown"))
	if err != nil {
		t.Fatalf("create unit B: %v", err)
	}

	service := levy.NewServiceWithAuditAndJobs(testPool, &notification.NoopSender{}, whatsapp.NewNoOpSender(), nil, jobs.NewService(testQ, jobs.Registry{}, nil, jobs.RealClock{}, jobs.Config{WorkerID: "integration-importer"}))
	service.SetMaxJobAttempts(3)

	period, err := service.CreatePeriod(ctx, identity, schemeID, levy.CreatePeriodInput{
		Label:       "May 2026",
		AmountCents: 245000,
		DueDate:     time.Now().UTC().AddDate(0, 0, -1),
	})
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	rawCSV := []byte("Date,Description,Reference,Amount\n2026-04-01,EFT Unit 1A,1A,2450.00\n2026-04-02,Manual review needed,unknown,1200.00\n")

	importResp, err := service.ImportBankStatement(ctx, identity, schemeID, levy.BankStatementImportInput{
		BankName:         "fnb",
		OriginalFilename: "fnb_statement.csv",
		RawCSV:           rawCSV,
	})
	if err != nil {
		t.Fatalf("import statement: %v", err)
	}
	if importResp.Status != string(dbgen.BankStatementImportStatusQueued) {
		t.Fatalf("import status = %q, want queued", importResp.Status)
	}

	var queuedJobs int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM background_jobs WHERE kind = $1 AND idempotency_key = $2`, jobs.KindBankStatementImport, importResp.ID).Scan(&queuedJobs); err != nil {
		t.Fatalf("count background jobs: %v", err)
	}
	if queuedJobs != 1 {
		t.Fatalf("queued jobs = %d, want 1", queuedJobs)
	}

	if err := service.ProcessBankStatementImport(ctx, importResp.ID); err != nil {
		t.Fatalf("process import: %v", err)
	}

	details, err := service.GetBankStatementImport(ctx, identity, schemeID, importResp.ID)
	if err != nil {
		t.Fatalf("get import details: %v", err)
	}
	if details.Status != string(dbgen.BankStatementImportStatusReviewRequired) {
		t.Fatalf("status = %q, want review_required", details.Status)
	}
	if details.MatchedRows != 1 || details.UnmatchedRows != 1 {
		t.Fatalf("unexpected import counts: %+v", details.BankStatementImportResponse)
	}
	if len(details.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(details.Rows))
	}

	accounts, err := testQ.ListLevyAccountsByPeriod(ctx, uuid.MustParse(period.ID))
	if err != nil {
		t.Fatalf("list levy accounts by period: %v", err)
	}
	var unitBAccountID uuid.UUID
	for _, account := range accounts {
		if account.UnitIdentifier == "2B" {
			unitBAccountID = account.ID
			break
		}
	}
	if unitBAccountID == uuid.Nil {
		t.Fatal("missing unit B account")
	}

	var manualRowID string
	for _, row := range details.Rows {
		if row.Status == "unmatched" {
			manualRowID = row.ID
			break
		}
	}
	if manualRowID == "" {
		t.Fatal("missing unmatched row")
	}

	appliedResp, err := service.ApplyBankStatementImport(ctx, identity, schemeID, importResp.ID, []levy.BankStatementManualMatchInput{
		{
			RowID:       manualRowID,
			AccountID:   unitBAccountID.String(),
			PaymentDate: "2026-04-02",
			AmountCents: 120000,
			Reference:   "MANUAL-002",
			BankRef:     ptrString("Manual review needed"),
		},
	})
	if err != nil {
		t.Fatalf("apply import: %v", err)
	}
	if appliedResp.Status != string(dbgen.BankStatementImportStatusApplied) {
		t.Fatalf("applied status = %q, want applied", appliedResp.Status)
	}
	if appliedResp.AppliedRows != 2 {
		t.Fatalf("applied rows = %d, want 2", appliedResp.AppliedRows)
	}

	paymentsA, err := testQ.ListLevyPaymentsByUnit(ctx, unitA.ID)
	if err != nil {
		t.Fatalf("list unit A payments: %v", err)
	}
	paymentsB, err := testQ.ListLevyPaymentsByUnit(ctx, unitB.ID)
	if err != nil {
		t.Fatalf("list unit B payments: %v", err)
	}
	if len(paymentsA) != 1 || len(paymentsB) != 1 {
		t.Fatalf("expected one payment per unit, got A=%d B=%d", len(paymentsA), len(paymentsB))
	}

	reupload, err := service.ImportBankStatement(ctx, identity, schemeID, levy.BankStatementImportInput{
		BankName:         "fnb",
		OriginalFilename: "fnb_statement.csv",
		RawCSV:           rawCSV,
	})
	if err != nil {
		t.Fatalf("reupload import: %v", err)
	}
	if err := service.ProcessBankStatementImport(ctx, reupload.ID); err != nil {
		t.Fatalf("reprocess import: %v", err)
	}
	paymentsA, err = testQ.ListLevyPaymentsByUnit(ctx, unitA.ID)
	if err != nil {
		t.Fatalf("list unit A payments after reupload: %v", err)
	}
	paymentsB, err = testQ.ListLevyPaymentsByUnit(ctx, unitB.ID)
	if err != nil {
		t.Fatalf("list unit B payments after reupload: %v", err)
	}
	if len(paymentsA) != 1 || len(paymentsB) != 1 {
		t.Fatalf("reupload duplicated payments: A=%d B=%d", len(paymentsA), len(paymentsB))
	}
}

func ptrString(value string) *string {
	return &value
}

func TestLevy_BankStatementImportCrossSchemeForbidden(t *testing.T) {
	ctx := context.Background()

	accessTokenA, orgID := setupAgent(t)
	claimsA, err := auth.ValidateAccessToken(accessTokenA, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate access token A: %v", err)
	}
	identityA := auth.Identity{UserID: claimsA.Subject, OrgID: orgID, Role: claimsA.Role}
	schemeID_A := setupScheme(t, accessTokenA)

	unitA, err := testQ.CreateUnit(ctx, createUnitParams(schemeID_A, "A1", "Alice"))
	if err != nil {
		t.Fatalf("create unit A1: %v", err)
	}
	_ = unitA

	accessTokenB, _ := setupAgent(t)
	claimsB, err := auth.ValidateAccessToken(accessTokenB, testJWTSigningKey, "", "")
	if err != nil {
		t.Fatalf("validate access token B: %v", err)
	}
	identityB := auth.Identity{UserID: claimsB.Subject, OrgID: claimsB.OrgID, Role: claimsB.Role}
	schemeID_B := setupScheme(t, accessTokenB)

	unitB, err := testQ.CreateUnit(ctx, createUnitParams(schemeID_B, "B1", "Bob"))
	if err != nil {
		t.Fatalf("create unit B1: %v", err)
	}
	_ = unitB

	service := levy.NewServiceWithAuditAndJobs(testPool, &notification.NoopSender{}, whatsapp.NewNoOpSender(), nil, jobs.NewService(testQ, jobs.Registry{}, nil, jobs.RealClock{}, jobs.Config{WorkerID: "integration-cross-scheme"}))
	service.SetMaxJobAttempts(3)

	_, err = service.CreatePeriod(ctx, identityA, schemeID_A, levy.CreatePeriodInput{
		Label:       "May 2026",
		AmountCents: 245000,
		DueDate:     time.Now().UTC().AddDate(0, 0, -1),
	})
	if err != nil {
		t.Fatalf("create period A: %v", err)
	}

	periodB, err := service.CreatePeriod(ctx, identityB, schemeID_B, levy.CreatePeriodInput{
		Label:       "May 2026",
		AmountCents: 300000,
		DueDate:     time.Now().UTC().AddDate(0, 0, -1),
	})
	if err != nil {
		t.Fatalf("create period B: %v", err)
	}

	rawCSV := []byte("Date,Description,Reference,Amount\n2026-04-01,Payment for B1,B1,3000.00\n")

	importResp, err := service.ImportBankStatement(ctx, identityA, schemeID_A, levy.BankStatementImportInput{
		BankName:         "fnb",
		OriginalFilename: "cross_scheme.csv",
		RawCSV:           rawCSV,
	})
	if err != nil {
		t.Fatalf("import statement: %v", err)
	}

	if err := service.ProcessBankStatementImport(ctx, importResp.ID); err != nil {
		t.Fatalf("process import: %v", err)
	}

	details, err := service.GetBankStatementImport(ctx, identityA, schemeID_A, importResp.ID)
	if err != nil {
		t.Fatalf("get import details: %v", err)
	}
	if len(details.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(details.Rows))
	}
	row := details.Rows[0]

	accountsB, err := testQ.ListLevyAccountsByPeriod(ctx, uuid.MustParse(periodB.ID))
	if err != nil {
		t.Fatalf("list levy accounts for scheme B: %v", err)
	}
	if len(accountsB) == 0 {
		t.Fatal("no levy accounts for scheme B")
	}
	schemeBAccountID := accountsB[0].ID.String()

	_, applyErr := service.ApplyBankStatementImport(ctx, identityA, schemeID_A, importResp.ID, []levy.BankStatementManualMatchInput{
		{
			RowID:       row.ID,
			AccountID:   schemeBAccountID,
			PaymentDate: "2026-04-02",
			AmountCents: row.AmountCents,
			Reference:   "CROSS-SCHEME-ATTACK",
		},
	})
	if !errors.Is(applyErr, levy.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when matching to another scheme's levy account, got: %v", applyErr)
	}
}
