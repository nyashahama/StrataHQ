"use server";

import { cookies } from "next/headers";
import { readApiData, readApiError } from "./api-contract";
import { writeAuthCookies, clearAuthCookies } from "./server-auth";
import type { SessionUser } from "./session";
import { APP_ROLES, parseSessionCookie } from "./session";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const AUTH_FETCH_TIMEOUT_MS = 10_000;

async function fetchWithTimeout(
  url: string,
  options: RequestInit,
  timeoutMs = AUTH_FETCH_TIMEOUT_MS,
): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, { ...options, signal: controller.signal });
  } finally {
    clearTimeout(timeout);
  }
}

const TEMP_UNAVAILABLE = "Service temporarily unavailable — please try again";

const SESSION_OPTS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
  sameSite: "lax" as const,
  path: "/",
  maxAge: 30 * 24 * 60 * 60,
};

// ─── Login ────────────────────────────────────────────────────────────────────

export async function loginAction(
  email: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  let res: Response;
  try {
    res = await fetchWithTimeout(`${BACKEND()}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
  } catch {
    return { error: TEMP_UNAVAILABLE };
  }

  if (!res.ok) {
    return {
      error: await readApiError(
        res,
        res.status === 401
          ? "Invalid email or password"
          : "Login failed — please try again",
      ),
    };
  }

  const { access_token, refresh_token, session } = await readApiData<{
    access_token: string;
    refresh_token: string;
    session: SessionUser;
  }>(res);

  await writeAuthCookies({ access_token, refresh_token, session });
  return { user: session };
}

// ─── Register ─────────────────────────────────────────────────────────────────

export async function registerAction(
  email: string,
  password: string,
  full_name: string,
): Promise<{ user: SessionUser } | { error: string }> {
  let res: Response;
  try {
    res = await fetchWithTimeout(`${BACKEND()}/api/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, full_name }),
    });
  } catch {
    return { error: TEMP_UNAVAILABLE };
  }

  if (!res.ok) {
    if (res.status === 409)
      return {
        error: await readApiError(
          res,
          "An account with this email already exists",
        ),
      };
    return {
      error: await readApiError(res, "Registration failed — please try again"),
    };
  }

  const { access_token, refresh_token, session } = await readApiData<{
    access_token: string;
    refresh_token: string;
    session: SessionUser;
  }>(res);

  await writeAuthCookies({ access_token, refresh_token, session });
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

  await clearAuthCookies();
}

// ─── Clear auth ───────────────────────────────────────────────────────────────

export async function clearAuth(): Promise<void> {
  await clearAuthCookies();
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

  const raw = cookieStore.get("sh_session")?.value;
  const session = parseSessionCookie(raw);
  if (!session) {
    await clearAuthCookies();
    return { error: "Session expired — please log in again" };
  }

  let res: Response;
  try {
    res = await fetchWithTimeout(`${BACKEND()}/api/v1/onboarding/setup`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify(data),
    });
  } catch {
    return { error: TEMP_UNAVAILABLE };
  }

  if (!res.ok) {
    return {
      error: await readApiError(res, "Setup failed — please try again"),
    };
  }

  const result = await readApiData<{
    org: { id: string; name: string; contact_email?: string | null; contact_phone?: string | null };
    scheme: { id: string; name: string };
  }>(res);

  // Update session cookie: wizard_complete + first scheme membership
  session.wizard_complete = true;
  session.org = {
    id: result.org.id,
    name: result.org.name,
    contact_email: data.contact_email,
    contact_phone: result.org.contact_phone ?? session.org?.contact_phone ?? null,
  };
  session.scheme_memberships = [
    {
      scheme_id: result.scheme.id,
      scheme_name: result.scheme.name,
      unit_id: null,
      unit_identifier: null,
      role: APP_ROLES.admin,
    },
  ];
  cookieStore.set(
    "sh_session",
    encodeURIComponent(JSON.stringify(session)),
    SESSION_OPTS,
  );
  return { user: session };
}

// ─── Forgot password ──────────────────────────────────────────────────────────

export async function forgotPasswordAction(email: string): Promise<void> {
  fetchWithTimeout(`${BACKEND()}/api/v1/auth/forgot-password`, {
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
  let res: Response;
  try {
    res = await fetchWithTimeout(`${BACKEND()}/api/v1/auth/reset-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, password }),
    });
  } catch {
    return { error: TEMP_UNAVAILABLE };
  }

  if (!res.ok) {
    if (res.status === 401)
      return {
        error: await readApiError(
          res,
          "This reset link is invalid or has expired",
        ),
      };
    return { error: await readApiError(res, "Reset failed — please try again") };
  }

  return { ok: true };
}

// ─── Accept invite ────────────────────────────────────────────────────────────

export async function acceptInviteAction(
  token: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  let res: Response;
  try {
    res = await fetchWithTimeout(
      `${BACKEND()}/api/v1/invitations/verify/${token}/accept`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      },
    );
  } catch {
    return { error: TEMP_UNAVAILABLE };
  }

  if (!res.ok) {
    if (res.status === 401)
      return {
        error: await readApiError(
          res,
          "This invite link is invalid or has expired",
        ),
      };
    if (res.status === 409)
      return {
        error: await readApiError(
          res,
          "An account with this email already exists — log in instead",
        ),
      };
    return {
      error: await readApiError(res, "Something went wrong — please try again"),
    };
  }

  const { access_token, refresh_token, session } = await readApiData<{
    access_token: string;
    refresh_token: string;
    session: SessionUser;
  }>(res);

  await writeAuthCookies({ access_token, refresh_token, session });
  return { user: session };
}
