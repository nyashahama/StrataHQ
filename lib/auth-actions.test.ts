import { beforeEach, describe, expect, it, vi } from "vitest";

const cookieGet = vi.fn();
const cookieSet = vi.fn();
const writeAuthCookies = vi.fn();
const clearAuthCookies = vi.fn();

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
    set: cookieSet,
  })),
}));

vi.mock("./server-auth", () => ({
  writeAuthCookies,
  clearAuthCookies,
}));

describe("auth actions", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    cookieGet.mockReset();
    cookieSet.mockReset();
    writeAuthCookies.mockReset();
    clearAuthCookies.mockReset();
  });

  it("surfaces the backend onboarding error message from setupAction", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            error: {
              code: "BAD_REQUEST",
              message: "Scheme address is required",
            },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const { setupAction } = await import("./auth-actions");
    const result = await setupAction({
      org_name: "Org",
      contact_email: "person@example.com",
      scheme_name: "Scheme",
      scheme_address: "",
      unit_count: 10,
    });

    expect(result).toEqual({ error: "Scheme address is required" });
  });

  it("applies a timeout signal when accepting an invite", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          data: {
            access_token: "access-token",
            refresh_token: "refresh-token",
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
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const { acceptInviteAction } = await import("./auth-actions");
    const result = await acceptInviteAction("invite-token", "Password_123");

    expect(result).toEqual({
      user: {
        id: "user-1",
        email: "person@example.com",
        full_name: "Person Example",
        phone: null,
        role: "admin",
        wizard_complete: true,
        scheme_memberships: [],
        org: { id: "org-1", name: "Org 1" },
      },
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/invitations/verify/invite-token/accept",
      expect.objectContaining({
        method: "POST",
        signal: expect.any(AbortSignal),
      }),
    );
    expect(writeAuthCookies).toHaveBeenCalledWith({
      access_token: "access-token",
      refresh_token: "refresh-token",
      session: expect.objectContaining({ id: "user-1" }),
    });
  });
});
