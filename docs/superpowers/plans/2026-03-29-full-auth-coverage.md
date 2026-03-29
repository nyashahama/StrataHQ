# Full Auth Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add onboarding, invitation, and password-reset flows so all three user types (agent, trustee, resident) have complete end-to-end auth.

**Architecture:** Simplify register (drop org_name); add onboarding endpoint that finalises org + creates first scheme; add forgot/reset-password via Redis tokens; enhance /me with wizard_complete + scheme_memberships; create a new invitation package (create/list/resend/revoke/accept); wire a Resend-based notification package for emails.

**Tech Stack:** Go, chi, pgx v5, sqlc, go-redis/v9, Resend email API, bcrypt, crypto/rand.

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Create | `backend/db/migrations/00010_auth_invitations.sql` | orgs.contact_email; org_memberships CHECK update; invitations table |
| Modify | `backend/db/queries/auth.sql` | Add UpdateOrg, ListSchemeMembershipsByUser |
| Create | `backend/db/queries/invitations.sql` | CRUD queries for invitations table |
| Modify | `backend/internal/config/config.go` | Add APP_BASE_URL, EMAIL_FROM |
| Modify | `backend/internal/notification/email.go` | Implement Sender interface + EmailClient |
| Create | `backend/internal/notification/noop.go` | NoopSender for tests |
| Modify | `backend/internal/notification/templates.go` | Invitation + password reset templates |
| Modify | `backend/internal/auth/service.go` | Add Setup, ForgotPassword, ResetPassword; enhance Me; update Register |
| Modify | `backend/internal/auth/handler.go` | Add Setup, ForgotPassword, ResetPassword handlers; update Register |
| Modify | `backend/internal/auth/routes.go` | Add /forgot-password, /reset-password, /onboarding/setup routes |
| Create | `backend/internal/invitation/service.go` | Invitation business logic |
| Create | `backend/internal/invitation/handler.go` | HTTP handlers for invitation endpoints |
| Create | `backend/internal/invitation/routes.go` | Route registration |
| Modify | `backend/cmd/server/main.go` | Wire notification client + invitation handler |
| Modify | `backend/internal/server/router.go` | Register invitation routes |
| Modify | `backend/tests/integration/auth_test.go` | Extend with onboarding + me wizard_complete tests |
| Create | `backend/tests/integration/invitation_test.go` | Full invite→accept→login→me flow |

---

## Task 1: Migration 00010

**Files:**
- Create: `backend/db/migrations/00010_auth_invitations.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- backend/db/migrations/00010_auth_invitations.sql
-- +goose Up

-- Allow trustee/resident roles in org_memberships (previously only admin|agent).
ALTER TABLE org_memberships DROP CONSTRAINT IF EXISTS org_memberships_role_check;
ALTER TABLE org_memberships ADD CONSTRAINT org_memberships_role_check
    CHECK (role IN ('admin', 'agent', 'trustee', 'resident'));

-- Contact email for the managing agent org (set during onboarding wizard).
ALTER TABLE orgs ADD COLUMN IF NOT EXISTS contact_email TEXT;

-- Invitations sent by agents to trustees and residents.
CREATE TABLE invitations (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        NOT NULL REFERENCES orgs(id)    ON DELETE CASCADE,
    scheme_id  UUID        NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    unit_id    UUID        REFERENCES units(id)            ON DELETE SET NULL,
    email      TEXT        NOT NULL,
    full_name  TEXT        NOT NULL,
    role       TEXT        NOT NULL CHECK (role IN ('trustee', 'resident')),
    token      TEXT        NOT NULL UNIQUE,
    status     TEXT        NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'accepted', 'revoked')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX invitations_org_status_idx ON invitations(org_id, status);

-- +goose Down

DROP TABLE IF EXISTS invitations;
ALTER TABLE orgs DROP COLUMN IF EXISTS contact_email;
ALTER TABLE org_memberships DROP CONSTRAINT IF EXISTS org_memberships_role_check;
ALTER TABLE org_memberships ADD CONSTRAINT org_memberships_role_check
    CHECK (role IN ('admin', 'agent'));
```

- [ ] **Step 2: Apply the migration**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
goose -dir db/migrations postgres "$DATABASE_URL" up
```

Expected: `OK    00010_auth_invitations.sql`

---

## Task 2: New sqlc queries — auth.sql additions

**Files:**
- Modify: `backend/db/queries/auth.sql`

- [ ] **Step 1: Append to auth.sql**

```sql
-- name: UpdateOrg :one
UPDATE orgs
SET name = $1, contact_email = $2
WHERE id = $3
RETURNING id, name;

-- name: ListSchemeMembershipsByUser :many
SELECT sm.scheme_id, s.name AS scheme_name, sm.unit_id, sm.role
FROM scheme_memberships sm
JOIN schemes s ON s.id = sm.scheme_id
WHERE sm.user_id = $1
ORDER BY s.name;
```

---

## Task 3: New sqlc queries — invitations.sql

**Files:**
- Create: `backend/db/queries/invitations.sql`

- [ ] **Step 1: Create the file**

```sql
-- backend/db/queries/invitations.sql

-- name: CreateInvitation :one
INSERT INTO invitations (org_id, scheme_id, unit_id, email, full_name, role, token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations WHERE token = $1 LIMIT 1;

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = $1 LIMIT 1;

-- name: ListInvitationsByOrg :many
SELECT * FROM invitations
WHERE org_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: UpdateInvitationStatus :exec
UPDATE invitations SET status = $1 WHERE id = $2;

-- name: UpdateInvitationToken :one
UPDATE invitations
SET token = $1, expires_at = $2
WHERE id = $3
RETURNING *;
```

---

## Task 4: Run sqlc generate

**Files:**
- Modify: `backend/db/gen/*` (regenerated)

- [ ] **Step 1: Regenerate**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
sqlc generate
```

Expected: no errors, new/updated files in `db/gen/`.

- [ ] **Step 2: Verify new functions exist**

```bash
grep -l "UpdateOrg\|ListSchemeMembershipsByUser\|CreateInvitation\|GetInvitationByToken" db/gen/*.go
```

Expected: one or more `.go` files listed.

- [ ] **Step 3: Compile check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add db/migrations/00010_auth_invitations.sql db/queries/auth.sql db/queries/invitations.sql db/gen/
git commit -m "feat(db): add invitations table, contact_email, and new sqlc queries"
```

---

## Task 5: Config additions

**Files:**
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Add APP_BASE_URL and EMAIL_FROM to the Config struct**

In the `Config` struct, after `ResendAPIKey`:

```go
AppBaseURL  string
EmailFrom   string
```

- [ ] **Step 2: Load the values in Load()**

After the `ResendAPIKey` line:

```go
AppBaseURL: os.Getenv("APP_BASE_URL"),
EmailFrom:  getEnv("EMAIL_FROM", "noreply@stratahq.co.za"),
```

- [ ] **Step 3: Require APP_BASE_URL in validate()**

Add to the `required` map:

```go
"APP_BASE_URL": c.AppBaseURL,
```

- [ ] **Step 4: Add to .env**

```bash
grep -q APP_BASE_URL /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend/.env || \
  echo -e "\nAPP_BASE_URL=http://localhost:3000\nEMAIL_FROM=noreply@stratahq.co.za" >> \
  /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend/.env
```

- [ ] **Step 5: Compile check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go .env
git commit -m "feat(config): add APP_BASE_URL and EMAIL_FROM"
```

---

## Task 6: Implement the notification package

**Files:**
- Modify: `backend/internal/notification/email.go`
- Create: `backend/internal/notification/noop.go`
- Modify: `backend/internal/notification/templates.go`

- [ ] **Step 1: Replace email.go with full implementation**

```go
// backend/internal/notification/email.go
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Sender is the interface all notification implementations must satisfy.
type Sender interface {
	SendInvitation(ctx context.Context, to, name, inviteURL string) error
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

type EmailClient struct {
	apiKey    string
	fromAddr  string
	httpClient *http.Client
}

func NewEmailClient(apiKey, fromAddr string) *EmailClient {
	return &EmailClient{
		apiKey:    apiKey,
		fromAddr:  fromAddr,
		httpClient: &http.Client{},
	}
}

func (c *EmailClient) SendInvitation(ctx context.Context, to, name, inviteURL string) error {
	subject, body := InvitationEmail(name, inviteURL)
	return c.send(ctx, to, subject, body)
}

func (c *EmailClient) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	subject, body := PasswordResetEmail(resetURL)
	return c.send(ctx, to, subject, body)
}

func (c *EmailClient) send(ctx context.Context, to, subject, htmlBody string) error {
	payload := map[string]any{
		"from":    c.fromAddr,
		"to":      []string{to},
		"subject": subject,
		"html":    htmlBody,
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("resend: unexpected status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 2: Create noop.go**

```go
// backend/internal/notification/noop.go
package notification

import "context"

// NoopSender captures calls without hitting the network. Use in tests.
type NoopSender struct {
	InvitationsSent  []string
	PasswordResets   []string
}

func (n *NoopSender) SendInvitation(_ context.Context, to, _, _ string) error {
	n.InvitationsSent = append(n.InvitationsSent, to)
	return nil
}

func (n *NoopSender) SendPasswordReset(_ context.Context, to, _ string) error {
	n.PasswordResets = append(n.PasswordResets, to)
	return nil
}
```

- [ ] **Step 3: Replace templates.go**

```go
// backend/internal/notification/templates.go
package notification

import "fmt"

func InvitationEmail(name, inviteURL string) (subject, htmlBody string) {
	subject = "You've been invited to StrataHQ"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Hi %s,</h2>
<p>You've been invited to manage a scheme on StrataHQ.</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Accept invitation
  </a>
</p>
<p style="color:#71717a;font-size:13px">This link expires in 7 days. If you didn't expect this email, you can safely ignore it.</p>
</body></html>`, name, inviteURL)
	return
}

func PasswordResetEmail(resetURL string) (subject, htmlBody string) {
	subject = "Reset your StrataHQ password"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Password reset</h2>
<p>Click the button below to reset your StrataHQ password. This link expires in 1 hour.</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Reset password
  </a>
</p>
<p style="color:#71717a;font-size:13px">If you didn't request a password reset, you can safely ignore this email.</p>
</body></html>`, resetURL)
	return
}
```

- [ ] **Step 4: Compile check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/notification/
git commit -m "feat(notification): implement Resend email client and templates"
```

---

## Task 7: Auth — simplify register, update Servicer, add new service methods

**Files:**
- Modify: `backend/internal/auth/service.go`

This task updates the service. The handler and routes are updated in Tasks 8 and 9.

- [ ] **Step 1: Update imports in service.go**

Add to the import block:
```go
"crypto/rand"
"encoding/hex"
"fmt"

"github.com/stratahq/backend/internal/notification"
```

- [ ] **Step 2: Update MeResponse to the enhanced shape**

Replace the existing `MeResponse` struct:

```go
type SchemeMembership struct {
	SchemeID   string  `json:"scheme_id"`
	SchemeName string  `json:"scheme_name"`
	UnitID     *string `json:"unit_id"`
	Role       string  `json:"role"`
}

type MeResponse struct {
	ID                string             `json:"id"`
	Email             string             `json:"email"`
	FullName          string             `json:"full_name"`
	Org               OrgInfo            `json:"org"`
	Role              string             `json:"role"`
	WizardComplete    bool               `json:"wizard_complete"`
	SchemeMemberships []SchemeMembership `json:"scheme_memberships"`
}
```

- [ ] **Step 3: Add SetupResponse**

After `MeResponse`:

```go
type SetupResponse struct {
	Org    OrgInfo `json:"org"`
	Scheme struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"scheme"`
}
```

- [ ] **Step 4: Update the Servicer interface**

Replace the current `Servicer` interface:

```go
type Servicer interface {
	Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID, orgID string) (*MeResponse, error)
	Setup(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, password string) error
}
```

- [ ] **Step 5: Add sender field to Service struct and update NewService**

Add `sender notification.Sender` to the Service struct:

```go
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	sender        notification.Sender
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, cache *redis.Client, sender notification.Sender, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		sender:        sender,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}
```

- [ ] **Step 6: Update Register — remove orgName parameter**

Replace the existing `Register` method:

```go
func (s *Service) Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	_, err := s.db.Q.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var user dbgen.User
	var org dbgen.Org

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			FullName:     fullName,
		})
		if txErr != nil {
			return txErr
		}
		org, txErr = q.CreateOrg(ctx, "") // org name set during onboarding
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  org.ID,
			Role:   "admin",
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, org.ID.String(), "admin")
}
```

- [ ] **Step 7: Add Setup method**

```go
func (s *Service) Setup(ctx context.Context, orgID, orgName, contactEmail, schemeName, schemeAddress string, unitCount int32) (*SetupResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org id: %w", err)
	}

	var resp SetupResponse

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		orgRow, txErr := q.UpdateOrg(ctx, dbgen.UpdateOrgParams{
			Name:         orgName,
			ContactEmail: pgtype.Text{String: contactEmail, Valid: true},
			ID:           pgtype.UUID{Bytes: oid, Valid: true},
		})
		if txErr != nil {
			return txErr
		}
		resp.Org = OrgInfo{ID: orgRow.ID.String(), Name: orgRow.Name}

		scheme, txErr := q.CreateScheme(ctx, dbgen.CreateSchemeParams{
			OrgID:     pgtype.UUID{Bytes: oid, Valid: true},
			Name:      schemeName,
			Address:   schemeAddress,
			UnitCount: unitCount,
		})
		if txErr != nil {
			return txErr
		}
		resp.Scheme.ID = scheme.ID.String()
		resp.Scheme.Name = scheme.Name
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
```

- [ ] **Step 8: Add ForgotPassword method**

```go
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.db.Q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // silent — no enumeration
	}
	if err != nil {
		return err
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)

	key := "pwreset:" + token
	if err := s.cache.Set(ctx, key, user.ID.String(), time.Hour).Err(); err != nil {
		return err
	}

	resetURL := s.appBaseURL + "/auth/reset-password?token=" + token
	return s.sender.SendPasswordReset(ctx, user.Email, resetURL)
}
```

Wait — the Service struct needs `appBaseURL`. Add it to the struct and NewService:

```go
type Service struct {
	db            *database.Pool
	cache         *redis.Client
	sender        notification.Sender
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
	appBaseURL    string
}

func NewService(db *database.Pool, cache *redis.Client, sender notification.Sender, jwtSecret, appBaseURL string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		cache:         cache,
		sender:        sender,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
		appBaseURL:    appBaseURL,
	}
}
```

- [ ] **Step 9: Add ResetPassword method**

```go
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	key := "pwreset:" + token
	userIDStr, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		return ErrInvalidToken
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		if txErr := q.UpdateUserPassword(ctx, dbgen.UpdateUserPasswordParams{
			ID:           pgtype.UUID{Bytes: uid, Valid: true},
			PasswordHash: string(hash),
		}); txErr != nil {
			return txErr
		}
		return q.RevokeAllUserRefreshTokens(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	})
	if err != nil {
		return err
	}

	s.cache.Del(ctx, key)
	return nil
}
```

- [ ] **Step 10: Enhance Me method**

Replace the existing `Me` method:

```go
func (s *Service) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.db.Q.GetUserByID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}
	org, err := s.db.Q.GetOrg(ctx, pgtype.UUID{Bytes: oid, Valid: true})
	if err != nil {
		return nil, err
	}
	membership, err := s.db.Q.GetOrgMembershipByUser(ctx, dbgen.GetOrgMembershipByUserParams{
		UserID: pgtype.UUID{Bytes: uid, Valid: true},
		OrgID:  pgtype.UUID{Bytes: oid, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	resp := &MeResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Org:      OrgInfo{ID: org.ID.String(), Name: org.Name},
		Role:     membership.Role,
	}

	if membership.Role == "admin" {
		schemes, err := s.db.Q.ListSchemesByOrg(ctx, pgtype.UUID{Bytes: oid, Valid: true})
		if err != nil {
			return nil, err
		}
		resp.WizardComplete = len(schemes) > 0
		for _, sc := range schemes {
			resp.SchemeMemberships = append(resp.SchemeMemberships, SchemeMembership{
				SchemeID:   sc.ID.String(),
				SchemeName: sc.Name,
				UnitID:     nil,
				Role:       "admin",
			})
		}
	} else {
		resp.WizardComplete = true
		memberships, err := s.db.Q.ListSchemeMembershipsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true})
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			sm := SchemeMembership{
				SchemeID:   m.SchemeID.String(),
				SchemeName: m.SchemeName,
				Role:       m.Role,
			}
			if m.UnitID.Valid {
				s := m.UnitID.String()
				sm.UnitID = &s
			}
			resp.SchemeMemberships = append(resp.SchemeMemberships, sm)
		}
	}

	if resp.SchemeMemberships == nil {
		resp.SchemeMemberships = []SchemeMembership{}
	}

	return resp, nil
}
```

- [ ] **Step 11: Add missing pgtype import**

Verify that `github.com/jackc/pgx/v5/pgtype` is in the import block. Add it if missing.

- [ ] **Step 12: Compile check**

```bash
go build ./...
```

Expected: errors about `NewService` call sites (main.go, tests) — fix those in later tasks.

---

## Task 8: Auth handler — update Register, add Setup/ForgotPassword/ResetPassword handlers

**Files:**
- Modify: `backend/internal/auth/handler.go`

- [ ] **Step 1: Update registerRequest — remove OrgName**

Replace the `registerRequest` struct and `Register` handler:

```go
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, and full_name are required")
		return
	}
	res, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if err == ErrEmailExists {
			response.Error(w, http.StatusConflict, "CONFLICT", "email already registered")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "registration failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}
```

- [ ] **Step 2: Add Setup handler**

```go
type setupRequest struct {
	OrgName       string `json:"org_name"`
	ContactEmail  string `json:"contact_email"`
	SchemeName    string `json:"scheme_name"`
	SchemeAddress string `json:"scheme_address"`
	UnitCount     int32  `json:"unit_count"`
}

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can complete onboarding")
		return
	}
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.OrgName == "" || req.ContactEmail == "" || req.SchemeName == "" || req.SchemeAddress == "" || req.UnitCount <= 0 {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "org_name, contact_email, scheme_name, scheme_address, and unit_count are required")
		return
	}
	orgID, _ := r.Context().Value(OrgIDKey).(string)
	res, err := h.service.Setup(r.Context(), orgID, req.OrgName, req.ContactEmail, req.SchemeName, req.SchemeAddress, req.UnitCount)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "onboarding failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}
```

- [ ] **Step 3: Add ForgotPassword and ResetPassword handlers**

```go
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req) // parse best-effort; always return 200
	_ = h.service.ForgotPassword(r.Context(), req.Email)
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "token and password are required")
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		if err == ErrInvalidToken {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired reset token")
			return
		}
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "password reset failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

---

## Task 9: Auth routes update

**Files:**
- Modify: `backend/internal/auth/routes.go`

- [ ] **Step 1: Update Routes() and add OnboardingRoutes()**

Replace the file:

```go
package auth

import "github.com/go-chi/chi/v5"

// Routes registers public auth endpoints (no JWT required).
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

// OnboardingRoutes registers protected onboarding endpoints (JWT required, mounted separately).
func (h *Handler) OnboardingRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/setup", h.Setup)
	return r
}
```

- [ ] **Step 2: Register onboarding routes in router.go**

In `backend/internal/server/router.go`, inside the protected group, after `r.Get("/auth/me", h.Auth.Me)`, add:

```go
r.Mount("/onboarding", h.Auth.OnboardingRoutes())
```

- [ ] **Step 3: Compile check**

```bash
go build ./...
```

Expected: errors about `NewService` signature changes — fix next.

---

## Task 10: Auth handler tests

**Files:**
- Modify: `backend/tests/integration/auth_test.go`

- [ ] **Step 1: Fix newAuthHandler to match new NewService signature**

Update the helper (it now needs a sender and appBaseURL):

```go
func newAuthHandler(t *testing.T) *auth.Handler {
	t.Helper()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	sender := &notification.NoopSender{}
	svc := auth.NewService(pool, testRedis, sender, testJWTSigningKey, "http://localhost:3000", 15*time.Minute, 7*24*time.Hour)
	return auth.NewHandler(svc)
}
```

Add `"github.com/stratahq/backend/internal/notification"` to imports.

- [ ] **Step 2: Update TestAuth_RegisterLoginRefreshLogout — remove org_name from register body**

Find the line:
```go
regBody, _ := json.Marshal(map[string]string{
    "email": email, "password": testPassword,
    "full_name": "Integration User", "org_name": "Integration Org",
})
```
Replace with:
```go
regBody, _ := json.Marshal(map[string]string{
    "email": email, "password": testPassword,
    "full_name": "Integration User",
})
```

- [ ] **Step 3: Add TestAuth_SetupOnboarding**

```go
func TestAuth_SetupOnboarding(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// Register (creates user + empty-name org)
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Setup User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", w.Code, w.Body)
	}
	var regResp struct{ Data auth.AuthResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&regResp)
	accessToken := regResp.Data.AccessToken

	// /me before onboarding → wizard_complete=false
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = withAuthContext(req, regResp.Data.AccessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me pre-onboarding: status=%d", w.Code)
	}
	var meResp struct{ Data auth.MeResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&meResp)
	if meResp.Data.WizardComplete {
		t.Error("expected wizard_complete=false before onboarding")
	}

	// POST /onboarding/setup
	setupBody, _ := json.Marshal(map[string]any{
		"org_name": "Sunset Heights", "contact_email": "admin@sunset.co.za",
		"scheme_name": "Sunset Heights Body Corporate", "scheme_address": "1 Kloof St, Cape Town",
		"unit_count": 12,
	})
	req = httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status=%d body=%s", w.Code, w.Body)
	}
	var setupResp struct {
		Data auth.SetupResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&setupResp)
	if setupResp.Data.Org.Name != "Sunset Heights" {
		t.Errorf("setup: org name=%q, want Sunset Heights", setupResp.Data.Org.Name)
	}
	if setupResp.Data.Scheme.Name != "Sunset Heights Body Corporate" {
		t.Errorf("setup: scheme name=%q", setupResp.Data.Scheme.Name)
	}

	// /me after onboarding → wizard_complete=true, scheme_memberships populated
	req = httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	json.NewDecoder(w.Body).Decode(&meResp)
	if !meResp.Data.WizardComplete {
		t.Error("expected wizard_complete=true after onboarding")
	}
	if len(meResp.Data.SchemeMemberships) != 1 {
		t.Errorf("expected 1 scheme membership, got %d", len(meResp.Data.SchemeMemberships))
	}

	// Setup with non-admin → 403
	req = httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withNonAdminContext(req)
	w = httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("non-admin setup: status=%d, want 403", w.Code)
	}
}
```

- [ ] **Step 4: Add withAuthContext and withNonAdminContext helpers**

Add at the bottom of auth_test.go:

```go
// withAuthContext injects auth claims extracted from the given access token into the request context.
func withAuthContext(r *http.Request, accessToken, jwtSecret string) *http.Request {
	claims, err := auth.ValidateAccessToken(accessToken, jwtSecret)
	if err != nil {
		panic("withAuthContext: invalid token: " + err.Error())
	}
	ctx := context.WithValue(r.Context(), auth.UserIDKey, claims.Subject)
	ctx = context.WithValue(ctx, auth.OrgIDKey, claims.OrgID)
	ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)
	return r.WithContext(ctx)
}

func withNonAdminContext(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey, "00000000-0000-0000-0000-000000000001")
	ctx = context.WithValue(ctx, auth.OrgIDKey, "00000000-0000-0000-0000-000000000002")
	ctx = context.WithValue(ctx, auth.RoleKey, "trustee")
	return r.WithContext(ctx)
}
```

- [ ] **Step 5: Add TestAuth_ForgotResetPassword**

```go
func TestAuth_ForgotResetPassword(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// Register user first
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Reset User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}

	// ForgotPassword with known email → 200
	fpBody, _ := json.Marshal(map[string]string{"email": email})
	req = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(fpBody))
	w = httptest.NewRecorder()
	h.ForgotPassword(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("forgot-password: status=%d, want 200", w.Code)
	}

	// ForgotPassword with unknown email → still 200 (no enumeration)
	fpBody, _ = json.Marshal(map[string]string{"email": "nobody@example.com"})
	req = httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(fpBody))
	w = httptest.NewRecorder()
	h.ForgotPassword(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("forgot-password unknown: status=%d, want 200", w.Code)
	}

	// Read the token from Redis directly to test reset-password
	ctx := context.Background()
	keys, err := testRedis.Keys(ctx, "pwreset:*").Result()
	if err != nil || len(keys) == 0 {
		t.Fatal("no pwreset key found in redis after forgot-password")
	}
	token := strings.TrimPrefix(keys[0], "pwreset:")

	// ResetPassword with correct token → 204
	rpBody, _ := json.Marshal(map[string]string{"token": token, "password": "NewP@ssw0rd!"})
	req = httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(rpBody))
	w = httptest.NewRecorder()
	h.ResetPassword(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("reset-password: status=%d body=%s, want 204", w.Code, w.Body)
	}

	// Login with new password succeeds
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": "NewP@ssw0rd!"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("login after reset: status=%d, want 200", w.Code)
	}

	// Login with OLD password fails
	loginBody, _ = json.Marshal(map[string]string{"email": email, "password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("old password login: status=%d, want 401", w.Code)
	}

	// Token is deleted after use → 401 on reuse
	req = httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewReader(rpBody))
	w = httptest.NewRecorder()
	h.ResetPassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("reuse token: status=%d, want 401", w.Code)
	}
}
```

- [ ] **Step 6: Run auth integration tests**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test -v -tags integration ./tests/integration/ -run TestAuth
```

Expected: all `TestAuth_*` tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/auth/ tests/integration/auth_test.go
git commit -m "feat(auth): simplify register, add onboarding, forgot/reset password, enhance /me"
```

---

## Task 11: Invitation service

**Files:**
- Create: `backend/internal/invitation/service.go`

- [ ] **Step 1: Create the file**

```go
// backend/internal/invitation/service.go
package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/stratahq/backend/internal/auth"
	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound       = errors.New("invitation not found")
	ErrForbidden      = errors.New("invitation belongs to a different org")
	ErrInvalidToken   = errors.New("invalid, expired, or already used invitation token")
	ErrEmailExists    = errors.New("email already registered")
)

// CreateParams holds input for creating an invitation.
type CreateParams struct {
	Email     string
	FullName  string
	Role      string // trustee | resident
	SchemeID  string
	UnitID    string // required when role == "resident", otherwise empty
}

// InvitationResponse is what the API returns for an invitation record.
type InvitationResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	SchemeID  string    `json:"scheme_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VerifyResponse is returned by the public verify-token endpoint.
type VerifyResponse struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	SchemeID string `json:"scheme_id"`
}

// Servicer is the interface Handler depends on.
type Servicer interface {
	Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error)
	List(ctx context.Context, orgID string) ([]InvitationResponse, error)
	Resend(ctx context.Context, orgID, invitationID, appBaseURL string) (*InvitationResponse, error)
	Revoke(ctx context.Context, orgID, invitationID string) error
	Verify(ctx context.Context, token string) (*VerifyResponse, error)
	Accept(ctx context.Context, token, password string) (*auth.AuthResponse, error)
}

// Service implements Servicer.
type Service struct {
	db            *database.Pool
	sender        notification.Sender
	jwtSecret     string
	jwtExpiry     time.Duration
	refreshExpiry time.Duration
}

func NewService(db *database.Pool, sender notification.Sender, jwtSecret string, jwtExpiry, refreshExpiry time.Duration) *Service {
	return &Service{
		db:            db,
		sender:        sender,
		jwtSecret:     jwtSecret,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) Create(ctx context.Context, orgID string, p CreateParams, appBaseURL string) (*InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	sid, err := uuid.Parse(p.SchemeID)
	if err != nil {
		return nil, errors.New("invalid scheme_id")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	var unitID pgtype.UUID
	if p.UnitID != "" {
		uid, err := uuid.Parse(p.UnitID)
		if err != nil {
			return nil, errors.New("invalid unit_id")
		}
		unitID = pgtype.UUID{Bytes: uid, Valid: true}
	}

	inv, err := s.db.Q.CreateInvitation(ctx, dbgen.CreateInvitationParams{
		OrgID:     pgtype.UUID{Bytes: oid, Valid: true},
		SchemeID:  pgtype.UUID{Bytes: sid, Valid: true},
		UnitID:    unitID,
		Email:     p.Email,
		FullName:  p.FullName,
		Role:      p.Role,
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	inviteURL := appBaseURL + "/auth/invite/" + token
	if err := s.sender.SendInvitation(ctx, p.Email, p.FullName, inviteURL); err != nil {
		return nil, err
	}

	return toResponse(inv), nil
}

func (s *Service) List(ctx context.Context, orgID string) ([]InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	invs, err := s.db.Q.ListInvitationsByOrg(ctx, pgtype.UUID{Bytes: oid, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]InvitationResponse, len(invs))
	for i, inv := range invs {
		out[i] = *toResponse(inv)
	}
	return out, nil
}

func (s *Service) Resend(ctx context.Context, orgID, invitationID, appBaseURL string) (*InvitationResponse, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, ErrForbidden
	}
	iid, err := uuid.Parse(invitationID)
	if err != nil {
		return nil, ErrNotFound
	}

	existing, err := s.db.Q.GetInvitationByID(ctx, pgtype.UUID{Bytes: iid, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if existing.OrgID.Bytes != oid {
		return nil, ErrForbidden
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	inv, err := s.db.Q.UpdateInvitationToken(ctx, dbgen.UpdateInvitationTokenParams{
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		ID:        pgtype.UUID{Bytes: iid, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	inviteURL := appBaseURL + "/auth/invite/" + token
	if err := s.sender.SendInvitation(ctx, inv.Email, inv.FullName, inviteURL); err != nil {
		return nil, err
	}

	return toResponse(inv), nil
}

func (s *Service) Revoke(ctx context.Context, orgID, invitationID string) error {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return ErrForbidden
	}
	iid, err := uuid.Parse(invitationID)
	if err != nil {
		return ErrNotFound
	}

	existing, err := s.db.Q.GetInvitationByID(ctx, pgtype.UUID{Bytes: iid, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if existing.OrgID.Bytes != oid {
		return ErrForbidden
	}

	return s.db.Q.UpdateInvitationStatus(ctx, dbgen.UpdateInvitationStatusParams{
		Status: "revoked",
		ID:     pgtype.UUID{Bytes: iid, Valid: true},
	})
}

func (s *Service) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	inv, err := s.db.Q.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if inv.Status != "pending" || inv.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrInvalidToken
	}
	return &VerifyResponse{
		Email:    inv.Email,
		FullName: inv.FullName,
		Role:     inv.Role,
		SchemeID: inv.SchemeID.String(),
	}, nil
}

func (s *Service) Accept(ctx context.Context, token, password string) (*auth.AuthResponse, error) {
	inv, err := s.db.Q.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if inv.Status != "pending" || inv.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	_, err = s.db.Q.GetUserByEmail(ctx, inv.Email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var user dbgen.User

	err = database.WithTxQueries(ctx, s.db, func(q *dbgen.Queries) error {
		var txErr error
		user, txErr = q.CreateUser(ctx, dbgen.CreateUserParams{
			Email:        inv.Email,
			PasswordHash: string(hash),
			FullName:     inv.FullName,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.CreateOrgMembership(ctx, dbgen.CreateOrgMembershipParams{
			UserID: user.ID,
			OrgID:  inv.OrgID,
			Role:   inv.Role,
		})
		if txErr != nil {
			return txErr
		}
		_, txErr = q.UpsertSchemeMembership(ctx, dbgen.UpsertSchemeMembershipParams{
			UserID:   user.ID,
			SchemeID: inv.SchemeID,
			UnitID:   inv.UnitID,
			Role:     inv.Role,
		})
		if txErr != nil {
			return txErr
		}
		return q.UpdateInvitationStatus(ctx, dbgen.UpdateInvitationStatusParams{
			Status: "accepted",
			ID:     inv.ID,
		})
	})
	if err != nil {
		return nil, err
	}

	// Issue JWT pair directly — invitation package controls token creation.
	accessToken, err := auth.GenerateAccessToken(user.ID.String(), inv.OrgID.String(), inv.Role, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	_, err = s.db.Q.CreateRefreshToken(ctx, dbgen.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.refreshExpiry), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &auth.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtExpiry.Seconds()),
		User: auth.UserInfo{
			ID:       user.ID.String(),
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

func toResponse(inv dbgen.Invitation) *InvitationResponse {
	return &InvitationResponse{
		ID:        inv.ID.String(),
		Email:     inv.Email,
		FullName:  inv.FullName,
		Role:      inv.Role,
		SchemeID:  inv.SchemeID.String(),
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt.Time,
	}
}
```

---

## Task 12: Invitation handler

**Files:**
- Create: `backend/internal/invitation/handler.go`

- [ ] **Step 1: Create the file**

```go
// backend/internal/invitation/handler.go
package invitation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service    Servicer
	appBaseURL string
}

func NewHandler(service Servicer, appBaseURL string) *Handler {
	return &Handler{service: service, appBaseURL: appBaseURL}
}

// Create — POST /api/v1/invitations (protected, admin only)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.OrgIDKey).(string)
	role, _ := r.Context().Value(auth.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can send invitations")
		return
	}

	var req struct {
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
		SchemeID string `json:"scheme_id"`
		UnitID   string `json:"unit_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.FullName == "" || req.Role == "" || req.SchemeID == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "email, full_name, role, and scheme_id are required")
		return
	}
	if req.Role == "resident" && req.UnitID == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unit_id is required for residents")
		return
	}

	inv, err := h.service.Create(r.Context(), orgID, CreateParams{
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		SchemeID: req.SchemeID,
		UnitID:   req.UnitID,
	}, h.appBaseURL)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create invitation")
		return
	}
	response.JSON(w, http.StatusCreated, inv)
}

// List — GET /api/v1/invitations (protected, admin only)
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.OrgIDKey).(string)
	role, _ := r.Context().Value(auth.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can list invitations")
		return
	}
	invs, err := h.service.List(r.Context(), orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list invitations")
		return
	}
	response.JSON(w, http.StatusOK, invs)
}

// Resend — POST /api/v1/invitations/{id}/resend (protected, admin only)
func (h *Handler) Resend(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.OrgIDKey).(string)
	role, _ := r.Context().Value(auth.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can resend invitations")
		return
	}
	id := chi.URLParam(r, "id")
	inv, err := h.service.Resend(r.Context(), orgID, id, h.appBaseURL)
	if err != nil {
		switch err {
		case ErrNotFound:
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "invitation not found")
		case ErrForbidden:
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "invitation belongs to a different org")
		default:
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resend invitation")
		}
		return
	}
	response.JSON(w, http.StatusOK, inv)
}

// Revoke — DELETE /api/v1/invitations/{id} (protected, admin only)
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID, _ := r.Context().Value(auth.OrgIDKey).(string)
	role, _ := r.Context().Value(auth.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "only org admins can revoke invitations")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Revoke(r.Context(), orgID, id); err != nil {
		switch err {
		case ErrNotFound:
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "invitation not found")
		case ErrForbidden:
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "invitation belongs to a different org")
		default:
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to revoke invitation")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Verify — GET /api/v1/invitations/{token} (public)
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	v, err := h.service.Verify(r.Context(), token)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired invitation token")
		return
	}
	response.JSON(w, http.StatusOK, v)
}

// Accept — POST /api/v1/invitations/{token}/accept (public)
func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "password is required")
		return
	}
	authResp, err := h.service.Accept(r.Context(), token, req.Password)
	if err != nil {
		switch err {
		case ErrInvalidToken:
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired invitation token")
		case ErrEmailExists:
			response.Error(w, http.StatusConflict, "CONFLICT", "email already registered")
		default:
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to accept invitation")
		}
		return
	}
	response.JSON(w, http.StatusCreated, authResp)
}
```

---

## Task 13: Invitation routes

**Files:**
- Create: `backend/internal/invitation/routes.go`

- [ ] **Step 1: Create the file**

```go
// backend/internal/invitation/routes.go
package invitation

import "github.com/go-chi/chi/v5"

// PublicRoutes registers endpoints that require no authentication.
func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{token}", h.Verify)
	r.Post("/{token}/accept", h.Accept)
	return r
}

// ProtectedRoutes registers endpoints that require a valid JWT (mounted under auth middleware).
func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/{id}/resend", h.Resend)
	r.Delete("/{id}", h.Revoke)
	return r
}
```

---

## Task 14: Wire in main.go and router.go

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/server/router.go`

- [ ] **Step 1: Update main.go**

Add imports:
```go
"github.com/stratahq/backend/internal/invitation"
"github.com/stratahq/backend/internal/notification"
```

After the Redis init, add:

```go
// Notification client
emailClient := notification.NewEmailClient(cfg.ResendAPIKey, cfg.EmailFrom)
```

Update the auth service init (new signature with sender and appBaseURL):

```go
authService := auth.NewService(db, rdb, emailClient, cfg.JWTSecret, cfg.AppBaseURL, cfg.JWTExpiry, cfg.RefreshExpiry)
```

Add invitation service + handler after `Billing`:

```go
invitationService := invitation.NewService(db, emailClient, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
```

Update the `Handlers` struct initialisation — add:

```go
Invitation: invitation.NewHandler(invitationService, cfg.AppBaseURL),
```

- [ ] **Step 2: Update server/router.go**

Add to the `Handlers` struct:

```go
Invitation *invitation.Handler
```

Add import:

```go
"github.com/stratahq/backend/internal/invitation"
```

In the public group, add:

```go
r.Mount("/invitations", h.Invitation.PublicRoutes())
```

In the protected group, after `/auth/me`, add:

```go
r.Mount("/invitations", h.Invitation.ProtectedRoutes())
```

- [ ] **Step 3: Compile check**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/invitation/ internal/server/router.go cmd/server/main.go
git commit -m "feat(invitation): add invitation package and wire into server"
```

---

## Task 15: Invitation handler tests

**Files:**
- Create: `backend/tests/integration/invitation_test.go`

- [ ] **Step 1: Create the file**

```go
//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/invitation"
	"github.com/stratahq/backend/internal/notification"
	"github.com/stratahq/backend/internal/platform/database"
)

const testAppBaseURL = "http://localhost:3000"

func newInvitationHandler(t *testing.T) (*invitation.Handler, *notification.NoopSender) {
	t.Helper()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	sender := &notification.NoopSender{}
	svc := invitation.NewService(pool, sender, testJWTSigningKey, 15*time.Minute, 7*24*time.Hour)
	return invitation.NewHandler(svc, testAppBaseURL), sender
}

// setupAgent creates a registered agent and returns their access token, user ID, and org ID.
func setupAgent(t *testing.T) (accessToken, orgID string) {
	t.Helper()
	h := newAuthHandler(t)
	email := uniqueEmail(t)
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Test Agent",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setupAgent register: %d %s", w.Code, w.Body)
	}
	var resp struct{ Data auth.AuthResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&resp)
	claims, _ := auth.ValidateAccessToken(resp.Data.AccessToken, testJWTSigningKey)
	return resp.Data.AccessToken, claims.OrgID
}

// setupScheme creates a scheme for the given org, returns the scheme ID.
func setupScheme(t *testing.T, accessToken string) string {
	t.Helper()
	h := newAuthHandler(t)
	setupBody, _ := json.Marshal(map[string]any{
		"org_name": "Test Org", "contact_email": "admin@test.co.za",
		"scheme_name": "Test Scheme", "scheme_address": "1 Test St",
		"unit_count": 5,
	})
	req := httptest.NewRequest(http.MethodPost, "/onboarding/setup", bytes.NewReader(setupBody))
	req = withAuthContext(req, accessToken, testJWTSigningKey)
	w := httptest.NewRecorder()
	h.Setup(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setupScheme: %d %s", w.Code, w.Body)
	}
	var resp struct {
		Data auth.SetupResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Data.Scheme.ID
}

func TestInvitation_CreateListRevokeResend(t *testing.T) {
	ih, sender := newInvitationHandler(t)
	agentToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, agentToken)

	// Create invitation
	body, _ := json.Marshal(map[string]string{
		"email": "trustee@example.com", "full_name": "Jane Trustee",
		"role": "trustee", "scheme_id": schemeID,
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = withOrgRoleContext(req, orgID, "admin")
	w := httptest.NewRecorder()
	ih.Create(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create invitation: %d %s", w.Code, w.Body)
	}
	if len(sender.InvitationsSent) != 1 || sender.InvitationsSent[0] != "trustee@example.com" {
		t.Errorf("expected 1 invitation email sent, got %v", sender.InvitationsSent)
	}
	var invResp struct {
		Data invitation.InvitationResponse `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&invResp)
	invID := invResp.Data.ID
	if invID == "" {
		t.Fatal("create: missing invitation id")
	}

	// List — should have 1 pending
	req = httptest.NewRequest(http.MethodGet, "/invitations", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	var listResp struct{ Data []invitation.InvitationResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("list: expected 1, got %d", len(listResp.Data))
	}

	// Resend — sends a new email
	req = httptest.NewRequest(http.MethodPost, "/invitations/"+invID+"/resend", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.Resend(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("resend: %d %s", w.Code, w.Body)
	}
	if len(sender.InvitationsSent) != 2 {
		t.Errorf("resend: expected 2 emails sent, got %d", len(sender.InvitationsSent))
	}

	// Revoke
	req = httptest.NewRequest(http.MethodDelete, "/invitations/"+invID, nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.Revoke(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", w.Code, w.Body)
	}

	// List after revoke — should be empty (only returns pending)
	req = httptest.NewRequest(http.MethodGet, "/invitations", nil)
	req = withOrgRoleContext(req, orgID, "admin")
	w = httptest.NewRecorder()
	ih.List(w, req)
	json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp.Data) != 0 {
		t.Errorf("list after revoke: expected 0, got %d", len(listResp.Data))
	}
}

func TestInvitation_AcceptFlow(t *testing.T) {
	ih, _ := newInvitationHandler(t)
	agentToken, orgID := setupAgent(t)
	schemeID := setupScheme(t, agentToken)

	// Create invitation
	body, _ := json.Marshal(map[string]string{
		"email": "newtrust@example.com", "full_name": "New Trustee",
		"role": "trustee", "scheme_id": schemeID,
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req = withOrgRoleContext(req, orgID, "admin")
	w := httptest.NewRecorder()
	ih.Create(w, req)
	var invResp struct{ Data invitation.InvitationResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&invResp)

	// Verify token — need to read token from DB
	ctx := context.Background()
	pool := &database.Pool{Pool: testDB, Q: dbgen.New(testDB)}
	invs, _ := pool.Q.ListInvitationsByOrg(ctx, pgtype(orgID))
	if len(invs) == 0 {
		t.Fatal("no invitations found")
	}
	token := invs[0].Token

	// Verify endpoint
	req = httptest.NewRequest(http.MethodGet, "/invitations/"+token, nil)
	w = httptest.NewRecorder()
	ih.Verify(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", w.Code, w.Body)
	}
	var vResp struct{ Data invitation.VerifyResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&vResp)
	if vResp.Data.Email != "newtrust@example.com" {
		t.Errorf("verify: email=%q", vResp.Data.Email)
	}

	// Accept invitation
	acceptBody, _ := json.Marshal(map[string]string{"password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/invitations/"+token+"/accept", bytes.NewReader(acceptBody))
	w = httptest.NewRecorder()
	ih.Accept(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("accept: %d %s", w.Code, w.Body)
	}
	var authResp struct{ Data auth.AuthResponse `json:"data"` }
	json.NewDecoder(w.Body).Decode(&authResp)
	if authResp.Data.AccessToken == "" {
		t.Fatal("accept: missing access token")
	}

	// Validate JWT claims — role should be trustee
	claims, err := auth.ValidateAccessToken(authResp.Data.AccessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("accept: invalid token: %v", err)
	}
	if claims.Role != "trustee" {
		t.Errorf("accept: role=%q, want trustee", claims.Role)
	}

	// Verify token → 401 after acceptance (status changed to accepted)
	req = httptest.NewRequest(http.MethodGet, "/invitations/"+token, nil)
	w = httptest.NewRecorder()
	ih.Verify(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("verify after accept: status=%d, want 401", w.Code)
	}

	// Duplicate email → 409
	req = httptest.NewRequest(http.MethodPost, "/invitations/"+token+"/accept", bytes.NewReader(acceptBody))
	w = httptest.NewRecorder()
	ih.Accept(w, req)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusConflict {
		t.Errorf("double accept: status=%d, want 401 or 409", w.Code)
	}
}

// withOrgRoleContext injects org + role context without a full JWT (simulates auth middleware).
func withOrgRoleContext(r *http.Request, orgID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.OrgIDKey, orgID)
	ctx = context.WithValue(ctx, auth.RoleKey, role)
	return r.WithContext(ctx)
}

// pgtype parses a UUID string into pgtype.UUID — helper for test DB queries.
func pgtype(id string) pgtype.UUID {
	uid, _ := uuid.Parse(id)
	return pgtype.UUID{Bytes: uid, Valid: true}
}
```

> **Note:** The `pgtype` helper function at the bottom conflicts with the imported package name. Rename the helper to `mustUUID` and update its usage:

```go
func mustUUID(id string) pgtype2.UUID {
    uid, _ := uuid.Parse(id)
    return pgtype2.UUID{Bytes: uid, Valid: true}
}
```

Add import alias: `pgtype2 "github.com/jackc/pgx/v5/pgtype"` and `"github.com/google/uuid"`.

Replace `pgtype(orgID)` with `mustUUID(orgID)` in the test.

- [ ] **Step 2: Run invitation integration tests**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app/backend
go test -v -tags integration ./tests/integration/ -run TestInvitation
```

Expected: all `TestInvitation_*` tests pass.

- [ ] **Step 3: Run all integration tests**

```bash
go test -v -tags integration ./tests/integration/
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add tests/integration/
git commit -m "test(integration): add invitation flow and extended auth tests"
```

---

## Final Verification

- [ ] Full build passes: `go build ./...`
- [ ] All integration tests pass: `go test -tags integration ./tests/integration/ -v`
- [ ] `POST /auth/register` no longer accepts `org_name` (returns 201 without it)
- [ ] `POST /api/v1/onboarding/setup` returns `{org, scheme}` and sets `wizard_complete=true` on /me
- [ ] `GET /auth/me` returns `wizard_complete` and `scheme_memberships`
- [ ] `POST /auth/forgot-password` always returns 200
- [ ] `POST /auth/reset-password` returns 204 on success, 401 on bad token
- [ ] Invitation CRUD endpoints return correct status codes
- [ ] Accepted invitation → valid JWT with correct role
