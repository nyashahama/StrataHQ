"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { CommunicationsDashboard, NoticeInfo, NoticeType } from "@/lib/communications";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getCommunicationsDashboard(
  schemeId: string,
  type?: NoticeType | "all",
): Promise<CommunicationsDashboard> {
  const query = type && type !== "all" ? `?type=${encodeURIComponent(type)}` : "";
  return parse(
    await apiFetch(`/api/v1/communications/${schemeId}${query}`),
    "Failed to load notices",
  );
}

export async function createNotice(
  schemeId: string,
  input: {
    title: string;
    body: string;
    type: NoticeType;
  },
): Promise<NoticeInfo> {
  return parse(
    await apiFetch(`/api/v1/communications/${schemeId}`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to send notice",
  );
}
