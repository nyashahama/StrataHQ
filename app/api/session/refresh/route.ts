import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST() {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  if (!accessToken) {
    return NextResponse.json(null, { status: 401 });
  }

  const res = await fetch(`${BACKEND()}/api/v1/auth/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!res.ok) {
    return NextResponse.json(null, { status: res.status });
  }

  const me = await res.json();
  const session = {
    id: me.id,
    email: me.email,
    full_name: me.full_name,
    phone: me.phone ?? null,
    role: me.role,
    wizard_complete: me.wizard_complete,
    scheme_memberships: me.scheme_memberships ?? [],
    org: me.org,
  };

  cookieStore.set(
    "sh_session",
    encodeURIComponent(JSON.stringify(session)),
    {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 30 * 24 * 60 * 60,
    },
  );

  return NextResponse.json(session);
}