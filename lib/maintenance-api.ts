"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { MaintenanceDashboard, MaintenanceRequestInfo } from "@/lib/maintenance";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getMaintenanceDashboard(
  schemeId: string,
): Promise<MaintenanceDashboard> {
  return parse(
    await apiFetch(`/api/v1/maintenance/${schemeId}`),
    "Failed to load maintenance requests",
  );
}

export async function createMaintenanceRequest(
  schemeId: string,
  input: {
    title: string;
    description: string;
    category: string;
  },
): Promise<MaintenanceRequestInfo> {
  return parse(
    await apiFetch(`/api/v1/maintenance/${schemeId}`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to create maintenance request",
  );
}

export async function assignMaintenanceRequest(
  schemeId: string,
  requestId: string,
  input: {
    contractor_name: string;
    contractor_phone?: string | null;
  },
): Promise<MaintenanceRequestInfo> {
  return parse(
    await apiFetch(`/api/v1/maintenance/${schemeId}/${requestId}/assign`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to assign contractor",
  );
}

export async function resolveMaintenanceRequest(
  schemeId: string,
  requestId: string,
): Promise<MaintenanceRequestInfo> {
  return parse(
    await apiFetch(`/api/v1/maintenance/${schemeId}/${requestId}/resolve`, {
      method: "POST",
    }),
    "Failed to resolve maintenance request",
  );
}
