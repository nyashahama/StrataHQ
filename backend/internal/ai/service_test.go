package ai

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/stratahq/backend/db/gen"
)

func TestMapMembersRedactsPersonalIdentifiers(t *testing.T) {
	items := mapMembers([]dbgen.ListSchemeMembersBySchemeRow{
		{
			FullName: "Resident Owner",
			Email:    "resident@example.com",
			Role:     "resident",
			UnitID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UnitIdentifier: pgtype.Text{
				String: "7B",
				Valid:  true,
			},
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapMembers() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["full_name"]; ok {
		t.Fatalf("mapMembers() leaked full_name: %+v", items[0])
	}
	if _, ok := items[0]["email"]; ok {
		t.Fatalf("mapMembers() leaked email: %+v", items[0])
	}
	if _, ok := items[0]["unit_id"]; ok {
		t.Fatalf("mapMembers() leaked unit_id: %+v", items[0])
	}
	if items[0]["role"] != "resident" {
		t.Fatalf("mapMembers() role = %v, want resident", items[0]["role"])
	}
	if items[0]["unit_identifier"] != "7B" {
		t.Fatalf("mapMembers() unit_identifier = %v, want 7B", items[0]["unit_identifier"])
	}
}

func TestMapUnitsRedactsOwnerName(t *testing.T) {
	items := mapUnits([]dbgen.Unit{
		{
			Identifier:      "7B",
			OwnerName:       "Resident Owner",
			Floor:           7,
			SectionValueBps: 1250,
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapUnits() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["owner_name"]; ok {
		t.Fatalf("mapUnits() leaked owner_name: %+v", items[0])
	}
}

func TestTopOverdueAccountsRedactsOwnerName(t *testing.T) {
	items := topOverdueAccounts([]dbgen.ListLevyAccountsByPeriodRow{
		{
			UnitIdentifier: "7B",
			OwnerName:      "Resident Owner",
			AmountCents:    245000,
			PaidCents:      0,
			DueDate:        pgtype.Date{Time: time.Now().AddDate(0, 0, -2), Valid: true},
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("topOverdueAccounts() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["owner_name"]; ok {
		t.Fatalf("topOverdueAccounts() leaked owner_name: %+v", items[0])
	}
}

func TestCollectionPctFromRowsTreatsEmptyAndZeroDueAsFullyCollected(t *testing.T) {
	if got := collectionPctFromRows(nil); got != 100 {
		t.Fatalf("empty collection pct = %d, want 100", got)
	}

	got := collectionPctFromRows([]dbgen.ListLevyAccountsByPeriodRow{
		{AmountCents: 0, PaidCents: 0},
	})
	if got != 100 {
		t.Fatalf("zero-due collection pct = %d, want 100", got)
	}
}

func TestMapMaintenanceRedactsContractorName(t *testing.T) {
	items := mapMaintenance([]dbgen.MaintenanceRequest{
		{
			Title:    "Fix leak",
			Category: dbgen.MaintenanceCategoryPlumbing,
			Status:   dbgen.MaintenanceStatusOpen,
			ContractorName: pgtype.Text{
				String: "Acme Plumbing",
				Valid:  true,
			},
			SubmittedByUnit: pgtype.Text{
				String: "7B",
				Valid:  true,
			},
			SlaHours:  24,
			CreatedAt: time.Now(),
		},
	}, 5)

	if len(items) != 1 {
		t.Fatalf("mapMaintenance() returned %d items, want 1", len(items))
	}
	if _, ok := items[0]["contractor_name"]; ok {
		t.Fatalf("mapMaintenance() leaked contractor_name: %+v", items[0])
	}
}

func TestMapFinancialsScopesTotalsToCurrentBudgetPeriod(t *testing.T) {
	lines := []dbgen.BudgetLine{
		{PeriodLabel: "October 2025", Category: "Maintenance", BudgetedCents: 1000, ActualCents: 250},
		{PeriodLabel: "September 2026", Category: "Maintenance", BudgetedCents: 5000, ActualCents: 1000},
		{PeriodLabel: "September 2026", Category: "Insurance", BudgetedCents: 7000, ActualCents: 3000},
	}

	payload := mapFinancials(lines, false, dbgen.ReserveFund{})

	if payload["selected_period"] != "September 2026" {
		t.Fatalf("selected_period = %v, want September 2026", payload["selected_period"])
	}
	if payload["total_budgeted_cents"] != int64(12000) || payload["total_actual_cents"] != int64(4000) {
		t.Fatalf("totals = (%v, %v), want (12000, 4000)", payload["total_budgeted_cents"], payload["total_actual_cents"])
	}
	budgetLines, ok := payload["budget_lines"].([]map[string]any)
	if !ok {
		t.Fatalf("budget_lines type = %T", payload["budget_lines"])
	}
	if len(budgetLines) != 2 {
		t.Fatalf("budget_lines len = %d, want 2: %#v", len(budgetLines), budgetLines)
	}
}
