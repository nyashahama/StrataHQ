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

export interface ReminderChannelDraft {
  enabled: boolean;
  to: string;
  subject?: string;
  body: string;
  disabled_reason?: string;
}

export interface ReminderDraft {
  account_id: string;
  scheme_id: string;
  scheme_name: string;
  unit_label: string;
  owner_name: string;
  email: ReminderChannelDraft;
  whatsapp: ReminderChannelDraft;
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
  email_status?: "sent" | "failed" | "skipped" | null;
  email_body?: string | null;
  whatsapp_status?: "sent" | "failed" | "skipped" | null;
  whatsapp_body?: string | null;
  created_at: string;
}