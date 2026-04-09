import { describe, expect, it } from "vitest";

import { buildAllowedBackendProxyPath } from "@/lib/backend-proxy";

describe("buildAllowedBackendProxyPath", () => {
  it("allows backend api v1 paths", () => {
    expect(
      buildAllowedBackendProxyPath(["api", "v1", "documents", "abc"], "?category=rules"),
    ).toBe("/api/v1/documents/abc?category=rules");
  });

  it("rejects non-api-v1 paths", () => {
    expect(buildAllowedBackendProxyPath(["admin", "metrics"], "")).toBeNull();
  });

  it("rejects traversal segments", () => {
    expect(buildAllowedBackendProxyPath(["api", "v1", "..", "secrets"], "")).toBeNull();
  });
});
