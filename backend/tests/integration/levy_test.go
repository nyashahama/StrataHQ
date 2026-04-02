//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/levy"
)

func newLevyHandler(t *testing.T) *levy.Handler {
	t.Helper()
	return levy.NewHandler(levy.NewService(testPool))
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
				"reference":    "FNB-LEVY-APR-1A",
				"bank_ref":     "FNB-001",
			},
			{
				"account_id":   dashboard.LevyRoll[1].ID,
				"amount_cents": 120000,
				"payment_date": "2026-04-06",
				"reference":    "FNB-LEVY-APR-2B",
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
