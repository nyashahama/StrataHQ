import { createHmac } from "crypto";
import { NextRequest, NextResponse } from "next/server";

import { parseSessionCookie } from "@/lib/session";

const PUBLIC_PATHS = [
  "/auth/login",
  "/auth/register",
  "/auth/forgot-password",
  "/auth/reset-password",
  "/auth/invite",
  "/auth/pending",
  "/early-access",
  "/api/health",
  "/api/session",
  "/api/proxy",
];

function isPublicPath(pathname: string): boolean {
  if (pathname === "/" || pathname === "/auth") {
    return true;
  }
  return PUBLIC_PATHS.some((p) => pathname.startsWith(p));
}

function isAllowedOrigin(rawOrigin: string | null, request: NextRequest): boolean {
  if (!rawOrigin) {
    return false;
  }

  const allowedOrigins = [
    request.nextUrl.origin,
    ...(process.env.NEXT_PUBLIC_APP_URL ? [process.env.NEXT_PUBLIC_APP_URL] : []),
  ];

  try {
    const parsed = new URL(rawOrigin);
    return allowedOrigins.includes(parsed.origin);
  } catch {
    return false;
  }
}

function decodeBase64URL(input: string): string {
  const normalized = input
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(input.length + ((4 - (input.length % 4)) % 4), "=");
  return Buffer.from(normalized, "base64").toString("utf8");
}

function computeJwtSignature(headerPayload: string, secret: string): string {
  return createHmac("sha256", secret)
    .update(headerPayload)
    .digest("base64url");
}

function isValidAccessToken(token: string, sessionId: string): boolean {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return false;
  }

  const secret = process.env.JWT_SECRET;
  if (secret) {
    const headerPayload = parts[0] + "." + parts[1];
    const expectedSig = computeJwtSignature(headerPayload, secret);
    if (parts[2] !== expectedSig) return false;
  }

  const payload = parts[1];
  if (payload === undefined) {
    return false;
  }

  let claims: Record<string, unknown>;
  try {
    claims = JSON.parse(decodeBase64URL(payload)) as Record<string, unknown>;
  } catch {
    return false;
  }

  const tokenSub = claims.sub;
  if (typeof tokenSub === "string" && tokenSub !== sessionId) {
    return false;
  }

  const rawExp = claims.exp;
  if (typeof rawExp !== "number" || !Number.isFinite(rawExp)) {
    return false;
  }

  const nowEpoch = Math.floor(Date.now() / 1000);
  if (rawExp <= nowEpoch) {
    return false;
  }

  return true;
}

function clearStaleSessionCookies(response: NextResponse): NextResponse {
  response.cookies.delete("sh_session");
  response.cookies.delete("sh_access");
  response.cookies.delete("sh_refresh");
  response.cookies.delete("sh_csrf");
  return response;
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const method = request.method.toUpperCase();
  if (method !== "GET" && method !== "HEAD" && method !== "OPTIONS") {
    const origin = request.headers.get("origin");
    const referer = request.headers.get("referer");

    if (!isAllowedOrigin(origin, request) && !isAllowedOrigin(referer, request)) {
      return new NextResponse("Forbidden", { status: 403 });
    }
  }

  if (!isPublicPath(pathname)) {
    const sessionCookie = request.cookies.get("sh_session");
    const accessToken = request.cookies.get("sh_access")?.value;
    const refreshToken = request.cookies.get("sh_refresh")?.value;
    const session = parseSessionCookie(sessionCookie?.value);

    if (!sessionCookie || !session) {
      const loginUrl = new URL("/auth/login", request.url);
      loginUrl.searchParams.set("redirect", pathname);
      return clearStaleSessionCookies(NextResponse.redirect(loginUrl));
    }

    if (!accessToken || !isValidAccessToken(accessToken, session.id)) {
      if (!refreshToken) {
        const loginUrl = new URL("/auth/login", request.url);
        loginUrl.searchParams.set("redirect", pathname);
        return clearStaleSessionCookies(NextResponse.redirect(loginUrl));
      }
      // Access token expired but refresh token exists. Allow the request to
      // proceed; the API client will refresh the session on the first 401.
    }
  }

  const response = NextResponse.next();

  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  response.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(), geolocation=()",
  );
  response.headers.set(
    "Content-Security-Policy",
    "default-src 'self'; script-src 'self' 'unsafe-inline'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'",
  );

  if (process.env.NODE_ENV === "production") {
    response.headers.set(
      "Strict-Transport-Security",
      "max-age=63072000; includeSubDomains; preload",
    );
  }

  return response;
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|stratahq_logo.svg).*)",
  ],
};
