# Full Auth Coverage Design

**Date:** 2026-03-29
**Status:** Approved
**Scope:** Complete backend auth for all 3 user types (agent, trustee, resident) — onboarding wizard, invitation system, password reset, and enhanced /me routing signal.

---

## Context

The existing backend covers agent-only auth: register (user + org), login, refresh, logout, and /me. The frontend models three distinct user types with separate onboarding flows:

- **Agent** — self-registers, completes a setup wizard (org name + first scheme), manages schemes
- **Trustee** — invited by agent, sets password via invite link, accesses a specific scheme
- **Resident** — same as trustee, also linked to a specific unit

The gaps are: onboarding endpoint, invitation system, password reset, and an enhanced /me that tells the client where to route the user.

---

## User Flows

```
Agent self-registers:
  POST /auth/register {email, password, full_name}
  → JWT issued (role: "admin", wizard_complete: false)
  → frontend wizard → POST /api/v1/onboarding/setup {org_name, contact_email, scheme_name, scheme_address, unit_count}
  → returns {org, scheme} → client routes to /agent

Trustee / Resident (invited):
  Agent: POST /api/v1/invitations → email sent with token link
  Invitee sees /auth/pending (invitation outstanding)
  Invitee clicks link → GET /api/v1/invitations/{token} (verify, pre-fill name/email)
  Invitee sets password → POST /api/v1/invitations/{token}/accept {password}
  → user + memberships created, JWT issued → client routes to /app/{scheme_id}

All users after login:
  POST /auth/login → GET /auth/me
  "admin"   + wizard_complete: false → /agent/setup
  "admin"   + wizard_complete: true  → /agent
  "trustee" / "resident"             → /app/{scheme_memberships[0].scheme_id}
```

---

## DB Schema Changes

### New table: `invitations`

```sql
CREATE TABLE invitations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  scheme_id   UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
  unit_id     UUID REFERENCES units(id),
  email       TEXT NOT NULL,
  full_name   TEXT NOT NULL,
  role        TEXT NOT NULL CHECK (role IN ('trustee', 'resident')),
  token       TEXT NOT NULL UNIQUE,
  status      TEXT NOT NULL DEFAULT 'pending'
              CHECK (status IN ('pending', 'accepted', 'revoked')),
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX invitations_org_status_idx ON invitations(org_id, status);
```

### Extend `orgs`

```sql
ALTER TABLE orgs ADD COLUMN contact_email TEXT;
```

### No migration for password reset

Password reset tokens are stored in Redis (`pwreset:{token}` → `userID`, TTL 1h). No DB table needed.

### New sqlc queries

| File | Query |
|------|-------|
| `db/queries/auth.sql` | `UpdateOrg` — sets name + contact_email by org ID |
| `db/queries/auth.sql` | `ListSchemeMembershipsByUser` — joins scheme_memberships → schemes for /me |
| `db/queries/invitations.sql` | `CreateInvitation`, `GetInvitationByToken`, `ListInvitationsByOrg`, `UpdateInvitationStatus` |

---

## Auth Package Changes (`internal/auth`)

### `POST /auth/register` — simplified

- Request: `{email, password, full_name}` (org_name removed)
- Creates user + empty-name org + org_membership(role: "admin")
- Response: same `AuthResponse` shape as today

### `POST /api/v1/onboarding/setup` — new, protected (role: admin)

- Request: `{org_name, contact_email, scheme_name, scheme_address, unit_count}`
- Middleware enforces `role == "admin"` → `403` otherwise
- Transaction: `UpdateOrg(name, contact_email)` → `CreateScheme(org_id, name, address, unit_count)`
- Response:
  ```json
  {
    "org": { "id": "...", "name": "..." },
    "scheme": { "id": "...", "name": "..." }
  }
  ```

### `POST /auth/forgot-password` — new, public

- Request: `{email}`
- Looks up user; always returns `200` regardless of whether email exists (no enumeration)
- If user found: generate 32-byte random token → store in Redis `pwreset:{token}` = `userID`, TTL 1h → send password reset email
- Response: `200 {"message": "if that email is registered, a reset link has been sent"}`

### `POST /auth/reset-password` — new, public

- Request: `{token, password}`
- Looks up Redis key `pwreset:{token}` → `401` if missing or expired
- `bcrypt.GenerateFromPassword` on new password
- Transaction: `UpdateUserPassword` + `RevokeAllUserRefreshTokens` (forces re-login on all devices)
- Deletes Redis key
- Response: `204 No Content`

### `GET /auth/me` — enhanced

New response shape:
```json
{
  "id": "...",
  "email": "...",
  "full_name": "...",
  "org": { "id": "...", "name": "..." },
  "role": "admin",
  "wizard_complete": true,
  "scheme_memberships": [
    {
      "scheme_id": "...",
      "scheme_name": "...",
      "unit_id": null,
      "role": "admin"
    }
  ]
}
```

- `wizard_complete`: true if `ListSchemesByOrg` returns ≥ 1 scheme
- `scheme_memberships` (agents): populated from `ListSchemesByOrg` — agents own all schemes in their org, so no `scheme_memberships` rows are needed
- `scheme_memberships` (trustee/resident): from `ListSchemeMembershipsByUser` query
- Handler branches on `role == "admin"` to use the correct data path

---

## Invitation Package (`internal/invitation`)

### Agent-facing endpoints (protected, role: admin)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/invitations` | Create invitation + send email |
| `GET` | `/api/v1/invitations` | List pending invitations for org |
| `POST` | `/api/v1/invitations/{id}/resend` | Resend invitation email (new token) |
| `DELETE` | `/api/v1/invitations/{id}` | Revoke invitation |

### Public endpoints (no auth)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/invitations/{token}` | Verify token, return name/email/role for pre-fill |
| `POST` | `/api/v1/invitations/{token}/accept` | Accept invitation, create account, return JWT pair |

### `POST /invitations` request/response

```json
// request
{
  "email": "jane@gmail.com",
  "full_name": "Jane Smith",
  "role": "trustee",
  "scheme_id": "...",
  "unit_id": "..."
}
// 201 — invitation record
{
  "id": "...", "email": "...", "full_name": "...", "role": "trustee",
  "scheme_id": "...", "status": "pending", "expires_at": "..."
}
```

- Generates 32-byte random token, TTL 7 days
- Sends email with link: `{APP_BASE_URL}/auth/invite/{token}`
- `unit_id` required when role is `resident`

### `POST /invitations/{token}/accept` request/response

```json
// request
{ "password": "..." }

// 201 — same AuthResponse shape as register/login
{
  "access_token": "...", "refresh_token": "...", "expires_in": 900,
  "user": { "id": "...", "email": "...", "full_name": "..." }
}
```

Transaction:
1. Validate token (status=pending, not expired) → `401` if invalid
2. Check email not already registered → `409` if taken
3. `CreateUser` → `CreateOrgMembership(role)` → `UpsertSchemeMembership(role, unit_id)` → `UpdateInvitationStatus("accepted")`
4. Issue JWT pair using exported `auth.GenerateAccessToken` + `auth.GenerateRefreshToken` + `db.Q.CreateRefreshToken` directly (avoids coupling to auth.Service internals)
5. Client is logged in immediately

### Resend

Generates a new token, updates the record, sends fresh email. Old token invalidated.

### Revoke

Sets status to `revoked`. Returns `403` if invitation belongs to a different org (ownership check). `404` if not found.

### Ownership checks

All agent-facing endpoints verify `invitation.org_id == orgID from JWT claims`. Agents can only manage their own org's invitations.

---

## Notification Package (`internal/notification`)

### Interface

```go
type Sender interface {
    SendInvitation(ctx context.Context, to, name, inviteURL string) error
    SendPasswordReset(ctx context.Context, to, resetURL string) error
}
```

### Implementation — Resend API

`EmailClient` implements `Sender`. HTTP POST to `https://api.resend.com/emails`.

### Templates (`templates.go`)

```go
func InvitationEmail(name, inviteURL string) (subject, htmlBody string)
func PasswordResetEmail(resetURL string) (subject, htmlBody string)
```

Both return a plain subject line and a minimal HTML body with a single CTA button.

### New config fields

```
RESEND_API_KEY   string  (required)
APP_BASE_URL     string  (required, e.g. https://app.stratahq.co.za)
EMAIL_FROM       string  (default: noreply@stratahq.co.za)
```

### Testing

`Sender` is an interface. Tests inject a `noopSender` that captures calls without hitting the network.

---

## Error Handling

| Scenario | Status | Code |
|----------|--------|------|
| Email already registered (register) | 409 | `CONFLICT` |
| Email already registered (invite accept) | 409 | `CONFLICT` |
| Invalid/expired/revoked invite token | 401 | `UNAUTHORIZED` |
| Invalid/expired password reset token | 401 | `UNAUTHORIZED` |
| Forgot password (any outcome) | 200 | — |
| Onboarding called by non-admin | 403 | `FORBIDDEN` |
| Invitation not found or wrong org | 404 / 403 | `NOT_FOUND` / `FORBIDDEN` |
| Missing required fields | 400 | `BAD_REQUEST` |

---

## Files Changed

| File | Change |
|------|--------|
| `backend/db/migrations/XXX_invitations.sql` | New: invitations table + orgs.contact_email column |
| `backend/db/queries/auth.sql` | Add UpdateOrg, ListSchemeMembershipsByUser queries |
| `backend/db/queries/invitations.sql` | New: all invitation queries |
| `backend/db/gen/*` | Regenerated by sqlc |
| `backend/internal/auth/handler.go` | Simplify register; add onboarding, forgot-password, reset-password handlers |
| `backend/internal/auth/service.go` | Update Register; add Onboarding, ForgotPassword, ResetPassword; enhance Me |
| `backend/internal/auth/routes.go` | Add onboarding + password reset routes |
| `backend/internal/invitation/handler.go` | New |
| `backend/internal/invitation/service.go` | New |
| `backend/internal/invitation/routes.go` | New |
| `backend/internal/notification/email.go` | Implement Resend client |
| `backend/internal/notification/templates.go` | Add invitation + reset email templates |
| `backend/internal/config/config.go` | Add RESEND_API_KEY, APP_BASE_URL, EMAIL_FROM |
| `backend/cmd/server/main.go` | Wire invitation handler + notification client |
| `backend/internal/server/router.go` | Register invitation routes |

---

## Testing

| File | Coverage |
|------|---------|
| `internal/auth/handler_test.go` | Register without org_name; onboarding happy path + 403 for non-admin; forgot/reset password flows |
| `internal/invitation/handler_test.go` | Create + list + resend + revoke (ownership checks); accept happy path; expired/revoked token → 401; duplicate email → 409 |
| `internal/invitation/service_test.go` | Token generation uniqueness; invitation expiry logic |
| `tests/integration/auth_test.go` | Extend: register → wizard → login → /me wizard_complete=true |
| `tests/integration/invitation_test.go` | Full flow: agent invites → accept → login as trustee/resident → /me returns correct scheme_memberships |
