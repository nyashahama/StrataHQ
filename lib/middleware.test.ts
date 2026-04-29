import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";

import { proxy } from "@/proxy";

describe("proxy", () => {
  it("allows protected pages when session cookie exists (API handles token refresh)", () => {
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        cookie: "sh_session=fake-session",
      },
    });

    const response = proxy(request);

    expect(response.status).toBe(200);
  });

  it("redirects protected pages when no session cookie", () => {
    const request = new NextRequest("http://localhost/agent");

    const response = proxy(request);

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe(
      "http://localhost/auth/login?redirect=%2Fagent",
    );
  });
});
