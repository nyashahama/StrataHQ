import { describe, expect, it } from "vitest";

import {
  getApiErrorMessage,
  unwrapApiData,
} from "./api-contract";

describe("unwrapApiData", () => {
  it("unwraps data envelopes", () => {
    expect(unwrapApiData<{ id: string }>({ data: { id: "abc" } })).toEqual({
      id: "abc",
    });
  });

  it("returns raw payloads when no envelope exists", () => {
    expect(unwrapApiData<string[]>(["a", "b"])).toEqual(["a", "b"]);
  });
});

describe("getApiErrorMessage", () => {
  it("prefers the backend error message", () => {
    expect(
      getApiErrorMessage(
        { error: { code: "BAD_REQUEST", message: "invalid request body" } },
        "fallback",
      ),
    ).toBe("invalid request body");
  });

  it("falls back when the payload is not an api error", () => {
    expect(getApiErrorMessage({ nope: true }, "fallback")).toBe("fallback");
  });
});
