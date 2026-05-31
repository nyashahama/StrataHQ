import { beforeEach, describe, expect, it, vi } from "vitest";

const refreshAuthSession = vi.fn();
const clearAuthCookies = vi.fn();

vi.mock("@/lib/server-auth", () => ({
  refreshAuthSession,
  clearAuthCookies,
}));

const mockAccessToken = { current: "expired-access-token" };

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: vi.fn((name: string) => {
      if (name === "sh_access") {
        return { value: mockAccessToken.current };
      }
      return undefined;
    }),
  })),
}));

function mockRefreshSuccess(): void {
  refreshAuthSession.mockImplementation(async () => {
    mockAccessToken.current = "new-access-token";
    return {
      kind: "success",
      session: {
        id: "user-1",
        email: "admin@example.com",
        full_name: "Admin User",
        phone: null,
        role: "admin",
        wizard_complete: true,
        scheme_memberships: [],
        org: { id: "org-1", name: "Org 1" },
      },
    };
  });
}

describe("early access admin API", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    refreshAuthSession.mockReset();
    clearAuthCookies.mockReset();
    mockAccessToken.current = "expired-access-token";
  });

  it("retries the admin request list after refreshing an expired access token", async () => {
    mockRefreshSuccess();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "request-1",
                full_name: "Person Example",
                email: "person@example.com",
                scheme_name: "Example Scheme",
                unit_count: 12,
                status: "pending",
                created_at: "2026-05-31T10:00:00Z",
              },
            ],
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    const { listEarlyAccessRequests } = await import("./early-access-api");
    const requests = await listEarlyAccessRequests();

    expect(requests).toHaveLength(1);
    expect(requests[0]?.id).toBe("request-1");
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/admin/early-access",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer new-access-token",
        }),
      }),
    );
  });

  it.each([
    ["approve", "approveEarlyAccessRequest"],
    ["reject", "rejectEarlyAccessRequest"],
  ] as const)("retries %s after refreshing an expired access token", async (action, exportName) => {
    mockRefreshSuccess();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    const api = await import("./early-access-api");
    const result = await api[exportName]("request-1");

    expect(result).toEqual({ ok: true });
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      `http://localhost:8080/api/v1/admin/early-access/request-1/${action}`,
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer new-access-token",
        }),
      }),
    );
  });

  it("clears auth cookies when refresh recovery is rejected", async () => {
    refreshAuthSession.mockResolvedValue({ kind: "invalid" });
    vi.stubGlobal("fetch", vi.fn(async () => new Response(null, { status: 401 })));

    const { approveEarlyAccessRequest } = await import("./early-access-api");
    await expect(approveEarlyAccessRequest("request-1")).resolves.toEqual({
      error: "Not authenticated",
    });

    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });
});
