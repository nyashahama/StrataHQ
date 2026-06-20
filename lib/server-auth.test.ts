import { beforeEach, describe, expect, it, vi } from "vitest";
import { refreshAuthSession, withAuthRetry } from "./server-auth";

const cookieSet = vi.fn();
const cookieDelete = vi.fn();
const cookieGet = vi.fn();

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
    set: cookieSet,
    delete: cookieDelete,
  })),
}));

describe("withAuthRetry", () => {
  beforeEach(() => {
    cookieSet.mockReset();
    cookieDelete.mockReset();
    cookieGet.mockReset();
    vi.restoreAllMocks();
  });

  it("returns unauthorized when no access token is present", async () => {
    cookieGet.mockReturnValue(undefined);

    const result = await withAuthRetry(vi.fn());

    expect(result).toEqual({ kind: "unauthorized" });
  });

  it("returns the first response without retrying when it isn't 401", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "access-token" };
      return undefined;
    });
    const call = vi.fn(async () => new Response("ok", { status: 200 }));

    const result = await withAuthRetry(call);

    expect(result.kind).toBe("ok");
    if (result.kind === "ok") {
      expect(result.retried).toBe(false);
      expect(result.response.status).toBe(200);
    }
    expect(call).toHaveBeenCalledTimes(1);
  });

  it("refreshes and retries when the first call returns 401", async () => {
    let accessToken = "old-access";
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: accessToken };
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });
    cookieSet.mockImplementation((name: string, value: string) => {
      if (name === "sh_access") accessToken = value;
    });

    const call = vi.fn(async (token: string) => {
      if (token === "old-access") return new Response(null, { status: 401 });
      return new Response("ok", { status: 200 });
    });
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              access_token: "new-access",
              refresh_token: "new-refresh",
              session: {
                id: "user-1",
                email: "person@example.com",
                full_name: "Person",
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

    const result = await withAuthRetry(call);

    expect(call).toHaveBeenCalledTimes(2);
    expect(call).toHaveBeenNthCalledWith(1, "old-access");
    expect(call).toHaveBeenNthCalledWith(2, "new-access");
    expect(result.kind).toBe("ok");
    if (result.kind === "ok") {
      expect(result.retried).toBe(true);
      expect(result.response.status).toBe(200);
    }
  });

  it("clears auth cookies and returns unauthorized when refresh is rejected", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: "old-access" };
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });
    const call = vi.fn(async () => new Response(null, { status: 401 }));
    vi.stubGlobal("fetch", vi.fn(async () => new Response(null, { status: 401 })));

    const result = await withAuthRetry(call);

    expect(result).toEqual({ kind: "unauthorized" });
    expect(cookieDelete).toHaveBeenCalledWith("sh_access");
  });

  it("returns the retried 401 response and clears cookies when the retry also fails", async () => {
    let accessToken = "old-access";
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_access") return { value: accessToken };
      if (name === "sh_refresh") return { value: "refresh-token" };
      return undefined;
    });
    cookieSet.mockImplementation((name: string, value: string) => {
      if (name === "sh_access") accessToken = value;
    });
    const call = vi.fn(async () => new Response(null, { status: 401 }));
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            data: {
              access_token: "new-access",
              refresh_token: "new-refresh",
              session: {
                id: "user-1",
                email: "person@example.com",
                full_name: "Person",
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

    const result = await withAuthRetry(call);

    expect(result.kind).toBe("ok");
    if (result.kind === "ok") {
      expect(result.retried).toBe(true);
      expect(result.response.status).toBe(401);
    }
    expect(cookieDelete).toHaveBeenCalledWith("sh_access");
  });
});

describe("refreshAuthSession", () => {
  beforeEach(() => {
    cookieSet.mockReset();
    cookieGet.mockReset();
    vi.restoreAllMocks();
  });

  it("writes the rotated access and refresh tokens returned by /auth/refresh", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_refresh") return { value: "old-refresh-token" };
      if (name === "sh_access") return { value: "old-access-token" };
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

    const result = await refreshAuthSession();

    expect(result).toEqual({
      kind: "success",
      session: {
        id: "user-1",
        email: "person@example.com",
        full_name: "Person Example",
        role: "admin",
        wizard_complete: true,
        scheme_memberships: [],
      },
    });
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_access",
      "new-access-token",
      expect.any(Object),
    );
    expect(cookieSet).toHaveBeenCalledWith(
      "sh_refresh",
      "new-refresh-token",
      expect.any(Object),
    );
  });

  it("returns invalid when the backend rejects the refresh token", async () => {
    cookieGet.mockImplementation((name: string) => {
      if (name === "sh_refresh") return { value: "bad-refresh-token" };
      return undefined;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(null, { status: 401 })),
    );

    await expect(refreshAuthSession()).resolves.toEqual({ kind: "invalid" });
    expect(cookieSet).not.toHaveBeenCalled();
  });

  it("returns unavailable when /auth/refresh aborts", async () => {
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

    await expect(refreshAuthSession()).resolves.toEqual({ kind: "unavailable" });
    expect(cookieSet).not.toHaveBeenCalled();
  });
});