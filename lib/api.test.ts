import { beforeEach, describe, expect, it, vi } from "vitest";

const refreshTokens = vi.fn();

vi.mock("./auth-actions", () => ({
  refreshTokens,
}));

describe("apiFetch", () => {
  beforeEach(() => {
    refreshTokens.mockReset();
    vi.restoreAllMocks();
    vi.stubGlobal("window", {
      location: {
        replace: vi.fn(),
      },
    } as unknown as Window & typeof globalThis);
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
    expect(refreshTokens).not.toHaveBeenCalled();
    expect(window.location.replace).not.toHaveBeenCalled();
  });
});
