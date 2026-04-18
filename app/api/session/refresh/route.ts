import { NextResponse } from "next/server";

import { refreshAuthSession } from "@/lib/server-auth";

export async function POST() {
  const session = await refreshAuthSession();
  if (!session) {
    return NextResponse.json(null, { status: 401 });
  }

  return NextResponse.json(session);
}
