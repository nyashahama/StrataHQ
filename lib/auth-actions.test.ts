import { beforeEach, describe, expect, it, vi } from "vitest";

const cookieGet = vi.fn();
const cookieSet = vi.fn();
const writeAuthCookies = vi.fn();
const clearAuthCookies = vi.fn();

// Mirror the real withAuthRetry: pass the access token from the cookie to the
// caller, retry once if the response is 401 by re-reading the cookie. For
// these tests the retry branch is never exercised, so this is just a thin
// pass-through to the supplied call function.
async function withAuthRetry(
  call: (accessToken: string) => Promise<Response>,
): Promise<
  | { kind: "ok"; response: Response; retried: boolean }
  | { kind: "unauthorized" }
  | { kind: "unavailable" }
> {
  const accessToken = cookieGet("sh_access")?.value;
  if (!accessToken) return { kind: "unauthorized" };
  const response = await call(accessToken);
  return { kind: "ok", response, retried: false };
}

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
    set: cookieSet,
  })),
}));

vi.mock("./server-auth", () => ({
  writeAuthCookies,
  clearAuthCookies,
  withAuthRetry,
}));

function encodedSession(overrides: Record<string, unknown> = {}): string {
  return encodeURIComponent(
    JSON.stringify({
      id: "user-1",
      email: "person@example.com",
      full_name: "Person Example",
      phone: null,
      role: "admin",
      wizard_complete: false,
      scheme_memberships: [],
      org: { id: "org-1", name: "" },
      ...overrides,
    }),
  );
}

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
      if (name === "sh_session") return { value: encodedSession() };
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

  it.each([
    ["missing", undefined],
    ["corrupt", "not-json"],
  ] as const)("does not call onboarding setup when the session cookie is %s", async (_name, rawSession) => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      if (name === "sh_session" && rawSession !== undefined) return { value: rawSession };
      return undefined;
    });
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          data: {
            org: { id: "org-1", name: "Org 1" },
            scheme: { id: "scheme-1", name: "Scheme 1" },
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const { setupAction } = await import("./auth-actions");
    const result = await setupAction({
      org_name: "Org",
      contact_email: "person@example.com",
      scheme_name: "Scheme",
      scheme_address: "1 Main Road",
      unit_count: 10,
    });

    expect(result).toEqual({ error: "Session expired — please log in again" });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });

  it("updates a valid setup session from the backend setup response", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      if (name === "sh_session") return { value: encodedSession() };
      return undefined;
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              org: {
                id: "org-1",
                name: "Org 1",
                contact_email: "person@example.com",
                contact_phone: null,
              },
              scheme: { id: "scheme-1", name: "Scheme 1" },
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const { setupAction } = await import("./auth-actions");
    const result = await setupAction({
      org_name: "Org 1",
      contact_email: "person@example.com",
      scheme_name: "Scheme 1",
      scheme_address: "1 Main Road",
      unit_count: 10,
    });

    expect(result).toEqual({
      user: expect.objectContaining({
        id: "user-1",
        wizard_complete: true,
        org: expect.objectContaining({ id: "org-1", name: "Org 1" }),
        scheme_memberships: [
          expect.objectContaining({
            scheme_id: "scheme-1",
            scheme_name: "Scheme 1",
            role: "admin",
          }),
        ],
      }),
    });
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_session",
      expect.any(String),
      expect.objectContaining({ httpOnly: true }),
    );
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

  it("awaits the forgot-password request before returning", async () => {
    let resolveFetch: (value: Response) => void;
    const fetchPromise = new Promise<Response>((resolve) => {
      resolveFetch = resolve;
    });
    const fetchMock = vi.fn(() => fetchPromise);
    vi.stubGlobal("fetch", fetchMock);

    const { forgotPasswordAction } = await import("./auth-actions");
    let resolved = false;
    const action = forgotPasswordAction("person@example.com").then(() => {
      resolved = true;
    });

    await Promise.resolve();
    expect(resolved).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    resolveFetch!(new Response(null, { status: 200 }));
    await action;
    expect(resolved).toBe(true);
  });

  it("logout sends a timeout signal and clears cookies on the backend call", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });
    const fetchMock = vi.fn(async (_url: string, init: RequestInit) => {
      expect(init.signal).toBeInstanceOf(AbortSignal);
      return new Response(null, { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const { logoutAction } = await import("./auth-actions");
    await logoutAction();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });

  it("logout still clears cookies when the backend logout call fails", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });
    const fetchMock = vi.fn(async () => {
      throw new Error("network");
    });
    vi.stubGlobal("fetch", fetchMock);

    const { logoutAction } = await import("./auth-actions");
    await logoutAction();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });

  it("logout clears cookies when no refresh token is present", async () => {
    cookieGet.mockImplementation(() => undefined);
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const { logoutAction } = await import("./auth-actions");
    await logoutAction();

    expect(fetchMock).not.toHaveBeenCalled();
    expect(clearAuthCookies).toHaveBeenCalledTimes(1);
  });
});
