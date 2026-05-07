import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";

import { proxy } from "@/proxy";

describe("proxy", () => {
  it("blocks mutating requests when both Origin and Referer are missing", () => {
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        cookie: "sh_session=fake-session",
      },
      method: "POST",
    });

    const response = proxy(request);

    expect(response.status).toBe(403);
  });

  it("allows mutating requests when Referer is valid and Origin is missing", () => {
    const request = new NextRequest("http://localhost/agent", {
      method: "POST",
      headers: {
        cookie: "sh_session=fake-session",
        referer: "http://localhost/account",
      },
    });

    const response = proxy(request);

    expect(response.status).toBe(200);
  });

  it("blocks mutating requests when Origin and Referer are untrusted", () => {
    const request = new NextRequest("http://localhost/agent", {
      method: "POST",
      headers: {
        cookie: "sh_session=fake-session",
        origin: "https://evil.com",
        referer: "https://evil.com/phish",
      },
    });

    const response = proxy(request);

    expect(response.status).toBe(403);
  });

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
