"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type {
  DocumentCategory,
  DocumentsDashboard,
  SchemeDocumentInfo,
} from "@/lib/documents";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getDocumentsDashboard(
  schemeId: string,
  category?: DocumentCategory | "all",
): Promise<DocumentsDashboard> {
  const query =
    category && category !== "all"
      ? `?category=${encodeURIComponent(category)}`
      : "";
  return parse(
    await apiFetch(`/api/v1/documents/${schemeId}${query}`),
    "Failed to load documents",
  );
}

export async function createDocument(
  schemeId: string,
  input: {
    name: string;
    storage_key: string;
    file_type: string;
    category: string;
    size_bytes: number;
  },
): Promise<SchemeDocumentInfo> {
  return parse(
    await apiFetch(`/api/v1/documents/${schemeId}`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to upload document",
  );
}

export async function deleteDocument(
  schemeId: string,
  documentId: string,
): Promise<void> {
  const response = await apiFetch(`/api/v1/documents/${schemeId}/${documentId}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    throw new Error(await readApiError(response, "Failed to delete document"));
  }
}
