export interface AuditEventInfo {
  id: string;
  scheme_id: string;
  org_id: string;
  actor_user_id: string | null;
  actor_role: string;
  resource_type: string;
  resource_id: string | null;
  action: string;
  before_state: Record<string, unknown> | null;
  after_state: Record<string, unknown> | null;
  metadata: Record<string, unknown> | null;
  occurred_at: string;
}

export interface AuditEventsResponse {
  events: AuditEventInfo[];
  total: number;
  limit: number;
}
