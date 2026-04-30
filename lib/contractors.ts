import type { MaintenanceCategory } from "@/lib/maintenance";

export interface ContractorInfo {
  phone?: string | null;
  email?: string | null;
  notes?: string | null;
  id: string;
  org_id: string;
  name: string;
  trade: MaintenanceCategory;
  suburb: string;
  city: string;
  province: string;
  scheme_ids: string[];
  average_rating: number;
  review_count: number;
  completed_job_count: number;
  created_at: string;
  updated_at: string;
  public_profile: boolean;
  vetted: boolean;
  active: boolean;
  preferred: boolean;
}

export interface ContractorReviewInfo {
  comment?: string | null;
  id: string;
  contractor_id: string;
  scheme_id: string;
  maintenance_request_id: string;
  created_by_user_id: string;
  rating: number;
  created_at: string;
}

export interface ContractorUpsertInput {
  phone?: string | null;
  email?: string | null;
  notes?: string | null;
  name: string;
  trade: MaintenanceCategory;
  suburb: string;
  city?: string;
  province?: string;
  scheme_ids: string[];
  public_profile: boolean;
  vetted: boolean;
  active: boolean;
}
