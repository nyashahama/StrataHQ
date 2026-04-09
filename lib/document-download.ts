import type { DocumentFileType, SchemeDocumentInfo } from "@/lib/documents";

const DATA_URL_PREFIXES: Record<DocumentFileType, string[]> = {
  pdf: ["data:application/pdf;base64,"],
  docx: [
    "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,",
  ],
  xlsx: [
    "data:application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;base64,",
  ],
  jpg: ["data:image/jpeg;base64,", "data:image/jpg;base64,"],
  png: ["data:image/png;base64,"],
};

export function getSafeDocumentDownloadHref(
  document: Pick<SchemeDocumentInfo, "file_type" | "storage_key">,
): string | null {
  const storageKey = document.storage_key.trim();
  if (!storageKey) return null;

  const normalized = storageKey.toLowerCase();
  if (DATA_URL_PREFIXES[document.file_type].some(prefix => normalized.startsWith(prefix))) {
    return storageKey;
  }

  if (!storageKey.startsWith("/") || storageKey.startsWith("//")) {
    return null;
  }
  if (/[\r\n\t\\]/.test(storageKey)) {
    return null;
  }

  try {
    const parsed = new URL(storageKey, "https://app.local");
    if (parsed.origin !== "https://app.local") {
      return null;
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return null;
  }
}
