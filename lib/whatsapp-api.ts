"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import type {
  WhatsAppBroadcast,
  WhatsAppBroadcastType,
  WhatsAppDashboard,
  WhatsAppMaintenanceIntake,
} from "@/lib/whatsapp";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw await buildApiHttpError(response, fallback);
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

export async function createMaintenanceRequestFromWhatsAppMessage(
  schemeId: string,
  messageId: string,
  input: {
    title: string;
    description: string;
    category: WhatsAppMaintenanceIntake["category"];
  },
): Promise<WhatsAppMaintenanceIntake> {
  return parse(
    await apiFetch(`/api/v1/whatsapp/${schemeId}/messages/${messageId}/maintenance-request`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to create maintenance request from WhatsApp message",
  );
}

export async function dismissWhatsAppMaintenanceIntake(
  schemeId: string,
  intakeId: string,
): Promise<WhatsAppMaintenanceIntake> {
  return parse(
    await apiFetch(`/api/v1/whatsapp/${schemeId}/maintenance-intakes/${intakeId}`, {
      method: "PATCH",
      body: JSON.stringify({ status: "dismissed" }),
    }),
    "Failed to dismiss WhatsApp maintenance intake",
  );
}
