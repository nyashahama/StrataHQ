export type MaintenanceCategory =
  | "plumbing"
  | "electrical"
  | "structural"
  | "garden"
  | "pool"
  | "other";

export type MaintenanceStatus =
  | "open"
  | "in_progress"
  | "pending_approval"
  | "resolved";

export interface MaintenanceRequestInfo {
  contractor_name?: string | null;
  contractor_phone?: string | null;
  resolved_at?: string | null;
  unit_id?: string | null;
  unit_identifier?: string | null;
  owner_name?: string | null;
  id: string;
  scheme_id: string;
  title: string;
  description: string;
  category: MaintenanceCategory;
  status: MaintenanceStatus;
  submitted_by_unit?: string | null;
  sla_hours: number;
  created_at: string;
  updated_at: string;
  sla_breached: boolean;
}

export interface MaintenanceDashboard {
  requests: MaintenanceRequestInfo[];
  role: string;
  open_count: number;
  sla_breached_count: number;
  pending_approval_count: number;
  resolved_this_month: number;
}
