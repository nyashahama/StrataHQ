import { beforeEach, describe, expect, it, vi } from "vitest";
import { refreshAuthSession } from "./server-auth";

const cookieSet = vi.fn();
const cookieGet = vi.fn();

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
    set: cookieSet,
  })),
}));

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
      id: "user-1",
      email: "person@example.com",
      full_name: "Person Example",
      role: "admin",
      wizard_complete: true,
      scheme_memberships: [],
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
});