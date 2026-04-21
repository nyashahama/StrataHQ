import { beforeEach, describe, expect, it, vi } from "vitest";

const refreshAuthSession = vi.fn();
const clearAuthCookies = vi.fn();

vi.mock("@/lib/server-auth", () => ({
  refreshAuthSession,
  clearAuthCookies,
}));

const mockAccessToken = { current: "expired-access-token" };
let callCount = 0;

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: vi.fn((name: string) => {
      if (name === "sh_access") {
        return { value: mockAccessToken.current };
      }
      if (name === "sh_refresh") return { value: "valid-refresh-token" };
      return undefined;
    }),
    set: vi.fn(),
  })),
}));

describe("proxyRequest", () => {
  beforeEach(() => {
    mockAccessToken.current = "expired-access-token";
    callCount = 0;
    refreshAuthSession.mockReset();
    clearAuthCookies.mockReset();
    vi.restoreAllMocks();

    refreshAuthSession.mockImplementation(async () => {
      mockAccessToken.current = "new-access-token";
      return {
        kind: "success",
        session: { id: "1", email: "test@test.com" },
      };
    });
  });

  it("refreshes once and retries the upstream request after a 401", async () => {
    const calls: { url: string; options: RequestInit }[] = [];

    const fetchMock = vi.fn(async (url: string, options: RequestInit) => {
      callCount++;
      calls.push({ url, options: { ...options } });

      if (callCount === 1) {
        return new Response(null, { status: 401 });
      }

      return new Response(
        JSON.stringify({ data: { ok: true } }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    const { GET } = await import("./route");

    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(200);
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(clearAuthCookies).not.toHaveBeenCalled();

    const firstCall = calls[0];
    expect(firstCall.options.headers).toEqual(
      expect.objectContaining({
        Authorization: "Bearer expired-access-token",
      }),
    );

    const retryCall = calls[1];
    expect(retryCall.options.headers).toEqual(
      expect.objectContaining({
        Authorization: "Bearer new-access-token",
      }),
    );
  });

  it("does not retry when refresh fails", async () => {
    const fetchMock = vi.fn(async () => {
      return new Response(null, { status: 401 });
    });
    vi.stubGlobal("fetch", fetchMock);

    refreshAuthSession.mockResolvedValue({ kind: "invalid" });

    const { GET } = await import("./route");

    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(401);
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });

  it("returns 503 with shaped error when upstream times out", async () => {
    const fetchMock = vi.fn(async () => {
      throw new DOMException("The operation was aborted", "AbortError");
    });
    vi.stubGlobal("fetch", fetchMock);

    const { GET } = await import("./route");

    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(503);
    expect(await response.json()).toEqual({
      error: {
        code: "UPSTREAM_UNAVAILABLE",
        message: "Temporary service issue. Please retry.",
      },
    });
  });

  it("forwards the routed path segments and search string exactly once", async () => {
    const calls: string[] = [];
    vi.stubGlobal("fetch", vi.fn(async (url: string) => {
      calls.push(url);
      return new Response(JSON.stringify({ data: { ok: true } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }));

    const { GET } = await import("./route");

    const req = new Request(
      "http://localhost/api/proxy/api/v1/communications/scheme-1?type=agm",
    );
    const params = Promise.resolve({
      path: ["api", "v1", "communications", "scheme-1"],
    });

    await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(calls[0]).toBe("http://localhost:8080/api/v1/communications/scheme-1?type=agm");
  });

  it("reuses the original request body for a retry after refresh", async () => {
    let callCount = 0;
    const bodies: string[] = [];

    vi.stubGlobal("fetch", vi.fn(async (_url: string, options: RequestInit) => {
      callCount++;
      if (options.body) bodies.push(Buffer.from(options.body as Uint8Array).toString());
      if (callCount === 1) return new Response(null, { status: 401 });
      return new Response(JSON.stringify({ data: { ok: true } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }));

    const { POST } = await import("./route");

    const req = new Request("http://localhost/api/proxy/api/v1/invitations", {
      method: "POST",
      body: JSON.stringify({ full_name: "Test User" }),
    });
    const params = Promise.resolve({ path: ["api", "v1", "invitations"] });

    await POST(req as unknown as import("next/server").NextRequest, { params });

    expect(bodies).toEqual([
      JSON.stringify({ full_name: "Test User" }),
      JSON.stringify({ full_name: "Test User" }),
    ]);
  });

  it("returns 503 without clearing auth when refresh is temporarily unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(null, { status: 401 })));
    refreshAuthSession.mockResolvedValue({ kind: "unavailable" });

    const { GET } = await import("./route");
    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toEqual({
      error: {
        code: "UPSTREAM_UNAVAILABLE",
        message: "Temporary service issue. Please retry.",
      },
    });
    expect(clearAuthCookies).not.toHaveBeenCalled();
  });

  it("returns 503 when the post-refresh retry aborts", async () => {
    let attempt = 0;
    vi.stubGlobal("fetch", vi.fn(async () => {
      attempt++;
      if (attempt === 1) return new Response(null, { status: 401 });
      throw new DOMException("The operation was aborted", "AbortError");
    }));
    refreshAuthSession.mockResolvedValue({
      kind: "success",
      session: { id: "1", email: "test@test.com" },
    });
    mockAccessToken.current = "new-access-token";

    const { GET } = await import("./route");
    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toEqual({
      error: {
        code: "UPSTREAM_UNAVAILABLE",
        message: "Temporary service issue. Please retry.",
      },
    });
    expect(clearAuthCookies).not.toHaveBeenCalled();
  });

  it("adds server timing and upstream status headers", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Response(JSON.stringify({ data: { ok: true } }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    const { GET } = await import("./route");
    const req = new Request("http://localhost/api/proxy/api/v1/schemes");
    const params = Promise.resolve({ path: ["api", "v1", "schemes"] });

    const response = await GET(
      req as unknown as import("next/server").NextRequest,
      { params },
    );

    expect(response.headers.get("server-timing")).toContain("upstream;dur=");
    expect(response.headers.get("x-upstream-status")).toBe("200");
    expect(response.headers.get("x-request-id")).toBeTruthy();
  });
});
