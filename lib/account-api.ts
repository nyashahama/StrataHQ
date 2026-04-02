"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { SessionOrg, SessionUser } from "@/lib/session";

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
    throw new Error(await readApiError(res, "Failed to update profile"));
  }

  return readApiData<SessionUser>(res);
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
    throw new Error(
      await readApiError(res, "Failed to update organisation settings"),
    );
  }

  return readApiData<SessionOrg>(res);
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
    throw new Error(await readApiError(res, "Failed to update password"));
  }
}
