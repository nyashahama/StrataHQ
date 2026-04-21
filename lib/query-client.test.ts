import { describe, expect, it } from "vitest";

import { ApiHttpError } from "./api-contract";
import { defaultRetryDecider } from "./query-client";

describe("defaultRetryDecider", () => {
  it("does not retry authorization and not-found errors", () => {
    const error403 = new ApiHttpError("Forbidden", 403, "FORBIDDEN");
    const error404 = new ApiHttpError("Missing", 404, "NOT_FOUND");

    expect(defaultRetryDecider(0, error403)).toBe(false);
    expect(defaultRetryDecider(0, error404)).toBe(false);
  });

  it("retries non-terminal errors up to two attempts", () => {
    const error500 = new ApiHttpError("Temporary", 500, "UPSTREAM");

    expect(defaultRetryDecider(0, error500)).toBe(true);
    expect(defaultRetryDecider(1, error500)).toBe(true);
    expect(defaultRetryDecider(2, error500)).toBe(false);
  });
});
