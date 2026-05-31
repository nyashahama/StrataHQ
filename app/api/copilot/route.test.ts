import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: vi.fn((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      if (name === "sh_csrf") return { value: "csrf-token" };
      return undefined;
    }),
  })),
}));

describe("copilot route", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("returns 400 for malformed JSON without contacting the backend", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const { POST } = await import("./route");
    const request = new Request("http://localhost/api/copilot", {
      method: "POST",
      headers: { "Content-Type": "application/json", "x-csrf-token": "csrf-token" },
      body: "{not-json",
    });

    const response = await POST(
      request as unknown as import("next/server").NextRequest,
    );

    expect(response.status).toBe(400);
    await expect(response.text()).resolves.toContain("Invalid JSON");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("returns 503 when backend copilot request aborts", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new DOMException("aborted", "AbortError");
      }),
    );

    const { POST } = await import("./route");
    const request = new Request("http://localhost/api/copilot", {
      method: "POST",
      headers: { "Content-Type": "application/json", "x-csrf-token": "csrf-token" },
      body: JSON.stringify({ message: "hello", history: [] }),
    });

    const response = await POST(
      request as unknown as import("next/server").NextRequest,
    );

    expect(response.status).toBe(503);
    await expect(response.text()).resolves.toContain(
      "Copilot temporarily unavailable",
    );
  });
});
