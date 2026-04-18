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
      return { id: "1", email: "test@test.com" };
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

    refreshAuthSession.mockResolvedValue(null);

    const { GET } = await import("./route");

    const req = new Request("http://localhost/api/proxy/api/v1/some/endpoint");
    const params = Promise.resolve({ path: ["api", "v1", "some", "endpoint"] });

    const response = await GET(req as unknown as import("next/server").NextRequest, { params });

    expect(response.status).toBe(401);
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });
});