# Frontend Auth Integration Design

**Date:** 2026-03-29
**Status:** Approved
**Scope:** Wire all frontend auth flows (login, register, onboarding, invite accept, password reset) to the completed Go backend. Replace `MockAuthProvider` with a real hybrid cookie-based auth system.

---

## Context

The backend auth system is complete (see `2026-03-29-full-auth-coverage-design.md`). The frontend currently uses `MockAuthProvider` (`lib/mock-auth.tsx`) with hardcoded data and localStorage. This spec covers replacing the mock layer with real API calls across all auth-related pages, the onboarding wizard, and the invitations management page.

---

## Architecture: Hybrid Cookie Auth

Three cookies set atomically by server actions on every auth event (login, register, invite accept, token refresh):

| Cookie | Readable by JS | Content | TTL |
|--------|---------------|---------|-----|
| `sh_access` | Yes | JWT access token | 15 min |
| `sh_refresh` | No (httpOnly) | Refresh token | 30 days |
| `sh_session` | Yes | JSON: `{id, email, full_name, role, wizard_complete, scheme_memberships}` | 30 days |

**Why this split:**
- `sh_refresh` is httpOnly — the long-lived token is protected from XSS.
- `sh_access` is readable — needed by `apiFetch()` to attach as `Authorization: Bearer`. Short TTL (15 min) limits XSS exposure.
- `sh_session` is readable — `AuthProvider` reads it synchronously on mount, no network call, no loading flash.

All cookies: `Secure`, `SameSite=Lax`, `Path=/`.

---

## New Files

### `lib/auth.tsx` — replaces `lib/mock-auth.tsx`

```ts
interface SessionUser {
  id: string
  email: string
  full_name: string
  role: 'admin' | 'trustee' | 'resident'
  wizard_complete: boolean
  scheme_memberships: {
    scheme_id: string
    scheme_name: string
    unit_id: string | null
    role: string
  }[]
}
```

`AuthProvider` reads `sh_session` from `document.cookie` synchronously in the initial state — no `useEffect`, no loading flash. Exposes `{ user: SessionUser | null, clearUser: () => void }`. `clearUser()` is called by the logout flow (calls `logoutAction()` then redirects to `/auth/login`). No `login()` method — auth mutations are server actions; callers do `router.refresh()` or redirect after they return.

### `lib/auth-actions.ts` — server actions (`'use server'`)

All actions call the Go backend at `process.env.BACKEND_URL` (server-side env, e.g. `http://localhost:8080`). On success they set cookies via `cookies()` from `next/headers`.

| Action | Backend call | Cookie effect |
|--------|-------------|---------------|
| `loginAction(email, password)` | `POST /auth/login` → `GET /auth/me` | Sets all 3 cookies |
| `registerAction(email, password, full_name)` | `POST /auth/register` → `GET /auth/me` | Sets all 3 cookies |
| `logoutAction()` | `POST /auth/logout` (sends `sh_refresh`) | Clears all 3 cookies |
| `refreshTokens()` | `POST /auth/refresh` (sends `sh_refresh`) | Updates `sh_access` |
| `clearAuth()` | — | Clears all 3 cookies |
| `setupAction(data)` | `POST /api/v1/onboarding/setup` | Updates `sh_session` (`wizard_complete: true`) |
| `forgotPasswordAction(email)` | `POST /auth/forgot-password` | — |
| `resetPasswordAction(token, password)` | `POST /auth/reset-password` | — |
| `acceptInviteAction(token, password)` | `POST /api/v1/invitations/{token}/accept` → `GET /auth/me` | Sets all 3 cookies |

`loginAction` and `registerAction` return `{ role, wizard_complete, scheme_memberships }` so the client can route without parsing the cookie.

`refreshTokens()` returns the new access token string so `apiFetch()` can retry the original request.

### `lib/api.ts` — client-side fetch utility

`apiFetch(path, options?)`:
1. Reads `sh_access` from `document.cookie`.
2. Calls `NEXT_PUBLIC_API_URL` (e.g. `http://localhost:8080`) with `Authorization: Bearer {token}`.
3. On 401: calls `refreshTokens()` server action to get a new access token, retries once.
4. On second 401: calls `clearAuth()` server action and redirects to `/auth/login`.

Used by client components that need to read/write protected resources (invitations list, etc).

---

## Environment Variables

| Variable | Where used | Example |
|----------|-----------|---------|
| `NEXT_PUBLIC_API_URL` | `lib/api.ts` (client) | `http://localhost:8080` |
| `BACKEND_URL` | `lib/auth-actions.ts` (server) | `http://localhost:8080` |

Both point to the same Go backend. `NEXT_PUBLIC_*` is exposed to the browser; `BACKEND_URL` stays server-only.

---

## Modified Pages

### `app/auth/login/page.tsx`

- Remove role selector.
- Remove `useMockAuth`.
- Form calls `loginAction(email, password)`.
- On success, route based on return value:
  - `role === 'admin' && !wizard_complete` → `/agent/setup`
  - `role === 'admin'` → `/agent`
  - otherwise → `/app/{scheme_memberships[0].scheme_id}`
- Show inline error on failure (invalid credentials = 401, server error = generic message).
- Guard: if user already authenticated, `useEffect` redirects to their dashboard.

### `app/auth/register/page.tsx`

- Remove role selector (agent-only).
- Remove `useMockAuth`.
- Form fields: full name, email, password.
- Calls `registerAction(email, password, full_name)`.
- On success: always routes to `/agent/setup`.
- On 409: show "An account with this email already exists."
- Guard: redirect if already authenticated.

### `app/auth/forgot-password/page.tsx`

- Wire `forgotPasswordAction(email)` — currently just sets `submitted = true` without an API call.
- No UI changes.

---

## New Pages

### `app/auth/reset-password/page.tsx`

- Reads `?token=` from search params.
- Fields: password + confirm password (client-side match validation).
- Calls `resetPasswordAction(token, password)`.
- On success: show "Password updated" state with link to `/auth/login`.
- On 401: show "This reset link is invalid or has expired."

### `app/auth/invite/[token]/page.tsx`

- On mount: calls `GET /api/v1/invitations/{token}` (public, no auth) via plain `fetch` to get `{email, full_name, role}`.
- Shows email + name read-only, password field.
- On submit: calls `acceptInviteAction(token, password)`.
- On success: routes to `/app/{scheme_memberships[0].scheme_id}`.
- On 401 (expired/revoked): "This invite link is invalid or has expired" + link to `/auth/login`.
- On 409: "An account with this email already exists — log in instead."

---

## Modified Components

### `components/wizard/SetupWizard.tsx`

- Remove `useMockAuth`.
- Steps 1 & 2 remain local state.
- On step 2 submit: call `setupAction({org_name, contact_email, scheme_name, scheme_address, unit_count})`.
- Show loading state while action runs. On error show inline message, stay on step 2.
- On success: advance to step 3. "Go to dashboard" button does `router.replace('/agent')`.

### `app/agent/invitations/page.tsx`

- Remove `mockInvitations`.
- On mount: `apiFetch('/api/v1/invitations')` to load pending invitations.
- Resend: `apiFetch('/api/v1/invitations/{id}/resend', { method: 'POST' })`.
- Revoke: `apiFetch('/api/v1/invitations/{id}', { method: 'DELETE' })`.
- Show loading skeleton while fetching. Show empty state if array is empty.

---

## Route Guards

All layouts swap `useMockAuth()` → `useAuth()`. Field mappings:

| MockUser | SessionUser |
|----------|-------------|
| `role === 'agent'` | `role === 'admin'` |
| `isWizardComplete` | `wizard_complete` |
| `schemeId` | `scheme_memberships[0].scheme_id` |
| `orgName` | removed from session (not needed for guards) |

**`app/agent/layout.tsx`:** checks `role === 'admin'`, else redirects non-admins to `/app/{scheme_memberships[0].scheme_id}`.

**`app/agent/setup/page.tsx`:** checks `wizard_complete`, redirects to `/agent` if true.

**`app/app/[schemeId]/layout.tsx`:**
- Redirects to `/auth/login` if no user.
- `user.schemeId !== schemeId` guard becomes: if `schemeId` is not in `user.scheme_memberships.map(m => m.scheme_id)`, redirect to `/app/{scheme_memberships[0].scheme_id}`. Admins are not members so they always pass this check.
- `sidebarRole`: `role === 'admin'` maps to `'agent-scheme'`.
- Header label: residents show `scheme_memberships[0].scheme_name` only — `unitIdentifier` is not stored in the session cookie (the session has `unit_id` UUID, not the display identifier like "4B"). This is a known simplification; unit display identifier can be added to the session in a follow-up once the backend `/auth/me` exposes it.

**`app/layout.tsx`:** swap `MockAuthProvider` → `AuthProvider`.

**`lib/mock-auth.tsx`:** deleted after all consumers migrated.

---

## `AgentPortfolioPage` — `orgName` removal

`app/agent/page.tsx` currently renders `user.orgName`. Since `orgName` is not in `sh_session`, replace with a static fallback `'My Organisation'` for now — the portfolio page still uses mock scheme data, so this is consistent.

---

## Error Handling

| Scenario | Status | Frontend behaviour |
|----------|--------|--------------------|
| Wrong credentials | 401 | Inline: "Invalid email or password" |
| Email already registered | 409 | Inline: "An account with this email already exists" |
| Invite expired/revoked | 401 | Error state page: "This invite link is invalid or has expired" |
| Reset token expired | 401 | Error state: "This reset link is invalid or has expired" |
| Onboarding setup fails | 500 | Inline on wizard step 2: "Setup failed — please try again" |
| Token refresh fails | 401 | Clear cookies, redirect to `/auth/login` |
| Network error | — | Generic: "Something went wrong — please try again" |

---

## File Map

| Action | File |
|--------|------|
| Create | `lib/auth.tsx` |
| Create | `lib/auth-actions.ts` |
| Create | `lib/api.ts` |
| Create | `.env.local` (NEXT_PUBLIC_API_URL, BACKEND_URL) |
| Modify | `app/layout.tsx` — swap MockAuthProvider → AuthProvider |
| Modify | `app/auth/login/page.tsx` |
| Modify | `app/auth/register/page.tsx` |
| Modify | `app/auth/forgot-password/page.tsx` |
| Create | `app/auth/reset-password/page.tsx` |
| Create | `app/auth/invite/[token]/page.tsx` |
| Modify | `components/wizard/SetupWizard.tsx` |
| Modify | `app/agent/layout.tsx` |
| Modify | `app/agent/setup/page.tsx` |
| Modify | `app/agent/page.tsx` — orgName fallback |
| Modify | `app/agent/invitations/page.tsx` |
| Modify | `app/app/[schemeId]/layout.tsx` |
| Delete | `lib/mock-auth.tsx` |
