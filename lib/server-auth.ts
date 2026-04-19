"use server";

import { cookies } from "next/headers";
import { readApiData } from "./api-contract";
import type { SessionUser } from "./session";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

export interface AuthSessionPayload {
  access_token: string;
  refresh_token: string;
  session: SessionUser;
}

const SESSION_OPTS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 30 * 24 * 60 * 60,
};

const ACCESS_OPTS = {
  ...SESSION_OPTS,
  maxAge: 15 * 60,
};

export async function writeAuthCookies(
  payload: AuthSessionPayload,
): Promise<SessionUser> {
  const { access_token, refresh_token, session } = payload;
  const cookieStore = await cookies();

  cookieStore.set("sh_access", access_token, ACCESS_OPTS);
  cookieStore.set("sh_refresh", refresh_token, SESSION_OPTS);
  cookieStore.set(
    "sh_session",
    encodeURIComponent(JSON.stringify(session)),
    SESSION_OPTS,
  );

  return session;
}

export async function refreshAuthSession(): Promise<SessionUser | null> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get("sh_refresh")?.value;
  if (!refreshToken) return null;

  const res = await fetch(`${BACKEND()}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!res.ok) return null;

  const payload = await readApiData<AuthSessionPayload>(res);
  if (!payload.access_token || !payload.refresh_token || !payload.session) {
    return null;
  }

  return writeAuthCookies(payload);
}

export async function clearAuthCookies(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete("sh_access");
  cookieStore.delete("sh_refresh");
  cookieStore.delete("sh_session");
}