import { NextRequest, NextResponse } from "next/server";

const PUBLIC_PATHS = [
  "/auth/login",
  "/auth/register",
  "/auth/forgot-password",
  "/auth/reset-password",
  "/auth/invite",
  "/auth/pending",
  "/early-access",
  "/api/session",
  "/api/proxy",
];

function isPublicPath(pathname: string): boolean {
  if (pathname === "/" || pathname === "/auth" || pathname.startsWith("/api/")) {
    return true;
  }
  return PUBLIC_PATHS.some((p) => pathname.startsWith(p));
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // ─── CSRF Origin Validation (mutating requests) ─────────────────────────
  const method = request.method.toUpperCase();
  if (method !== "GET" && method !== "HEAD" && method !== "OPTIONS") {
    const origin = request.headers.get("origin");
    if (origin) {
      const allowedOrigins = [
        request.nextUrl.origin,
        ...(process.env.NEXT_PUBLIC_APP_URL
          ? [process.env.NEXT_PUBLIC_APP_URL]
          : []),
      ];
      if (!allowedOrigins.includes(origin)) {
        return new NextResponse("Forbidden", { status: 403 });
      }
    }
  }

  // ─── Auth Route Protection ──────────────────────────────────────────────
  if (!isPublicPath(pathname)) {
    const sessionCookie = request.cookies.get("sh_session");
    if (!sessionCookie) {
      const loginUrl = new URL("/auth/login", request.url);
      loginUrl.searchParams.set("redirect", pathname);
      return NextResponse.redirect(loginUrl);
    }
  }

  // ─── Security Headers ──────────────────────────────────────────────────
  const response = NextResponse.next();

  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  response.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(), geolocation=()",
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