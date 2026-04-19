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

  it("uses sh_refresh to rebuild the session via shared helper", async () => {
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
                session: {
                  id: "user-1",
                  email: "person@example.com",
                  full_name: "Person Example",
                  phone: null,
                  role: "admin",
                  wizard_complete: true,
                  scheme_memberships: [],
                },
              },
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }

        return new Response("{}", { status: 404 });
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
  });

  it("unwraps the backend data envelope before shaping the session", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        if (url.includes("/api/v1/auth/refresh")) {
          return new Response(
            JSON.stringify({
              data: {
                access_token: "new-access-token",
                refresh_token: "new-refresh-token",
                session: {
                  id: "user-1",
                  email: "person@example.com",
                  full_name: "Person Example",
                  phone: null,
                  role: "admin",
                  wizard_complete: true,
                  scheme_memberships: [
                    { scheme_id: "scheme-1", scheme_name: "Scheme 1", role: "admin" },
                  ],
                  org: { id: "org-1", name: "Org 1" },
                },
              },
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response("{}", { status: 404 });
      }),
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
      scheme_memberships: [
        { scheme_id: "scheme-1", scheme_name: "Scheme 1", role: "admin" },
      ],
      org: { id: "org-1", name: "Org 1" },
    });
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_session",
      expect.stringContaining("%22id%22%3A%22user-1%22"),
      expect.any(Object),
    );
  });

  it("writes sh_session with httpOnly and expected payload shape", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              access_token: "new-access-token",
              refresh_token: "new-refresh-token",
              session: {
                id: "user-1",
                email: "person@example.com",
                full_name: "Person Example",
                role: "admin",
                wizard_complete: true,
                scheme_memberships: [],
              },
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const { POST } = await import("./route");
    await POST();

    expect(cookieSet).toHaveBeenCalledWith(
      "sh_session",
      expect.stringContaining("%22role%22"),
      expect.objectContaining({ httpOnly: true }),
    );
  });

  it("writes rotated access and refresh tokens with httpOnly", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "old-access-token" };
      if (name === "sh_refresh") return { value: "old-refresh-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              access_token: "new-access-token",
              refresh_token: "new-refresh-token",
              session: {
                id: "user-1",
                email: "person@example.com",
                full_name: "Person Example",
                role: "admin",
                wizard_complete: true,
                scheme_memberships: [],
              },
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const { POST } = await import("./route");
    await POST();

    expect(cookieSet).toHaveBeenCalledWith(
      "sh_access",
      "new-access-token",
      expect.objectContaining({ httpOnly: true }),
    );
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_refresh",
      "new-refresh-token",
      expect.objectContaining({ httpOnly: true }),
    );
  });

  it("returns 503 with a shaped error when refresh is temporarily unavailable", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new DOMException("The operation was aborted", "AbortError");
      }),
    );

    const { POST } = await import("./route");
    const response = await POST();

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toEqual({
      error: {
        code: "UPSTREAM_UNAVAILABLE",
        message: "Temporary service issue. Please retry.",
      },
    });
  });
});
