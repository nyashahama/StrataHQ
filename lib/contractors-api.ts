"use client";

import { apiFetch } from "@/lib/api";
import { buildApiHttpError, readApiData } from "@/lib/api-contract";
import type { ContractorInfo, ContractorReviewInfo, ContractorUpsertInput } from "@/lib/contractors";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw await buildApiHttpError(response, fallback);
  }
  return readApiData<T>(response);
}

export async function listContractors(params: {
  scheme_id?: string;
  trade?: string;
  suburb?: string;
  q?: string;
  active?: boolean;
  vetted?: boolean;
} = {}): Promise<ContractorInfo[]> {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== "") search.set(key, String(value));
  });
  return parse(
    await apiFetch(`/api/v1/contractors${search.size ? `?${search}` : ""}`),
    "Failed to load contractors",
  );
}

export async function searchContractorMarketplace(params: {
  scheme_id: string;
  trade?: string;
  suburb?: string;
}): Promise<ContractorInfo[]> {
  const search = new URLSearchParams({ scheme_id: params.scheme_id });
  if (params.trade) search.set("trade", params.trade);
  if (params.suburb) search.set("suburb", params.suburb);
  return parse(
    await apiFetch(`/api/v1/contractors/marketplace?${search}`),
    "Failed to search contractor marketplace",
  );
}

export async function createContractor(input: ContractorUpsertInput): Promise<ContractorInfo> {
  return parse(
    await apiFetch("/api/v1/contractors", { method: "POST", body: JSON.stringify(input) }),
    "Failed to create contractor",
  );
}

export async function updateContractor(id: string, input: ContractorUpsertInput): Promise<ContractorInfo> {
  return parse(
    await apiFetch(`/api/v1/contractors/${id}`, { method: "PATCH", body: JSON.stringify(input) }),
    "Failed to update contractor",
  );
}

export async function createContractorReview(
  contractorId: string,
  input: { scheme_id: string; maintenance_request_id: string; rating: number; comment?: string | null },
): Promise<ContractorReviewInfo> {
  return parse(
    await apiFetch(`/api/v1/contractors/${contractorId}/reviews`, { method: "POST", body: JSON.stringify(input) }),
    "Failed to create contractor review",
  );
}
