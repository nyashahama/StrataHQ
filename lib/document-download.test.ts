import { describe, expect, it } from "vitest";

import { getSafeDocumentDownloadHref } from "@/lib/document-download";

describe("getSafeDocumentDownloadHref", () => {
  it("allows matching data URLs", () => {
    expect(
      getSafeDocumentDownloadHref({
        file_type: "pdf",
        storage_key: "data:application/pdf;base64,VEVTVA==",
      }),
    ).toBe("data:application/pdf;base64,VEVTVA==");
  });

  it("allows root-relative paths", () => {
    expect(
      getSafeDocumentDownloadHref({
        file_type: "pdf",
        storage_key: "/documents/test.pdf?download=1",
      }),
    ).toBe("/documents/test.pdf?download=1");
  });

  it("rejects javascript URLs", () => {
    expect(
      getSafeDocumentDownloadHref({
        file_type: "pdf",
        storage_key: "javascript:alert(1)",
      }),
    ).toBeNull();
  });

  it("rejects mismatched data URLs", () => {
    expect(
      getSafeDocumentDownloadHref({
        file_type: "pdf",
        storage_key: "data:text/html;base64,PHNjcmlwdD4=",
      }),
    ).toBeNull();
  });
});
