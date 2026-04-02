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
  role: string;
}

export interface SessionUser {
  id: string;
  email: string;
  full_name: string;
  role: AppRole;
  wizard_complete: boolean;
  scheme_memberships: SchemeMembership[];
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
  return schemeId ? `/app/${schemeId}` : "/auth/login";
}
