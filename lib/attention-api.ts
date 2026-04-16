"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { AttentionItem, CollectionEvent, ReminderDraft } from "@/lib/attention";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getPortfolioAttentionQueue(): Promise<{
  items: AttentionItem[];
  scope: "portfolio";
}> {
  return parse(await apiFetch("/api/v1/levies/attention"), "Failed to load portfolio queue");
}

export async function getSchemeAttentionQueue(
  schemeId: string
): Promise<{ items: AttentionItem[]; scope: "scheme" }> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/attention`),
    "Failed to load scheme queue"
  );
}

export async function listCollectionEvents(
  schemeId: string,
  accountId: string
): Promise<CollectionEvent[]> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/accounts/${accountId}/events`),
    "Failed to load collection events"
  );
}

export async function createCollectionEvent(
  schemeId: string,
  accountId: string,
  input: {
    event_type:
      | "reminder_sent"
      | "follow_up_logged"
      | "promise_to_pay"
      | "legal_review_flagged";
    note?: string;
    promise_amount_cents?: number;
    promise_date?: string;
  }
): Promise<CollectionEvent> {
  return parse(
    await apiFetch(
      `/api/v1/levies/${schemeId}/accounts/${accountId}/events`,
      {
        method: "POST",
        body: JSON.stringify(input),
      }
    ),
    "Failed to save collection action"
  );
}

export async function getCollectionReminderDraft(
  schemeId: string,
  accountId: string,
): Promise<ReminderDraft> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/accounts/${accountId}/reminder-draft`),
    "Failed to load reminder draft",
  );
}

export async function sendCollectionReminder(
  schemeId: string,
  accountId: string,
  input: {
    email: { enabled: boolean; subject?: string; body: string };
    whatsapp: { enabled: boolean; body: string };
  },
): Promise<CollectionEvent> {
  return parse(
    await apiFetch(`/api/v1/levies/${schemeId}/accounts/${accountId}/reminders`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to send reminder",
  );
}