import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import {
  hasSchemeMembership,
  isAdminRole,
  parseSessionCookie,
  postLoginPath,
  primarySchemeId,
  type SessionUser,
} from "@/lib/session";

export async function getServerSession(): Promise<SessionUser | null> {
  const cookieStore = await cookies();
  return parseSessionCookie(cookieStore.get("sh_session")?.value);
}

export async function requireServerSession(): Promise<SessionUser> {
  const session = await getServerSession();
  if (!session) {
    redirect("/auth/login");
  }
  return session;
}

export async function requireAdminSession(): Promise<SessionUser> {
  const session = await requireServerSession();
  if (!isAdminRole(session.role)) {
    redirect(postLoginPath(session));
  }
  return session;
}

export async function requireSchemeSession(
  schemeId: string,
): Promise<SessionUser> {
  const session = await requireServerSession();
  if (isAdminRole(session.role)) {
    return session;
  }

  if (!hasSchemeMembership(session, schemeId)) {
    redirect(`/app/${primarySchemeId(session) ?? ""}`);
  }

  return session;
}
