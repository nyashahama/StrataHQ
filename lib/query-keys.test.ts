import { describe, expect, it } from "vitest"
import { schemeKeys } from "./query-keys"

describe("schemeKeys", () => {
  it("keeps filtered communication keys under one invalidation family", () => {
    expect(schemeKeys.communicationsBase("scheme-1")).toEqual([
      "scheme",
      "scheme-1",
      "communications",
    ])
    expect(schemeKeys.communications("scheme-1", "agm")).toEqual([
      "scheme",
      "scheme-1",
      "communications",
      "agm",
    ])
  })

  it("includes audit key", () => {
    expect(schemeKeys.audit("scheme-1")).toEqual([
      "scheme",
      "scheme-1",
      "audit",
    ])
  })
})