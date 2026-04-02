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
}
