"use server";

import { cookies } from "next/headers";
import type { SessionUser } from "./auth";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const ACCESS_OPTS = {
  httpOnly: false,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 15 * 60,
};

const REFRESH_OPTS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 30 * 24 * 60 * 60,
};

const SESSION_OPTS = {
  httpOnly: false,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 30 * 24 * 60 * 60,
};

async function setAuthCookies(
  cookieStore: Awaited<ReturnType<typeof cookies>>,
  accessToken: string,
  refreshToken: string,
  me: SessionUser,
) {
  const session: SessionUser = {
    id: me.id,
    email: me.email,
    full_name: me.full_name,
    role: me.role,
    wizard_complete: me.wizard_complete,
    scheme_memberships: me.scheme_memberships ?? [],
  };
  cookieStore.set("sh_access", accessToken, ACCESS_OPTS);
  cookieStore.set("sh_refresh", refreshToken, REFRESH_OPTS);
  cookieStore.set(
    "sh_session",
    encodeURIComponent(JSON.stringify(session)),
    SESSION_OPTS,
  );
  return session;
}

async function fetchMe(accessToken: string): Promise<SessionUser | null> {
  const res = await fetch(`${BACKEND()}/api/v1/auth/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!res.ok) return null;
  const body = await res.json();
  // Unwrap { data: ... } envelope if present
  return body.data ?? body;
}

// ─── Login ────────────────────────────────────────────────────────────────────

export async function loginAction(
  email: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });

  if (!res.ok) {
    return {
      error:
        res.status === 401
          ? "Invalid email or password"
          : "Login failed — please try again",
    };
  }

  const { data } = await res.json();
  const { access_token, refresh_token, user: rawUser } = data;

  // Backend returns user inline; try /me for full shape, fall back to inline user
  const me = (await fetchMe(access_token)) ?? {
    id: rawUser.id,
    email: rawUser.email,
    full_name: rawUser.full_name,
    role: "admin" as const,
    wizard_complete: false,
    scheme_memberships: [],
  };

  const cookieStore = await cookies();
  const session = await setAuthCookies(
    cookieStore,
    access_token,
    refresh_token,
    me,
  );
  return { user: session };
}

// ─── Register ─────────────────────────────────────────────────────────────────

export async function registerAction(
  email: string,
  password: string,
  full_name: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, full_name }),
  });

  if (!res.ok) {
    if (res.status === 409)
      return { error: "An account with this email already exists" };
    return { error: "Registration failed — please try again" };
  }

  const { data } = await res.json();
  const { access_token, refresh_token, user: rawUser } = data;

  // Backend returns user inline; try /me for full shape, fall back to inline user
  const me = (await fetchMe(access_token)) ?? {
    id: rawUser.id,
    email: rawUser.email,
    full_name: rawUser.full_name,
    role: "admin" as const,
    wizard_complete: false,
    scheme_memberships: [],
  };

  const cookieStore = await cookies();
  const session = await setAuthCookies(
    cookieStore,
    access_token,
    refresh_token,
    me,
  );
  return { user: session };
}

// ─── Logout ───────────────────────────────────────────────────────────────────

export async function logoutAction(): Promise<void> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get("sh_refresh")?.value;

  if (refreshToken) {
    await fetch(`${BACKEND()}/api/v1/auth/logout`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    }).catch(() => {});
  }

  cookieStore.delete("sh_access");
  cookieStore.delete("sh_refresh");
  cookieStore.delete("sh_session");
}

// ─── Token refresh ────────────────────────────────────────────────────────────

export async function refreshTokens(): Promise<string | null> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get("sh_refresh")?.value;
  if (!refreshToken) return null;

  const res = await fetch(`${BACKEND()}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!res.ok) return null;

  const body = await res.json();
  const access_token = body.data?.access_token ?? body.access_token;
  if (!access_token) return null;
  cookieStore.set("sh_access", access_token, ACCESS_OPTS);
  return access_token;
}

// ─── Clear auth ───────────────────────────────────────────────────────────────

export async function clearAuth(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete("sh_access");
  cookieStore.delete("sh_refresh");
  cookieStore.delete("sh_session");
}

// ─── Onboarding setup ─────────────────────────────────────────────────────────

export async function setupAction(data: {
  org_name: string;
  contact_email: string;
  scheme_name: string;
  scheme_address: string;
  unit_count: number;
}): Promise<{ user: SessionUser } | { error: string }> {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  if (!accessToken) return { error: "Not authenticated" };

  const res = await fetch(`${BACKEND()}/api/v1/onboarding/setup`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(data),
  });

  if (!res.ok) return { error: "Setup failed — please try again" };

  const body = await res.json();
  const result = body.data ?? body;

  // Update session cookie: wizard_complete + first scheme membership
  const raw = cookieStore.get("sh_session")?.value;
  if (raw) {
    const session = JSON.parse(decodeURIComponent(raw)) as SessionUser;
    session.wizard_complete = true;
    session.scheme_memberships = [
      {
        scheme_id: result.scheme.id,
        scheme_name: result.scheme.name,
        unit_id: null,
        role: "admin",
      },
    ];
    cookieStore.set(
      "sh_session",
      encodeURIComponent(JSON.stringify(session)),
      SESSION_OPTS,
    );
    return { user: session };
  }

  return { error: "Session not found — please log in again" };
}

// ─── Forgot password ──────────────────────────────────────────────────────────

export async function forgotPasswordAction(email: string): Promise<void> {
  await fetch(`${BACKEND()}/api/v1/auth/forgot-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  }).catch(() => {});
  // Always succeeds from the client's perspective (no email enumeration)
}

// ─── Reset password ───────────────────────────────────────────────────────────

export async function resetPasswordAction(
  token: string,
  password: string,
): Promise<{ ok: true } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/auth/reset-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token, password }),
  });

  if (!res.ok) {
    if (res.status === 401)
      return { error: "This reset link is invalid or has expired" };
    return { error: "Reset failed — please try again" };
  }

  return { ok: true };
}

// ─── Accept invite ────────────────────────────────────────────────────────────

export async function acceptInviteAction(
  token: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/invitations/${token}/accept`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });

  if (!res.ok) {
    if (res.status === 401)
      return { error: "This invite link is invalid or has expired" };
    if (res.status === 409)
      return {
        error: "An account with this email already exists — log in instead",
      };
    return { error: "Something went wrong — please try again" };
  }

  const inviteBody = await res.json();
  const { access_token, refresh_token } = inviteBody.data ?? inviteBody;
  const me = await fetchMe(access_token);
  if (!me) return { error: "Something went wrong — please try again" };

  const cookieStore = await cookies();
  const session = await setAuthCookies(
    cookieStore,
    access_token,
    refresh_token,
    me,
  );
  return { user: session };
}
