"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import type { AuditEventsResponse } from "@/lib/audit";

export async function getSchemeAuditEvents(
  schemeId: string,
  limit = 50,
): Promise<AuditEventsResponse> {
  const response = await apiFetch(
    `/api/v1/audit/schemes/${schemeId}/events?limit=${limit}`,
  );
  if (!response.ok) {
    throw await buildApiHttpError(response, "Failed to load audit events");
  }
  return readApiData<AuditEventsResponse>(response);
}
