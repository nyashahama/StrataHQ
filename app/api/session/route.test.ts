import { beforeEach, describe, expect, it, vi } from "vitest";

const cookieGet = vi.fn();

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => ({
    get: cookieGet,
  })),
}));

describe("GET /api/session", () => {
  beforeEach(() => {
    cookieGet.mockReset();
  });

  it("returns null for incomplete session payloads", async () => {
    cookieGet.mockReturnValue({
      value: encodeURIComponent(
        JSON.stringify({
          id: "user-1",
          email: "person@example.com",
        }),
      ),
    });

    const { GET } = await import("./route");
    const response = await GET();

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toBeNull();
  });
});
