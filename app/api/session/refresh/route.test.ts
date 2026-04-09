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
    cookieGet.mockReturnValue({ value: "access-token" });
    vi.restoreAllMocks();
  });

  it("unwraps the backend data envelope before shaping the session", async () => {
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
