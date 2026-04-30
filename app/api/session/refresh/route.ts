import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { refreshAuthSession } from "@/lib/server-auth";

export async function POST(request: Request) {
  const cookieStore = await cookies();
  const csrfCookie = cookieStore.get("sh_csrf")?.value;
  const csrfHeader = request.headers.get("x-csrf-token");

  if (!csrfCookie || !csrfHeader || csrfCookie !== csrfHeader) {
    return NextResponse.json(
      {
        error: {
          code: "FORBIDDEN",
          message: "Invalid CSRF token",
        },
      },
      { status: 403 },
    );
  }

  const result = await refreshAuthSession();

  if (result.kind === "invalid") {
    return NextResponse.json(null, { status: 401 });
  }

  if (result.kind === "unavailable") {
    return NextResponse.json(
      {
        error: {
          code: "UPSTREAM_UNAVAILABLE",
          message: "Temporary service issue. Please retry.",
        },
      },
      { status: 503 },
    );
  }

  return NextResponse.json(result.session);
}
