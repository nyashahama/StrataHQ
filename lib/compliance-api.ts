"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { ComplianceDashboard } from "@/lib/compliance";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getComplianceDashboard(
  schemeId: string,
): Promise<ComplianceDashboard> {
  return parse(
    await apiFetch(`/api/v1/compliance/${schemeId}`),
    "Failed to load compliance dashboard",
  );
}
