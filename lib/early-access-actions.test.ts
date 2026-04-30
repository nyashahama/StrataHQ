import { beforeEach, describe, expect, it, vi } from "vitest";

describe("submitEarlyAccessRequest", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("applies a timeout signal to the early-access request", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    const { submitEarlyAccessRequest } = await import("./early-access-actions");
    const result = await submitEarlyAccessRequest({
      full_name: "Person Example",
      email: "person@example.com",
      scheme_name: "Scheme",
      unit_count: 42,
    });

    expect(result).toEqual({ ok: true });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/early-access",
      expect.objectContaining({
        method: "POST",
        signal: expect.any(AbortSignal),
      }),
    );
  });

  it("returns a stable unavailable error when the request times out", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new DOMException("The operation was aborted", "AbortError");
      }),
    );

    const { submitEarlyAccessRequest } = await import("./early-access-actions");
    await expect(
      submitEarlyAccessRequest({
        full_name: "Person Example",
        email: "person@example.com",
        scheme_name: "Scheme",
        unit_count: 42,
      }),
    ).resolves.toEqual({
      error: "Service temporarily unavailable — please try again",
    });
  });
});
