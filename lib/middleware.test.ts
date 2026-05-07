import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";

import { proxy } from "@/proxy";

function base64Url(value: string): string {
  return Buffer.from(value)
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

function mintJwt(subject: string, expSeconds: number): string {
  const header = base64Url(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = base64Url(JSON.stringify({ sub: subject, exp: expSeconds }));
  return `${header}.${payload}.sig`;
}

function encodedSessionCookie(session: Record<string, unknown>): string {
  return `sh_session=${encodeURIComponent(JSON.stringify(session))}`;
}

function getSetCookieValue(headers: Headers): string {
  return headers.get("set-cookie") ?? "";
}

describe("proxy", () => {
  const validSession = {
    id: "user-1",
    email: "user@example.com",
    full_name: "User One",
    role: "admin",
    wizard_complete: false,
    scheme_memberships: [],
  };
  const validSessionCookie = encodedSessionCookie(validSession);

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
        cookie: `${validSessionCookie}; sh_access=${mintJwt(
          "user-1",
          Math.floor(Date.now() / 1000) + 120,
        )}; sh_csrf=token`,
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

  it("clears auth cookies when session cookie is missing", () => {
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        referer: "http://localhost/account",
      },
      method: "POST",
    });

    const response = proxy(request);

    expect(response.status).toBe(307);
    const setCookie = getSetCookieValue(response.headers);
    expect(setCookie).toContain("sh_session=;");
    expect(setCookie).toContain("sh_access=;");
    expect(setCookie).toContain("sh_refresh=;");
    expect(setCookie).toContain("sh_csrf=;");
  });

  it("clears auth cookies when access token is missing", () => {
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        referer: "http://localhost/account",
        cookie:
          `${validSessionCookie}; sh_csrf=token`,
      },
      method: "POST",
    });

    const response = proxy(request);

    expect(response.status).toBe(307);
    const setCookie = getSetCookieValue(response.headers);
    expect(setCookie).toContain("sh_session=;");
  });

  it("clears auth cookies when access token is expired", () => {
    const expiredAccess = mintJwt("user-1", Math.floor(Date.now() / 1000) - 60);
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        origin: "http://localhost",
        cookie:
          `${validSessionCookie}; sh_access=${expiredAccess}; sh_csrf=token`,
      },
      method: "POST",
    });

    const response = proxy(request);

    expect(response.status).toBe(307);
    const setCookie = getSetCookieValue(response.headers);
    expect(setCookie).toContain("sh_session=;");
    expect(setCookie).toContain("sh_access=");
  });

  it("allows protected pages when session cookie exists (API handles token refresh)", () => {
    const accessToken = mintJwt("user-1", Math.floor(Date.now() / 1000) + 60);
    const request = new NextRequest("http://localhost/agent", {
      headers: {
        cookie: `${validSessionCookie}; sh_access=${accessToken}; sh_csrf=token`,
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
