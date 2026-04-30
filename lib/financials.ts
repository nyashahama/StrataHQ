export interface BudgetLineInfo {
  id: string;
  scheme_id: string;
  category: string;
  period_label: string;
  budgeted_cents: number;
  actual_cents: number;
  variance_cents: number;
  created_at: string;
  updated_at: string;
}

export interface ReserveFundInfo {
  scheme_id: string;
  balance_cents: number;
  target_cents: number;
  last_updated: string;
}

export interface LevySummaryInfo {
  period_label: string;
  total_billed_cents: number;
  total_collected_cents: number;
  collection_rate_pct: number;
  overdue_count: number;
}

export interface LevyForecastPointInfo {
  period_label: string;
  billed_cents: number;
  collected_cents: number;
  collection_rate_pct: number;
  expense_cents: number;
}

export interface LevyForecastInfo {
  data_points: LevyForecastPointInfo[];
  notes: string[];
  status: "healthy" | "watch" | "shortfall_risk" | "insufficient_data";
  confidence: "low" | "medium" | "high";
  months_projected: number;
  current_monthly_levy_cents: number;
  average_collection_rate_pct: number;
  average_monthly_income_cents: number;
  average_monthly_expense_cents: number;
  projected_reserve_balance_cents: number;
  projected_shortfall_cents: number;
  recommended_monthly_increase_cents: number;
  recommended_increase_pct: number;
}

export interface FinancialDashboard {
  reserve_fund?: ReserveFundInfo | null;
  levy_summary?: LevySummaryInfo | null;
  budget_lines: BudgetLineInfo[];
  available_periods: string[];
  role: string;
  selected_period: string;
  total_budgeted_cents: number;
  total_actual_cents: number;
  surplus_cents: number;
  levy_forecast?: LevyForecastInfo | null;
}
