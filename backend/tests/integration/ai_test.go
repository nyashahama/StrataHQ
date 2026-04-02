//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/ai"
	"github.com/stratahq/backend/internal/auth"
)

type fakeCompleter struct {
	lastSystem string
}

func (f *fakeCompleter) Complete(ctx context.Context, systemPrompt string, history []ai.Message, message string) (string, error) {
	f.lastSystem = systemPrompt
	return "AI response for: " + message, nil
}

func newAIHandler(t *testing.T, completer *fakeCompleter) *ai.Handler {
	t.Helper()
	return ai.NewHandler(ai.NewService(testPool, completer))
}

func TestAI_CopilotUsesRealSchemeContext(t *testing.T) {
	completer := &fakeCompleter{}
	h := newAIHandler(t, completer)
	accessToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, accessToken)

	unitID := createUnitRecord(t, schemeID, "7B")
	residentEmail := uniqueEmail(t)
	residentUserID := createMemberRecord(t, orgID, schemeID, residentEmail, "Resident Owner", string(auth.RoleResident), &unitID)
	trusteeEmail := uniqueEmail(t)
	createMemberRecord(t, orgID, schemeID, trusteeEmail, "Trustee Member", string(auth.RoleTrustee), nil)

	schemeUUID := mustParseAIUUID(t, schemeID)
	unitUUID := mustParseAIUUID(t, unitID)
	period, err := testQ.CreateLevyPeriod(t.Context(), dbgen.CreateLevyPeriodParams{
		SchemeID:    schemeUUID,
		Label:       "April 2026",
		AmountCents: 245000,
		DueDate:     pgtype.Date{Time: time.Now().AddDate(0, 0, -2), Valid: true},
	})
	if err != nil {
		t.Fatalf("create levy period: %v", err)
	}
	if _, err := testQ.CreateLevyAccount(t.Context(), dbgen.CreateLevyAccountParams{
		UnitID:      unitUUID,
		PeriodID:    period.ID,
		AmountCents: 245000,
		DueDate:     pgtype.Date{Time: time.Now().AddDate(0, 0, -2), Valid: true},
	}); err != nil {
		t.Fatalf("create levy account: %v", err)
	}
	if _, err := testQ.UpsertBudgetLine(t.Context(), dbgen.UpsertBudgetLineParams{
		SchemeID:      schemeUUID,
		Category:      "Maintenance",
		PeriodLabel:   "2026",
		BudgetedCents: 52000000,
		ActualCents:   38125000,
	}); err != nil {
		t.Fatalf("upsert budget line: %v", err)
	}
	if _, err := testQ.UpsertReserveFund(t.Context(), dbgen.UpsertReserveFundParams{
		SchemeID:     schemeUUID,
		BalanceCents: 18450000,
		TargetCents:  36000000,
	}); err != nil {
		t.Fatalf("upsert reserve fund: %v", err)
	}
	if _, err := testQ.CreateNotice(t.Context(), dbgen.CreateNoticeParams{
		SchemeID: schemeUUID,
		Title:    "Water shutdown",
		Body:     "Water will be off tomorrow for repairs.",
		Type:     dbgen.NoticeTypeUrgent,
	}); err != nil {
		t.Fatalf("create notice: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"scheme_id": schemeID,
		"message":   "Summarise levy and maintenance risk for this scheme",
		"history": []map[string]string{
			{"role": "user", "content": "What should I focus on today?"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/ai/copilot", bytes.NewReader(body))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Copilot(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ai copilot: status=%d body=%s", w.Code, w.Body)
	}
	resp := decodeSuccess[struct {
		Answer string `json:"answer"`
	}](t, w)
	if !strings.Contains(resp.Answer, "Summarise levy and maintenance risk for this scheme") {
		t.Fatalf("unexpected AI response: %+v", resp)
	}
	for _, expected := range []string{"Test Scheme", "Water shutdown", "Maintenance", "April 2026", "available_actions"} {
		if !strings.Contains(completer.lastSystem, expected) {
			t.Fatalf("expected %q in AI context, got %s", expected, completer.lastSystem)
		}
	}

	req = httptest.NewRequest(http.MethodPost, "/ai/copilot", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), residentUserID, orgID, string(auth.RoleResident)))
	w = httptest.NewRecorder()
	h.Copilot(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("resident copilot should be forbidden: status=%d body=%s", w.Code, w.Body)
	}
}

func mustParseAIUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", value, err)
	}
	return id
}
