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

export type RefreshAuthSessionResult =
  | { kind: "success"; session: SessionUser }
  | { kind: "invalid" }
  | { kind: "unavailable" }

export type WithAuthRetryResult =
  | { kind: "ok"; response: Response; retried: boolean }
  | { kind: "unauthorized" }
  | { kind: "unavailable" }

const AUTH_REFRESH_TIMEOUT_MS = 10_000;

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

const CSRF_OPTS = {
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 30 * 24 * 60 * 60,
};

function issueCSRFToken(): string {
  return crypto.randomUUID();
}

export async function writeAuthCookies(
  payload: AuthSessionPayload,
): Promise<SessionUser> {
  const { access_token, refresh_token, session } = payload;
  const cookieStore = await cookies();

  cookieStore.set("sh_access", access_token, ACCESS_OPTS);
  cookieStore.set("sh_refresh", refresh_token, SESSION_OPTS);
  cookieStore.set("sh_csrf", issueCSRFToken(), CSRF_OPTS);
  cookieStore.set(
    "sh_session",
    encodeURIComponent(JSON.stringify(session)),
    SESSION_OPTS,
  );

  return session;
}

export async function refreshAuthSession(): Promise<RefreshAuthSessionResult> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get("sh_refresh")?.value;
  if (!refreshToken) return { kind: "invalid" };

  const controller = new AbortController();
  const timeout = setTimeout(
    () => controller.abort(),
    AUTH_REFRESH_TIMEOUT_MS,
  );

  let res: Response;
  try {
    res = await fetch(`${BACKEND()}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
      signal: controller.signal,
    });
  } catch (error) {
    clearTimeout(timeout);
    if (error instanceof DOMException && error.name === "AbortError") {
      return { kind: "unavailable" };
    }
    return { kind: "unavailable" };
  }
  clearTimeout(timeout);

  if (res.status === 401 || res.status === 403) return { kind: "invalid" };
  if (!res.ok) return { kind: "unavailable" };

  const payload = await readApiData<AuthSessionPayload>(res);
  if (!payload.access_token || !payload.refresh_token || !payload.session) {
    return { kind: "unavailable" };
  }

  return { kind: "success", session: await writeAuthCookies(payload) };
}

// withAuthRetry centralises the "call the backend, on 401 refresh the
// session, retry once" pattern that the proxy, copilot, setup wizard, and
// change-password action all need. Callers supply the request factory and
// decide how to translate the three result kinds into HTTP responses.
export async function withAuthRetry(
  call: (accessToken: string) => Promise<Response>,
): Promise<WithAuthRetryResult> {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  if (!accessToken) {
    return { kind: "unauthorized" };
  }

  const initial = await call(accessToken);
  if (initial.status !== 401) {
    return { kind: "ok", response: initial, retried: false };
  }

  const refreshed = await refreshAuthSession();
  if (refreshed.kind === "invalid") {
    await clearAuthCookies();
    return { kind: "unauthorized" };
  }
  if (refreshed.kind === "unavailable") {
    return { kind: "unavailable" };
  }

  const newAccessToken = cookieStore.get("sh_access")?.value;
  if (!newAccessToken) {
    await clearAuthCookies();
    return { kind: "unauthorized" };
  }

  const retried = await call(newAccessToken);
  if (retried.status === 401) {
    await clearAuthCookies();
  }
  return { kind: "ok", response: retried, retried: true };
}

export async function clearAuthCookies(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete("sh_access");
  cookieStore.delete("sh_refresh");
  cookieStore.delete("sh_csrf");
  cookieStore.delete("sh_session");
}
