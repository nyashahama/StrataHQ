import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import type { SessionUser } from "@/lib/session";

export async function GET() {
  const cookieStore = await cookies();
  const raw = cookieStore.get("sh_session")?.value;
  if (!raw) {
    return NextResponse.json(null);
  }
  try {
    const session = JSON.parse(decodeURIComponent(raw)) as SessionUser;
    return NextResponse.json(session);
  } catch {
    return NextResponse.json(null);
  }
}