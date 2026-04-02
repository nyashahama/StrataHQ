"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type {
  BillingCheckoutSession,
  BillingPortalSession,
  BillingSubscription,
} from "@/lib/billing";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getSubscription(): Promise<BillingSubscription> {
  return parse(
    await apiFetch("/api/v1/billing/subscription"),
    "Failed to load billing subscription",
  );
}

export async function createCheckoutSession(): Promise<BillingCheckoutSession> {
  return parse(
    await apiFetch("/api/v1/billing/checkout", {
      method: "POST",
    }),
    "Failed to create checkout session",
  );
}

export async function createPortalSession(): Promise<BillingPortalSession> {
  return parse(
    await apiFetch("/api/v1/billing/portal", {
      method: "POST",
    }),
    "Failed to create customer portal session",
  );
}
