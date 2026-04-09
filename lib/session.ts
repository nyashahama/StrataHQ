export const APP_ROLES = {
  admin: "admin",
  trustee: "trustee",
  resident: "resident",
} as const;

export type AppRole = (typeof APP_ROLES)[keyof typeof APP_ROLES];

export interface SchemeMembership {
  scheme_id: string;
  scheme_name: string;
  unit_id: string | null;
  unit_identifier?: string | null;
  role: string;
}

export interface SessionOrg {
  id: string;
  name: string;
  contact_email?: string | null;
  contact_phone?: string | null;
}

export interface SessionUser {
  id: string;
  email: string;
  full_name: string;
  phone?: string | null;
  role: AppRole;
  wizard_complete: boolean;
  scheme_memberships: SchemeMembership[];
  org?: SessionOrg;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isNullableString(value: unknown): value is string | null | undefined {
  return typeof value === "string" || value === null || value === undefined;
}

function isSchemeMembership(value: unknown): value is SchemeMembership {
  return (
    isRecord(value) &&
    typeof value.scheme_id === "string" &&
    typeof value.scheme_name === "string" &&
    isNullableString(value.unit_id) &&
    isNullableString(value.unit_identifier) &&
    typeof value.role === "string"
  );
}

function isSessionOrg(value: unknown): value is SessionOrg {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.name === "string" &&
    isNullableString(value.contact_email) &&
    isNullableString(value.contact_phone)
  );
}

export function isSessionUser(value: unknown): value is SessionUser {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.email === "string" &&
    typeof value.full_name === "string" &&
    isNullableString(value.phone) &&
    (value.role === APP_ROLES.admin ||
      value.role === APP_ROLES.trustee ||
      value.role === APP_ROLES.resident) &&
    typeof value.wizard_complete === "boolean" &&
    Array.isArray(value.scheme_memberships) &&
    value.scheme_memberships.every(isSchemeMembership) &&
    (value.org === undefined || isSessionOrg(value.org))
  );
}

export function parseSessionCookie(raw?: string | null): SessionUser | null {
  if (!raw) return null;

  try {
    const parsed = JSON.parse(decodeURIComponent(raw));
    return isSessionUser(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

export function isAdminRole(role?: string | null): role is "admin" {
  return role === APP_ROLES.admin;
}

export function isResidentRole(role?: string | null): role is "resident" {
  return role === APP_ROLES.resident;
}

export function primarySchemeId(user: SessionUser | null): string | null {
  return user?.scheme_memberships[0]?.scheme_id ?? null;
}

export function hasSchemeMembership(
  user: SessionUser | null,
  schemeId: string,
): boolean {
  return (
    user?.scheme_memberships.some(
      (membership) => membership.scheme_id === schemeId,
    ) ?? false
  );
}

export function postLoginPath(user: SessionUser): string {
  if (isAdminRole(user.role)) {
    return user.wizard_complete ? "/agent" : "/agent/setup";
  }

  const schemeId = primarySchemeId(user);
  if (schemeId && /^[0-9a-f-]{36}$/i.test(schemeId)) {
    return `/app/${schemeId}`;
  }
  return "/auth/login";
}
