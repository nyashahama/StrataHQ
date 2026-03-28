# Backend Auth Design

**Date:** 2026-03-28
**Status:** Approved
**Scope:** Custom JWT authentication for the Go backend — register, login, refresh, logout, and /me

---

## Context

The backend has a fully scaffolded auth domain (`internal/auth/`) with stub handlers returning placeholder responses, an empty `tokens.go`, and a middleware that validates header format only. The database schema is complete: `users`, `orgs`, `org_memberships`, and `refresh_tokens` tables with all required sqlc queries already generated. This spec covers implementing the full auth flow.

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | none | Create user + org, return token pair |
| POST | `/api/v1/auth/login` | none | Verify credentials, return token pair |
| POST | `/api/v1/auth/refresh` | none | Rotate refresh token, return new pair |
| POST | `/api/v1/auth/logout` | none | Revoke refresh token |
| GET | `/api/v1/auth/me` | Bearer JWT | Return user profile + org membership |

### Request / Response Shapes

**POST /register**
```json
// request
{ "email": "jane@acme.co.za", "password": "...", "full_name": "Jane Smith", "org_name": "Acme Property Management" }

// 201
{ "access_token": "...", "refresh_token": "...", "expires_in": 900,
  "user": { "id": "...", "email": "...", "full_name": "..." } }
```

**POST /login**
```json
// request
{ "email": "jane@acme.co.za", "password": "..." }

// 200 — same shape as register response
```

**POST /refresh**
```json
// request
{ "refresh_token": "..." }

// 200
{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }
```

**POST /logout**
```json
// request
{ "refresh_token": "..." }
// 204 no content
```

**GET /me**
```json
// 200
{ "id": "...", "email": "...", "full_name": "...",
  "org": { "id": "...", "name": "..." }, "role": "admin" }
```

---

## Token Architecture

### Access Token
- Algorithm: HS256 JWT
- TTL: 15 minutes
- Signed with `cfg.JWTSecret`
- Claims:
  ```json
  { "sub": "<user_id>", "org_id": "<org_id>", "role": "admin", "iat": 1234567890, "exp": 1234567890 }
  ```

### Refresh Token
- Format: 32 bytes from `crypto/rand`, hex-encoded (64-char string)
- TTL: 7 days
- Stored in the `refresh_tokens` table (column: `token`, `user_id`, `expires_at`, `revoked`)
- Single-use: atomically revoked and replaced on every `/refresh` call
- Revoked on `/logout`

### `tokens.go` API
```go
GenerateAccessToken(userID, orgID, role, secret string) (string, error)
GenerateRefreshToken() (string, error)
ValidateAccessToken(tokenStr, secret string) (*Claims, error)
```

### Middleware
`middleware/auth.go` calls `ValidateAccessToken`, then writes `user_id`, `org_id`, and `role` into the request context — replacing the current placeholder stub. The `/me` route is moved into the protected group so it passes through this middleware.

---

## Service Layer

### Register(ctx, email, password, fullName, orgName)
1. `GetUserByEmail` — return `409 Conflict` if email already registered
2. `bcrypt.GenerateFromPassword` (cost 12)
3. Transaction: `CreateUser` → `CreateOrg` → `CreateOrgMembership(role: "admin")`
4. Generate token pair; `CreateRefreshToken`
5. Return token pair + user info

### Login(ctx, email, password)
1. `GetUserByEmail` — return `401` if not found (generic message; don't leak email existence)
2. `bcrypt.CompareHashAndPassword` — return `401` on mismatch
3. `ListOrgMembershipsByUser` — take first result (V1: one org per user)
4. Generate token pair; `CreateRefreshToken`
5. Return token pair + user info

### Refresh(ctx, refreshToken)
1. `GetRefreshToken` — `401` if revoked, expired, or not found
2. `GetUserByID` to re-fetch current user state
3. `ListOrgMembershipsByUser` to get current org + role for new claims
4. Transaction: `RevokeRefreshToken` + `CreateRefreshToken` (new token)
5. Return new token pair

### Logout(ctx, refreshToken)
1. `RevokeRefreshToken` — idempotent, no error if token not found

### Me(ctx, userID, orgID)
1. `GetUserByID`
2. `GetOrg(orgID)` — orgID comes from JWT claims already in context
3. `GetOrgMembershipByUser` for role
4. Return assembled user + org + role response

---

## Error Handling

| Scenario | Status | Code |
|----------|--------|------|
| Email already registered | 409 | `CONFLICT` |
| Invalid credentials (login) | 401 | `UNAUTHORIZED` |
| Invalid / expired / revoked token | 401 | `UNAUTHORIZED` |
| Missing or malformed request body | 400 | `BAD_REQUEST` |
| Missing required fields | 400 | `BAD_REQUEST` |

Login returns a single generic `401` for both "email not found" and "wrong password" to avoid leaking email existence.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/auth/tokens.go` | Implement 3 token functions |
| `internal/auth/service.go` | Implement 5 service methods |
| `internal/auth/handler.go` | Implement 5 handlers |
| `internal/auth/routes.go` | Add `GET /me` |
| `internal/middleware/auth.go` | Replace placeholder with real JWT validation |
| `go.mod` / `go.sum` | Add `golang-jwt/jwt/v5`, `golang.org/x/crypto` |

## New Test Files

| File | What it covers |
|------|---------------|
| `internal/auth/tokens_test.go` | Generate/validate access token; expired token rejected; tampered signature rejected; refresh token entropy |
| `internal/auth/handler_test.go` | Bad JSON → 400; duplicate email → 409; wrong password → 401; happy paths return correct status + token shape |
| `tests/integration/auth_test.go` | Full flow: register → login → refresh → logout → verify token revoked |

---

## Dependencies Added

- `github.com/golang-jwt/jwt/v5` — JWT sign/verify
- `golang.org/x/crypto` — bcrypt password hashing
