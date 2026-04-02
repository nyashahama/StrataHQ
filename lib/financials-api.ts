"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { BudgetLineInfo, FinancialDashboard, ReserveFundInfo } from "@/lib/financials";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getFinancialDashboard(
  schemeId: string,
  period?: string,
): Promise<FinancialDashboard> {
  const search = period ? `?period=${encodeURIComponent(period)}` : "";
  return parse(
    await apiFetch(`/api/v1/financials/${schemeId}${search}`),
    "Failed to load financial dashboard",
  );
}

export async function upsertBudgetLine(
  schemeId: string,
  input: {
    category: string;
    period_label: string;
    budgeted_cents: number;
    actual_cents: number;
  },
): Promise<BudgetLineInfo> {
  return parse(
    await apiFetch(`/api/v1/financials/${schemeId}/budget-lines`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
    "Failed to save budget line",
  );
}

export async function updateReserveFund(
  schemeId: string,
  input: {
    balance_cents: number;
    target_cents: number;
  },
): Promise<ReserveFundInfo> {
  return parse(
    await apiFetch(`/api/v1/financials/${schemeId}/reserve-fund`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
    "Failed to update reserve fund",
  );
}
