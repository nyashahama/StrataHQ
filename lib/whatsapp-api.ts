"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type {
  WhatsAppBroadcast,
  WhatsAppBroadcastType,
  WhatsAppDashboard,
} from "@/lib/whatsapp";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getWhatsAppDashboard(
  schemeId: string,
): Promise<WhatsAppDashboard> {
  return parse(
    await apiFetch(`/api/v1/whatsapp/${schemeId}`),
    "Failed to load WhatsApp dashboard",
  );
}

export async function createWhatsAppBroadcast(
  schemeId: string,
  input: {
    message: string;
    type: WhatsAppBroadcastType;
  },
): Promise<WhatsAppBroadcast> {
  return parse(
    await apiFetch(`/api/v1/whatsapp/${schemeId}/broadcasts`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to send WhatsApp broadcast",
  );
}
