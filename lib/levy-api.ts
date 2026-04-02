"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type {
  LevyDashboard,
  LevyPeriodInfo,
  ReconcilePaymentInput,
  ReconcileResult,
} from "@/lib/levy";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getLevyDashboard(
  schemeId: string,
): Promise<LevyDashboard> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}`),
    "Failed to load levy dashboard",
  );
}

export async function createLevyPeriod(
  schemeId: string,
  input: { label: string; due_date: string; amount_cents: number },
): Promise<LevyPeriodInfo> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/periods`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to create levy period",
  );
}

export async function reconcileLevyPayments(
  schemeId: string,
  payments: ReconcilePaymentInput[],
): Promise<ReconcileResult> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/reconcile`, {
      method: "POST",
      body: JSON.stringify({ payments }),
    }),
    "Failed to reconcile levy payments",
  );
}
