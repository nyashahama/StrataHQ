import { NextResponse } from "next/server";

import { refreshAuthSession } from "@/lib/server-auth";

export async function POST() {
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
