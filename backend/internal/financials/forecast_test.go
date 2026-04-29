package financials

import "testing"

func TestBuildLevyForecastShortfallRisk(t *testing.T) {
	forecast := buildLevyForecast(levyForecastInput{
		MonthsProjected: 12,
		CurrentReserveBalanceCents: 10_000_00,
		ReserveTargetCents:         25_000_00,
		CurrentMonthlyLevyCents:    2_500_00,
		LatestUnitCount:            10,
		Points: []levyForecastPointInput{
			{PeriodLabel: "January 2026", BilledCents: 25_000_00, CollectedCents: 20_000_00, ExpenseCents: 22_000_00, HasExpense: true},
			{PeriodLabel: "February 2026", BilledCents: 25_000_00, CollectedCents: 21_000_00, ExpenseCents: 23_000_00, HasExpense: true},
			{PeriodLabel: "March 2026", BilledCents: 25_000_00, CollectedCents: 20_000_00, ExpenseCents: 23_000_00, HasExpense: true},
		},
	})

	if forecast == nil {
		t.Fatal("expected forecast")
	}
	if forecast.Status != "shortfall_risk" {
		t.Fatalf("status = %q", forecast.Status)
	}
	if forecast.Confidence != "medium" {
		t.Fatalf("confidence = %q", forecast.Confidence)
	}
	if forecast.AverageCollectionRatePct != 81 {
		t.Fatalf("average collection pct = %d", forecast.AverageCollectionRatePct)
	}
	if forecast.ProjectedShortfallCents <= 0 {
		t.Fatalf("expected shortfall, got %d", forecast.ProjectedShortfallCents)
	}
	if forecast.RecommendedMonthlyIncreaseCents == 0 {
		t.Fatalf("expected recommended increase: %+v", forecast)
	}
}

func TestBuildLevyForecastHealthy(t *testing.T) {
	forecast := buildLevyForecast(levyForecastInput{
		MonthsProjected: 12,
		CurrentReserveBalanceCents: 30_000_00,
		ReserveTargetCents:         25_000_00,
		CurrentMonthlyLevyCents:    2_500_00,
		LatestUnitCount:            10,
		Points: []levyForecastPointInput{
			{PeriodLabel: "January 2026", BilledCents: 25_000_00, CollectedCents: 25_000_00, ExpenseCents: 20_000_00, HasExpense: true},
			{PeriodLabel: "February 2026", BilledCents: 25_000_00, CollectedCents: 25_000_00, ExpenseCents: 20_000_00, HasExpense: true},
			{PeriodLabel: "March 2026", BilledCents: 25_000_00, CollectedCents: 25_000_00, ExpenseCents: 20_000_00, HasExpense: true},
		},
	})

	if forecast == nil {
		t.Fatal("expected forecast")
	}
	if forecast.Status != "healthy" {
		t.Fatalf("status = %q", forecast.Status)
	}
	if forecast.ProjectedShortfallCents != 0 {
		t.Fatalf("shortfall = %d", forecast.ProjectedShortfallCents)
	}
	if forecast.RecommendedMonthlyIncreaseCents != 0 {
		t.Fatalf("recommended increase = %d", forecast.RecommendedMonthlyIncreaseCents)
	}
}

func TestBuildLevyForecastWatch(t *testing.T) {
	forecast := buildLevyForecast(levyForecastInput{
		MonthsProjected: 12,
		CurrentReserveBalanceCents: 40_000_00,
		ReserveTargetCents:         25_000_00,
		CurrentMonthlyLevyCents:    2_500_00,
		LatestUnitCount:            10,
		Points: []levyForecastPointInput{
			{PeriodLabel: "January 2026", BilledCents: 25_000_00, CollectedCents: 22_000_00, ExpenseCents: 23_000_00, HasExpense: true},
			{PeriodLabel: "February 2026", BilledCents: 25_000_00, CollectedCents: 22_000_00, ExpenseCents: 23_000_00, HasExpense: true},
			{PeriodLabel: "March 2026", BilledCents: 25_000_00, CollectedCents: 22_000_00, ExpenseCents: 23_000_00, HasExpense: true},
		},
	})

	if forecast == nil {
		t.Fatal("expected forecast")
	}
	if forecast.Status != "watch" {
		t.Fatalf("status = %q", forecast.Status)
	}
	if forecast.ProjectedShortfallCents != 0 {
		t.Fatalf("shortfall = %d", forecast.ProjectedShortfallCents)
	}
}

func TestBuildLevyForecastInsufficientData(t *testing.T) {
	forecast := buildLevyForecast(levyForecastInput{
		MonthsProjected: 12,
		CurrentReserveBalanceCents: 0,
		ReserveTargetCents:         10_000_00,
		CurrentMonthlyLevyCents:    2_000_00,
		LatestUnitCount:            5,
		Points: []levyForecastPointInput{
			{PeriodLabel: "March 2026", BilledCents: 10_000_00, CollectedCents: 8_000_00},
		},
	})

	if forecast == nil {
		t.Fatal("expected forecast")
	}
	if forecast.Status != "insufficient_data" {
		t.Fatalf("status = %q", forecast.Status)
	}
	if forecast.Confidence != "low" {
		t.Fatalf("confidence = %q", forecast.Confidence)
	}
	if len(forecast.Notes) == 0 {
		t.Fatalf("expected explanatory notes")
	}
}
