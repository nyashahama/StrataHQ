import { beforeEach, describe, expect, it, vi } from "vitest";

describe("apiFetch", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("does not refresh tokens for public requests that return 401", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 401 }));
    vi.stubGlobal("fetch", fetchMock);

    const { apiFetch } = await import("./api");
    const response = await apiFetch(
      "/api/v1/invitations/verify/token-123",
      { auth: false } as RequestInit & { auth?: boolean },
    );

    expect(response.status).toBe(401);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/proxy/api/v1/invitations/verify/token-123",
      expect.objectContaining({
        headers: expect.objectContaining({
          "x-skip-auth": "true",
        }),
      }),
    );
  });
});