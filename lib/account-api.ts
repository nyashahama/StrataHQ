"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import { readBrowserCSRFToken } from "@/lib/csrf";
import type { SessionOrg, SessionUser } from "@/lib/session";

async function refreshSession(): Promise<SessionUser> {
  const csrfToken = readBrowserCSRFToken();
  const res = await fetch("/api/session/refresh", {
    method: "POST",
    headers: csrfToken ? { "x-csrf-token": csrfToken } : undefined,
  });

  if (!res.ok) {
    throw await buildApiHttpError(res, "Failed to refresh session");
  }

  return readApiData<SessionUser>(res);
}

export async function updateProfile(input: {
  email: string;
  full_name: string;
  phone: string;
}): Promise<SessionUser> {
  const res = await apiFetch("/api/v1/auth/profile", {
    method: "PATCH",
    body: JSON.stringify({
      email: input.email,
      full_name: input.full_name,
      phone: input.phone || null,
    }),
  });

  if (!res.ok) {
    throw await buildApiHttpError(res, "Failed to update profile");
  }

  const updated = await readApiData<SessionUser>(res);
  try {
    return await refreshSession();
  } catch {
    return updated;
  }
}

export async function updateOrgSettings(input: {
  name: string;
  contact_email: string;
  contact_phone: string;
}): Promise<SessionOrg> {
  const res = await apiFetch("/api/v1/auth/org", {
    method: "PATCH",
    body: JSON.stringify({
      name: input.name,
      contact_email: input.contact_email || null,
      contact_phone: input.contact_phone || null,
    }),
  });

  if (!res.ok) {
    throw await buildApiHttpError(
      res,
      "Failed to update organisation settings",
    );
  }

  const updated = await readApiData<SessionOrg>(res);
  try {
    return (await refreshSession()).org ?? updated;
  } catch {
    return updated;
  }
}

export async function changePassword(input: {
  current_password: string;
  new_password: string;
}): Promise<void> {
  const res = await apiFetch("/api/v1/auth/change-password", {
    method: "POST",
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    throw await buildApiHttpError(res, "Failed to update password");
  }
}
