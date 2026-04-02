"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";

export interface SchemeSummary {
  unit_id?: string | null;
  unit_identifier?: string | null;
  next_agm_date?: string | null;
  id: string;
  name: string;
  address: string;
  role: string;
  health: "good" | "fair" | "poor";
  unit_count: number;
  total_members: number;
  trustee_count: number;
  resident_count: number;
  levy_collection_pct: number;
  open_maintenance_count: number;
  notice_count: number;
  days_to_agm?: number | null;
}

export interface UnitInfo {
  id: string;
  identifier: string;
  owner_name: string;
  floor: number;
  section_value_pct: number;
}

export interface NoticeInfo {
  id: string;
  title: string;
  type: string;
  sent_at: string;
}

export interface SchemeDetail extends SchemeSummary {
  units: UnitInfo[];
  recent_notices: NoticeInfo[];
}

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function listSchemes(): Promise<SchemeSummary[]> {
  return parse(
    await apiFetch("/api/v1/schemes"),
    "Failed to load schemes",
  );
}

export async function getScheme(id: string): Promise<SchemeDetail> {
  return parse(
    await apiFetch(`/api/v1/schemes/${id}`),
    "Failed to load scheme",
  );
}

export async function updateScheme(
  id: string,
  input: { name: string; address: string; unit_count: number },
): Promise<SchemeSummary> {
  return parse(
    await apiFetch(`/api/v1/schemes/${id}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
    "Failed to update scheme",
  );
}

export async function listSchemeUnits(id: string): Promise<UnitInfo[]> {
  return parse(
    await apiFetch(`/api/v1/schemes/${id}/units`),
    "Failed to load units",
  );
}

export async function createSchemeUnit(
  schemeId: string,
  input: {
    identifier: string;
    owner_name: string;
    floor: number;
    section_value_bps: number;
  },
): Promise<UnitInfo> {
  return parse(
    await apiFetch(`/api/v1/schemes/${schemeId}/units`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to create unit",
  );
}

export async function updateSchemeUnit(
  schemeId: string,
  unitId: string,
  input: {
    identifier: string;
    owner_name: string;
    floor: number;
    section_value_bps: number;
  },
): Promise<UnitInfo> {
  return parse(
    await apiFetch(`/api/v1/schemes/${schemeId}/units/${unitId}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
    "Failed to update unit",
  );
}
