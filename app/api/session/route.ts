import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { parseSessionCookie } from "@/lib/session";

export async function GET() {
  const cookieStore = await cookies();
  return NextResponse.json(parseSessionCookie(cookieStore.get("sh_session")?.value));
}
