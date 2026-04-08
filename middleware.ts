import { NextRequest, NextResponse } from "next/server";

export function middleware(request: NextRequest) {
  const method = request.method.toUpperCase();
  if (method === "GET" || method === "HEAD" || method === "OPTIONS") {
    return NextResponse.next();
  }

  const origin = request.headers.get("origin");
  if (!origin) {
    return NextResponse.next();
  }

  const allowedOrigins = [
    request.nextUrl.origin,
    ...(process.env.NEXT_PUBLIC_APP_URL
      ? [process.env.NEXT_PUBLIC_APP_URL]
      : []),
  ];

  if (!allowedOrigins.includes(origin)) {
    return new NextResponse("Forbidden", { status: 403 });
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|stratahq_logo.svg).*)",
  ],
};