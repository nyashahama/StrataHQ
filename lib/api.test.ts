import { beforeEach, describe, expect, it, vi } from "vitest";

describe("apiFetch", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("does not refresh tokens for public requests that return 401", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);

    const { apiFetch } = await import("./api");
    const response = await apiFetch(
      "/api/v1/invitations/verify/token-123",
      { auth: false } as RequestInit & { auth?: boolean },
    );

    expect(response.status).toBe(401);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const requestInit = fetchMock.mock.calls[0][1] as RequestInit;
    const headers = new Headers(requestInit.headers);
    expect(headers.get("x-skip-auth")).toBe("true");
  });

  it("adds the csrf header to mutating proxy requests", async () => {
    Object.defineProperty(document, "cookie", {
      configurable: true,
      value: "sh_csrf=test-csrf-token",
    });

    const fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    const { apiFetch } = await import("./api");
    await apiFetch("/api/v1/schemes/scheme-1", {
      method: "POST",
      body: JSON.stringify({ name: "Scheme" }),
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/proxy/api/v1/schemes/scheme-1",
      expect.anything(),
    );

    const requestInit = fetchMock.mock.calls[0][1] as RequestInit;
    const headers = new Headers(requestInit.headers);
    expect(headers.get("x-csrf-token")).toBe("test-csrf-token");
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("does not set JSON content type for multipart body uploads", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    const form = new FormData();
    form.append("bank", "chase");
    form.append("file", new File(["abc"], "statement.csv", { type: "text/csv" }));

    const { apiFetch } = await import("./api");
    await apiFetch("/api/v1/levies/scheme-1/reconcile/imports", {
      method: "POST",
      body: form,
    });

    const callArgs = fetchMock.mock.calls[0];
    const requestInit = callArgs[1] as RequestInit;
    const headers = new Headers(requestInit.headers);

    expect(headers.has("Content-Type")).toBe(false);
  });
});
