export type LevyStatus = "paid" | "partial" | "overdue" | "pending";

export interface LevyPeriodInfo {
  id: string;
  scheme_id: string;
  label: string;
  due_date: string;
  amount_cents: number;
  created_at: string;
}

export interface LevyAccountInfo {
  paid_date?: string | null;
  id: string;
  unit_id: string;
  unit_identifier: string;
  owner_name: string;
  period_id: string;
  amount_cents: number;
  paid_cents: number;
  status: LevyStatus;
  due_date: string;
}

export interface LevyPaymentInfo {
  bank_ref?: string | null;
  id: string;
  levy_account_id: string;
  amount_cents: number;
  payment_date: string;
  reference: string;
  created_at: string;
}

export interface CollectionTrendPoint {
  label: string;
  pct: number;
}

export interface LevyDashboard {
  current_period?: LevyPeriodInfo | null;
  my_account?: LevyAccountInfo | null;
  collection_trend: CollectionTrendPoint[];
  levy_roll: LevyAccountInfo[];
  my_payments: LevyPaymentInfo[];
  role: string;
  collection_rate_pct: number;
  overdue_count: number;
  total_collected_cents: number;
}

export interface ReconcilePaymentInput {
  account_id: string;
  payment_date: string;
  reference: string;
  bank_ref?: string | null;
  amount_cents: number;
}

export interface ReconcileResult {
  updated_account_ids: string[];
  applied_count: number;
  skipped_count: number;
}
