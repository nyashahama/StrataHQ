package financials

import (
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/stratahq/backend/db/gen"
)

func TestBudgetPeriodsSortByParsedPeriodDate(t *testing.T) {
	periods := budgetPeriods([]dbgen.BudgetLine{
		{PeriodLabel: "October 2025"},
		{PeriodLabel: "September 2026"},
		{PeriodLabel: "January 2026"},
		{PeriodLabel: "October 2025"},
	})

	want := []string{"September 2026", "January 2026", "October 2025"}
	if len(periods) != len(want) {
		t.Fatalf("period count = %d, want %d: %#v", len(periods), len(want), periods)
	}
	for i := range want {
		if periods[i] != want[i] {
			t.Fatalf("periods[%d] = %q, want %q (all=%#v)", i, periods[i], want[i], periods)
		}
	}
}

func TestSelectBudgetPeriodFallsBackWhenRequestedPeriodDoesNotExist(t *testing.T) {
	periods := []string{"September 2026", "January 2026"}

	if got := selectBudgetPeriod(periods, "Not a period"); got != "September 2026" {
		t.Fatalf("selected period = %q, want September 2026", got)
	}
}

func TestBudgetTotalsOnlyUseSelectedPeriod(t *testing.T) {
	schemeID := uuid.New()
	lines := []dbgen.BudgetLine{
		{ID: uuid.New(), SchemeID: schemeID, Category: "Maintenance", PeriodLabel: "September 2026", BudgetedCents: 1000, ActualCents: 250, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), SchemeID: schemeID, Category: "Insurance", PeriodLabel: "January 2026", BudgetedCents: 9000, ActualCents: 8000, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	items, totalBudgeted, totalActual := budgetLinesForPeriod(lines, "September 2026")

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if totalBudgeted != 1000 || totalActual != 250 {
		t.Fatalf("totals = (%d, %d), want (1000, 250)", totalBudgeted, totalActual)
	}
}
