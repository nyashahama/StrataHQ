export interface AttentionItem {
  levy_account_id: string;
  scheme_id: string;
  scheme_name: string;
  unit_id: string;
  unit_identifier: string;
  owner_name: string;
  outstanding_cents: number;
  days_overdue: number;
  risk_score: number;
  score_drivers: string[];
  recommended_action: "reminder_sent" | "follow_up_logged" | "promise_to_pay" | "legal_review_flagged";
}

export interface CollectionEvent {
  id: string;
  levy_account_id: string;
  scheme_id: string;
  actor_role: string;
  event_type: string;
  note?: string | null;
  promise_amount_cents?: number | null;
  promise_date?: string | null;
  created_at: string;
}