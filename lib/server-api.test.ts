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

describe("fetchBackendJson", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    refreshAuthSession.mockReset();
    clearAuthCookies.mockReset();
    mockAccessToken.current = "expired-access-token";
  });

  it("retries a server-side backend read once after refresh", async () => {
    refreshAuthSession.mockImplementation(async () => {
      mockAccessToken.current = "new-access-token";
      return {
        kind: "success",
        session: {
          id: "user-1",
          email: "person@example.com",
          full_name: "Person Example",
          phone: null,
          role: "admin",
          wizard_complete: true,
          scheme_memberships: [],
          org: { id: "org-1", name: "Org 1" },
        },
      };
    });

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 401 }))
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ data: [{ id: "scheme-1" }] }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    const { fetchBackendJson } = await import("./server-api");
    const data = await fetchBackendJson<Array<{ id: string }>>("/api/v1/schemes");

    expect(data).toEqual([{ id: "scheme-1" }]);
    expect(refreshAuthSession).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
