"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import type {
  LevyDashboard,
  LevyPeriodInfo,
  ReconcilePaymentInput,
  ReconcileResult,
  BankStatementImportResponse,
  BankStatementManualMatchInput,
} from "@/lib/levy";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw await buildApiHttpError(response, fallback);
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

export async function importBankStatementCsv(
  schemeId: string,
  bank: string,
  file: File,
): Promise<BankStatementImportResponse> {
  const form = new FormData()
  form.append("bank", bank)
  form.append("file", file)
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/reconcile/imports`, {
      method: "POST",
      body: form,
    }),
    "Failed to import bank statement",
  )
}

export async function getBankStatementImport(
  schemeId: string,
  importId: string,
): Promise<BankStatementImportResponse> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/reconcile/imports/${importId}`),
    "Failed to load bank statement import",
  )
}

export async function applyBankStatementImport(
  schemeId: string,
  importId: string,
  manualMatches: BankStatementManualMatchInput[],
): Promise<BankStatementImportResponse> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/reconcile/imports/${importId}/apply`, {
      method: "POST",
      body: JSON.stringify({ manual_matches: manualMatches }),
    }),
    "Failed to apply bank statement import",
  )
}
