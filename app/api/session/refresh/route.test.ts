import { beforeEach, describe, expect, it, vi } from "vitest";

const cookieSet = vi.fn();
const cookieGet = vi.fn();

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
    set: cookieSet,
  })),
}));

describe("POST /api/session/refresh", () => {
  beforeEach(() => {
    cookieSet.mockReset();
    cookieGet.mockReset();
    vi.restoreAllMocks();
  });

  it("uses sh_refresh, not sh_access, to rebuild the session", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "expired-access-token" };
      if (name === "sh_refresh") return { value: "valid-refresh-token" };
      return undefined;
    });

    const fetchCalls: { url: string; options: RequestInit }[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string, options: RequestInit) => {
        fetchCalls.push({ url, options: { ...options } });

        if (url.includes("/api/v1/auth/refresh")) {
          return new Response(
            JSON.stringify({
              data: {
                access_token: "new-access-token",
                refresh_token: "new-refresh-token",
              },
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }

        return new Response(
          JSON.stringify({
            data: {
              id: "user-1",
              email: "person@example.com",
              full_name: "Person Example",
              phone: null,
              role: "admin",
              wizard_complete: true,
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );

    const { POST } = await import("./route");
    const response = await POST();

    expect(response.status).toBe(200);

    const refreshCall = fetchCalls.find(call =>
      call.url.includes("/api/v1/auth/refresh"),
    );
    expect(refreshCall).toBeDefined();
    expect(refreshCall?.options.method).toBe("POST");

    const loginCall = fetchCalls.find(call =>
      call.url.includes("/api/v1/auth/me"),
    );
    expect(loginCall).toBeUndefined();
  });

  it("unwraps the backend data envelope before shaping the session", async () => {
    cookieGet.mockReturnValue({ value: "access-token" });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              id: "user-1",
              email: "person@example.com",
              full_name: "Person Example",
              phone: null,
              role: "admin",
              wizard_complete: true,
              scheme_memberships: [{ scheme_id: "scheme-1", role: "admin" }],
              org: { id: "org-1", name: "Org 1" },
            },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      ),
    );

    const { POST } = await import("./route");
    const response = await POST();

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      id: "user-1",
      email: "person@example.com",
      full_name: "Person Example",
      phone: null,
      role: "admin",
      wizard_complete: true,
      scheme_memberships: [{ scheme_id: "scheme-1", role: "admin" }],
      org: { id: "org-1", name: "Org 1" },
    });
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_session",
      expect.stringContaining("%22id%22%3A%22user-1%22"),
      expect.any(Object),
    );
  });
});
