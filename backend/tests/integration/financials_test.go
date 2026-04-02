//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/financials"
)

func newFinancialsHandler(t *testing.T) *financials.Handler {
	t.Helper()
	return financials.NewHandler(financials.NewService(testPool))
}

func TestFinancials_DashboardAndUpdates(t *testing.T) {
	h := newFinancialsHandler(t)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "4D")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident Owner", string(auth.RoleResident), &unitID)

	budgetBody, _ := json.Marshal(map[string]any{
		"category":       "Maintenance",
		"period_label":   "2026",
		"budgeted_cents": 52000000,
		"actual_cents":   38125000,
	})
	req := httptest.NewRequest(http.MethodPut, "/financials/"+schemeID+"/budget-lines", bytes.NewReader(budgetBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.UpsertBudgetLine(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert budget line: status=%d body=%s", w.Code, w.Body)
	}
	line := decodeSuccess[financials.BudgetLineInfo](t, w)
	if line.Category != "Maintenance" || line.PeriodLabel != "2026" {
		t.Fatalf("unexpected budget line: %+v", line)
	}

	reserveBody, _ := json.Marshal(map[string]any{
		"balance_cents": 18450000,
		"target_cents":  36000000,
	})
	req = httptest.NewRequest(http.MethodPut, "/financials/"+schemeID+"/reserve-fund", bytes.NewReader(reserveBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.UpdateReserveFund(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update reserve fund: status=%d body=%s", w.Code, w.Body)
	}

	schemeUUID := mustParseFinancialUUID(t, schemeID)
	unitUUID := mustParseFinancialUUID(t, unitID)
	period, err := testQ.CreateLevyPeriod(t.Context(), dbgen.CreateLevyPeriodParams{
		SchemeID:    schemeUUID,
		Label:       "March 2026",
		AmountCents: 245000,
		DueDate:     pgtype.Date{Time: time.Now().AddDate(0, 0, -3), Valid: true},
	})
	if err != nil {
		t.Fatalf("create levy period: %v", err)
	}
	if _, err := testQ.CreateLevyAccount(t.Context(), dbgen.CreateLevyAccountParams{
		UnitID:      unitUUID,
		PeriodID:    period.ID,
		AmountCents: 245000,
		DueDate:     pgtype.Date{Time: time.Now().AddDate(0, 0, -3), Valid: true},
	}); err != nil {
		t.Fatalf("create levy account: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/financials/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard: status=%d body=%s", w.Code, w.Body)
	}
	dashboard := decodeSuccess[financials.DashboardResponse](t, w)
	if dashboard.SelectedPeriod != "2026" || len(dashboard.BudgetLines) != 1 {
		t.Fatalf("unexpected dashboard budget state: %+v", dashboard)
	}
	if dashboard.ReserveFund == nil || dashboard.ReserveFund.TargetCents != 36000000 {
		t.Fatalf("unexpected reserve fund: %+v", dashboard.ReserveFund)
	}
	if dashboard.LevySummary == nil || dashboard.LevySummary.TotalBilledCents != 245000 || dashboard.LevySummary.OverdueCount != 1 {
		t.Fatalf("unexpected levy summary: %+v", dashboard.LevySummary)
	}

	req = httptest.NewRequest(http.MethodGet, "/financials/"+schemeID, nil)
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Dashboard(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resident dashboard: status=%d body=%s", w.Code, w.Body)
	}
	residentDashboard := decodeSuccess[financials.DashboardResponse](t, w)
	if residentDashboard.Role != string(auth.RoleResident) || len(residentDashboard.BudgetLines) != 1 {
		t.Fatalf("unexpected resident dashboard: %+v", residentDashboard)
	}

	req = httptest.NewRequest(http.MethodPut, "/financials/"+schemeID+"/budget-lines", bytes.NewReader(budgetBody))
	req = withRouteParams(req, map[string]string{"schemeId": schemeID})
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.UpsertBudgetLine(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident upsert budget line should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func mustParseFinancialUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", value, err)
	}
	return id
}
