# Early Access Gating Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace open signup with a gated early access flow — users submit a request form, the owner manually approves via an admin page, and approved users receive an email to set their password.

**Architecture:** A new `early_access_requests` table stores submissions. The backend exposes a public POST endpoint for submissions and admin-protected GET/PATCH endpoints for review. On approval, the backend creates a user account and sends a password-setup email (using the existing forgot-password token mechanism). The frontend removes all demo links and replaces landing page CTAs with "Request early access".

**Tech Stack:** Go (chi, pgx, sqlc pattern), Next.js App Router, server actions, Tailwind. No Supabase in frontend — all persistence goes through the Go backend at `BACKEND_URL`.

---

## File Map

### Backend — new files
- `backend/db/migrations/00013_early_access.sql` — DB schema for `early_access_requests`
- `backend/db/queries/early_access.sql` — sqlc query stubs (reference only)
- `backend/db/gen/early_access.sql.go` — hand-written sqlc-style query layer
- `backend/internal/earlyaccess/service.go` — business logic (submit, list, approve, reject)
- `backend/internal/earlyaccess/handler.go` — HTTP handlers
- `backend/internal/earlyaccess/routes.go` — route registration

### Backend — modified files
- `backend/internal/notification/email.go` — add `SendEarlyAccessApproval` method
- `backend/internal/notification/templates.go` — add `EarlyAccessApprovalEmail` template
- `backend/internal/notification/noop.go` — stub `SendEarlyAccessApproval`
- `backend/internal/server/router.go` — mount earlyaccess routes + add `EarlyAccess` to `Handlers`
- `backend/cmd/server/main.go` — wire earlyaccess service and handler

### Frontend — new files
- `app/early-access/page.tsx` — public request form
- `app/early-access/success/page.tsx` — confirmation screen
- `lib/early-access-actions.ts` — server action to POST submission to backend
- `app/admin/early-access/page.tsx` — admin-only review/approve/reject page
- `lib/early-access-api.ts` — typed fetch helpers for admin page

### Frontend — modified files
- `app/page.tsx` — remove `DemoTeaser` import/usage
- `components/Nav.tsx` — replace "Live demo" + "Get started" with "Request early access"
- `components/Hero.tsx` — replace "Start free trial" + "Try interactive demo" CTAs
- `components/CTASection.tsx` — replace both CTA buttons
- `app/auth/register/page.tsx` — redirect to `/early-access`

### Frontend — deleted files
- `app/demo/` (entire directory)
- `components/DemoTeaser.tsx`

---

## Task 1: DB migration — early_access_requests table

**Files:**
- Create: `backend/db/migrations/00013_early_access.sql`

- [ ] **Step 1: Write the migration**

```sql
-- backend/db/migrations/00013_early_access.sql
CREATE TYPE early_access_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE early_access_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name   TEXT NOT NULL,
    email       TEXT NOT NULL,
    scheme_name TEXT NOT NULL,
    unit_count  INTEGER NOT NULL,
    status      early_access_status NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_early_access_requests_status ON early_access_requests (status);
CREATE INDEX idx_early_access_requests_email  ON early_access_requests (email);
```

- [ ] **Step 2: Apply migration**

```bash
cd backend
psql "$DATABASE_URL" -f db/migrations/00013_early_access.sql
```

Expected: `CREATE TYPE`, `CREATE TABLE`, `CREATE INDEX` (×2)

- [ ] **Step 3: Commit**

```bash
cd backend
git add db/migrations/00013_early_access.sql
git commit -m "feat(db): add early_access_requests table"
```

---

## Task 2: DB query layer (hand-written sqlc-style)

**Files:**
- Create: `backend/db/gen/early_access.sql.go`

- [ ] **Step 1: Write the query layer**

```go
// backend/db/gen/early_access.sql.go
package dbgen

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EarlyAccessStatus string

const (
	EarlyAccessStatusPending  EarlyAccessStatus = "pending"
	EarlyAccessStatusApproved EarlyAccessStatus = "approved"
	EarlyAccessStatusRejected EarlyAccessStatus = "rejected"
)

type EarlyAccessRequest struct {
	ReviewedAt *time.Time        `json:"reviewed_at"`
	CreatedAt  time.Time         `json:"created_at"`
	ID         uuid.UUID         `json:"id"`
	FullName   string            `json:"full_name"`
	Email      string            `json:"email"`
	SchemeName string            `json:"scheme_name"`
	Status     EarlyAccessStatus `json:"status"`
	UnitCount  int32             `json:"unit_count"`
}

type CreateEarlyAccessRequestParams struct {
	FullName   string
	Email      string
	SchemeName string
	UnitCount  int32
}

const createEarlyAccessRequest = `
INSERT INTO early_access_requests (full_name, email, scheme_name, unit_count)
VALUES ($1, $2, $3, $4)
RETURNING id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
`

func (q *Queries) CreateEarlyAccessRequest(ctx context.Context, p CreateEarlyAccessRequestParams) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, createEarlyAccessRequest, p.FullName, p.Email, p.SchemeName, p.UnitCount)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}

const listEarlyAccessRequests = `
SELECT id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
FROM early_access_requests
ORDER BY created_at DESC
`

func (q *Queries) ListEarlyAccessRequests(ctx context.Context) ([]EarlyAccessRequest, error) {
	rows, err := q.db.Query(ctx, listEarlyAccessRequests)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []EarlyAccessRequest
	for rows.Next() {
		var r EarlyAccessRequest
		if err := rows.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

const getEarlyAccessRequest = `
SELECT id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
FROM early_access_requests
WHERE id = $1
`

func (q *Queries) GetEarlyAccessRequest(ctx context.Context, id uuid.UUID) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, getEarlyAccessRequest, id)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}

type UpdateEarlyAccessStatusParams struct {
	ID     uuid.UUID
	Status EarlyAccessStatus
}

const updateEarlyAccessStatus = `
UPDATE early_access_requests
SET status = $2, reviewed_at = NOW()
WHERE id = $1
RETURNING id, full_name, email, scheme_name, unit_count, status, created_at, reviewed_at
`

func (q *Queries) UpdateEarlyAccessStatus(ctx context.Context, p UpdateEarlyAccessStatusParams) (EarlyAccessRequest, error) {
	row := q.db.QueryRow(ctx, updateEarlyAccessStatus, p.ID, p.Status)
	var r EarlyAccessRequest
	err := row.Scan(&r.ID, &r.FullName, &r.Email, &r.SchemeName, &r.UnitCount, &r.Status, &r.CreatedAt, &r.ReviewedAt)
	return r, err
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
cd backend && go build ./...
```

Expected: no output (clean build)

- [ ] **Step 3: Commit**

```bash
git add backend/db/gen/early_access.sql.go
git commit -m "feat(db): add early access request query layer"
```

---

## Task 3: Notification — early access approval email

**Files:**
- Modify: `backend/internal/notification/email.go`
- Modify: `backend/internal/notification/templates.go`
- Modify: `backend/internal/notification/noop.go`

- [ ] **Step 1: Add template to templates.go**

In `backend/internal/notification/templates.go`, append:

```go
func EarlyAccessApprovalEmail(name, setPasswordURL string) (subject, htmlBody string) {
	subject = "Your StrataHQ access has been approved"
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
<h2>Hi %s,</h2>
<p>Great news — your early access request for StrataHQ has been approved.</p>
<p>Click the button below to set your password and get started:</p>
<p style="margin:32px 0">
  <a href="%s" style="background:#18181b;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">
    Set your password
  </a>
</p>
<p style="color:#71717a;font-size:13px">This link expires in 24 hours. If you didn't request access, you can safely ignore this email.</p>
</body></html>`, name, setPasswordURL)
	return
}
```

- [ ] **Step 2: Add method to Sender interface and EmailClient**

In `backend/internal/notification/email.go`, extend the `Sender` interface:

```go
type Sender interface {
	SendInvitation(ctx context.Context, to, name, inviteURL string) error
	SendPasswordReset(ctx context.Context, to, resetURL string) error
	SendEarlyAccessApproval(ctx context.Context, to, name, setPasswordURL string) error
}
```

Add the method to `EmailClient`:

```go
func (c *EmailClient) SendEarlyAccessApproval(ctx context.Context, to, name, setPasswordURL string) error {
	subject, body := EarlyAccessApprovalEmail(name, setPasswordURL)
	return c.send(ctx, to, subject, body)
}
```

- [ ] **Step 3: Add stub to NoopSender in noop.go**

```go
func (n *NoopSender) SendEarlyAccessApproval(_ context.Context, to, _, _ string) error {
	n.InvitationsSent = append(n.InvitationsSent, to)
	return nil
}
```

- [ ] **Step 4: Build to verify**

```bash
cd backend && go build ./...
```

Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add backend/internal/notification/
git commit -m "feat(notification): add early access approval email"
```

---

## Task 4: Early access — service

**Files:**
- Create: `backend/internal/earlyaccess/service.go`

The service depends on `auth.Servicer` (to call `Register` + `ForgotPassword`) and `notification.Sender` (to send the approval email). On approval: create the account with a random temp password, immediately generate a reset token via `ForgotPassword`, then send the custom approval email. The `ForgotPassword` method handles token storage and email sending internally — we override only by sending our template using the same token URL that `ForgotPassword` would produce.

Wait — `ForgotPassword` on `auth.Servicer` sends its own email. We want to send a different email (approval template). The cleanest way without refactoring auth: call a new method `CreatePasswordResetToken` that returns the URL without sending email, then we send our template. But that requires adding to `auth.Servicer`.

**Revised approach**: Add `IssuePasswordResetURL(ctx, email) (string, error)` to `auth.Servicer`. This generates and stores the token, returns the full URL, but does NOT send email. The earlyaccess service then sends its own email with that URL.

- [ ] **Step 1: Add IssuePasswordResetURL to auth.Servicer interface (auth/service.go)**

In `backend/internal/auth/service.go`, add to `Servicer` interface:

```go
IssuePasswordResetURL(ctx context.Context, email, appBaseURL string) (string, error)
```

Add the implementation to `Service` (place it near `ForgotPassword`):

```go
// IssuePasswordResetURL generates a password reset token and returns the URL
// without sending any email. Used by earlyaccess approval flow.
func (s *Service) IssuePasswordResetURL(ctx context.Context, email, appBaseURL string) (string, error) {
	// Check user exists
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	key := fmt.Sprintf("reset:%s", token)
	if err := s.rdb.Set(ctx, key, user.ID.String(), 24*time.Hour).Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/auth/reset-password?token=%s", appBaseURL, token), nil
}
```

- [ ] **Step 2: Build auth package to check**

```bash
cd backend && go build ./internal/auth/...
```

Expected: clean

- [ ] **Step 3: Write earlyaccess/service.go**

```go
// backend/internal/earlyaccess/service.go
package earlyaccess

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/stratahq/backend/db/gen"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/notification"
)

var ErrNotFound = errors.New("early access request not found")

type SubmitParams struct {
	FullName   string
	Email      string
	SchemeName string
	UnitCount  int32
}

type RequestResponse struct {
	ReviewedAt *string `json:"reviewed_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	ID         string  `json:"id"`
	FullName   string  `json:"full_name"`
	Email      string  `json:"email"`
	SchemeName string  `json:"scheme_name"`
	Status     string  `json:"status"`
	UnitCount  int32   `json:"unit_count"`
}

type Servicer interface {
	Submit(ctx context.Context, p SubmitParams) (*RequestResponse, error)
	List(ctx context.Context) ([]RequestResponse, error)
	Approve(ctx context.Context, id string) (*RequestResponse, error)
	Reject(ctx context.Context, id string) (*RequestResponse, error)
}

type Service struct {
	db          *dbgen.Queries
	authService auth.Servicer
	notifier    notification.Sender
	appBaseURL  string
}

func NewService(db *dbgen.Queries, authService auth.Servicer, notifier notification.Sender, appBaseURL string) *Service {
	return &Service{db: db, authService: authService, notifier: notifier, appBaseURL: appBaseURL}
}

func (s *Service) Submit(ctx context.Context, p SubmitParams) (*RequestResponse, error) {
	row, err := s.db.CreateEarlyAccessRequest(ctx, dbgen.CreateEarlyAccessRequestParams{
		FullName:   p.FullName,
		Email:      p.Email,
		SchemeName: p.SchemeName,
		UnitCount:  p.UnitCount,
	})
	if err != nil {
		return nil, err
	}
	return toResponse(row), nil
}

func (s *Service) List(ctx context.Context) ([]RequestResponse, error) {
	rows, err := s.db.ListEarlyAccessRequests(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RequestResponse, len(rows))
	for i, r := range rows {
		out[i] = *toResponse(r)
	}
	return out, nil
}

func (s *Service) Approve(ctx context.Context, id string) (*RequestResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}

	req, err := s.db.GetEarlyAccessRequest(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Create account with random temp password (user will reset it)
	tempPass := make([]byte, 16)
	if _, err := rand.Read(tempPass); err != nil {
		return nil, err
	}
	_, regErr := s.authService.Register(ctx, req.Email, hex.EncodeToString(tempPass), req.FullName)
	if regErr != nil && regErr != auth.ErrEmailExists {
		return nil, regErr
	}

	// Generate password reset URL without sending reset email
	setPasswordURL, err := s.authService.IssuePasswordResetURL(ctx, req.Email, s.appBaseURL)
	if err != nil {
		return nil, err
	}

	// Send our approval email with the set-password link
	_ = s.notifier.SendEarlyAccessApproval(ctx, req.Email, req.FullName, setPasswordURL)

	// Mark approved
	updated, err := s.db.UpdateEarlyAccessStatus(ctx, dbgen.UpdateEarlyAccessStatusParams{
		ID:     uid,
		Status: dbgen.EarlyAccessStatusApproved,
	})
	if err != nil {
		return nil, err
	}
	return toResponse(updated), nil
}

func (s *Service) Reject(ctx context.Context, id string) (*RequestResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	updated, err := s.db.UpdateEarlyAccessStatus(ctx, dbgen.UpdateEarlyAccessStatusParams{
		ID:     uid,
		Status: dbgen.EarlyAccessStatusRejected,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toResponse(updated), nil
}

func toResponse(r dbgen.EarlyAccessRequest) *RequestResponse {
	resp := &RequestResponse{
		ID:         r.ID.String(),
		FullName:   r.FullName,
		Email:      r.Email,
		SchemeName: r.SchemeName,
		UnitCount:  r.UnitCount,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.ReviewedAt != nil {
		s := r.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ReviewedAt = &s
	}
	return resp
}
```

- [ ] **Step 4: Build to verify**

```bash
cd backend && go build ./internal/earlyaccess/...
```

Expected: clean

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/service.go backend/internal/earlyaccess/service.go
git commit -m "feat(earlyaccess): add service with submit/list/approve/reject"
```

---

## Task 5: Early access — handler and routes

**Files:**
- Create: `backend/internal/earlyaccess/handler.go`
- Create: `backend/internal/earlyaccess/routes.go`

- [ ] **Step 1: Write handler.go**

```go
// backend/internal/earlyaccess/handler.go
package earlyaccess

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stratahq/backend/internal/auth"
	"github.com/stratahq/backend/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName   string `json:"full_name"`
		Email      string `json:"email"`
		SchemeName string `json:"scheme_name"`
		UnitCount  int32  `json:"unit_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.FullName == "" || req.Email == "" || req.SchemeName == "" || req.UnitCount <= 0 {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "full_name, email, scheme_name, and unit_count are required")
		return
	}
	result, err := h.service.Submit(r.Context(), SubmitParams{
		FullName:   req.FullName,
		Email:      req.Email,
		SchemeName: req.SchemeName,
		UnitCount:  req.UnitCount,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to submit request")
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	results, err := h.service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to list requests")
		return
	}
	response.JSON(w, http.StatusOK, results)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	id := chi.URLParam(r, "id")
	result, err := h.service.Approve(r.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "request not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to approve request")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok || !auth.IsAdminRole(identity.Role) {
		response.Error(w, http.StatusForbidden, response.CodeForbidden, "admin only")
		return
	}
	id := chi.URLParam(r, "id")
	result, err := h.service.Reject(r.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "request not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to reject request")
		return
	}
	response.JSON(w, http.StatusOK, result)
}
```

- [ ] **Step 2: Write routes.go**

```go
// backend/internal/earlyaccess/routes.go
package earlyaccess

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) PublicRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", h.Submit)
	return r
}

func (h *Handler) ProtectedRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/reject", h.Reject)
	return r
}
```

- [ ] **Step 3: Build**

```bash
cd backend && go build ./internal/earlyaccess/...
```

Expected: clean

- [ ] **Step 4: Commit**

```bash
git add backend/internal/earlyaccess/
git commit -m "feat(earlyaccess): add handler and routes"
```

---

## Task 6: Wire early access into router and main

**Files:**
- Modify: `backend/internal/server/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add EarlyAccess to Handlers struct in router.go**

In `backend/internal/server/router.go`, add the import and field:

```go
import (
    // existing imports ...
    "github.com/stratahq/backend/internal/earlyaccess"
)

type Handlers struct {
    // existing fields ...
    EarlyAccess *earlyaccess.Handler
}
```

- [ ] **Step 2: Mount routes in router.go**

In `NewRouter`, inside the `r.Route("/api/v1", ...)` block:

```go
// Public routes group — add alongside existing:
r.Mount("/early-access", h.EarlyAccess.PublicRoutes())

// Protected routes group — add alongside existing:
r.Mount("/admin/early-access", h.EarlyAccess.ProtectedRoutes())
```

- [ ] **Step 3: Wire in main.go**

In `backend/cmd/server/main.go`, add import and wiring:

```go
import (
    // existing ...
    "github.com/stratahq/backend/internal/earlyaccess"
)

// After existing service declarations:
earlyAccessService := earlyaccess.NewService(dbgen.New(db), authService, emailClient, cfg.AppBaseURL)

// In handlers struct:
handlers := server.Handlers{
    // existing ...
    EarlyAccess: earlyaccess.NewHandler(earlyAccessService),
}
```

Note: `dbgen.New(db)` — check how existing services receive queries. If they receive `*dbgen.Queries` directly, pass `dbgen.New(db)`. Look at `auth.NewService(db, ...)` — if `db` is already `*dbgen.Queries`, pass `db`.

- [ ] **Step 4: Build everything**

```bash
cd backend && go build ./...
```

Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add backend/internal/server/router.go backend/cmd/server/main.go
git commit -m "feat(server): wire early access routes"
```

---

## Task 7: Frontend — remove demo, update landing page CTAs

**Files:**
- Delete: `app/demo/layout.tsx`, `app/demo/page.tsx`
- Delete: `components/DemoTeaser.tsx`
- Modify: `app/page.tsx`
- Modify: `components/Nav.tsx`
- Modify: `components/Hero.tsx`
- Modify: `components/CTASection.tsx`

- [ ] **Step 1: Delete demo directory and DemoTeaser**

```bash
rm -rf app/demo components/DemoTeaser.tsx
```

- [ ] **Step 2: Update app/page.tsx — remove DemoTeaser**

Remove the `import DemoTeaser from '@/components/DemoTeaser'` line and the `<DemoTeaser />` usage:

```tsx
import Nav from '@/components/Nav'
import Hero from '@/components/Hero'
import StatsBar from '@/components/StatsBar'
import ProblemSection from '@/components/ProblemSection'
import FeaturesSection from '@/components/FeaturesSection'
import InsightsSection from '@/components/InsightsSection'
import ModulesSection from '@/components/ModulesSection'
import RolesSection from '@/components/RolesSection'
import QuoteSection from '@/components/QuoteSection'
import PricingSection from '@/components/PricingSection'
import CTASection from '@/components/CTASection'
import Footer from '@/components/Footer'

export default function Home() {
  return (
    <>
      <Nav />
      <main>
        <Hero />
        <StatsBar />
        <ProblemSection />
        <FeaturesSection />
        <InsightsSection />
        <ModulesSection />
        <RolesSection />
        <QuoteSection />
        <PricingSection />
        <CTASection />
      </main>
      <Footer />
    </>
  )
}
```

- [ ] **Step 3: Update Nav.tsx — replace demo + get started with early access**

Replace the entire `{/* Actions */}` div:

```tsx
{/* Actions */}
<div className="flex items-center gap-2">
  <Link
    href="/auth/login"
    className="px-[14px] py-[10px] text-[14px] font-medium text-ink-2 bg-transparent
      border border-border-2 rounded hover:bg-hover-subtle transition-colors duration-150 no-underline"
  >
    Log in
  </Link>
  <Link
    href="/early-access"
    className="px-4 py-[10px] text-[14px] font-medium text-white bg-accent
      border border-accent rounded hover:opacity-90 transition-opacity duration-150 no-underline"
  >
    Request early access
  </Link>
</div>
```

Also remove the "Live demo" link from the nav links list (the `<ul>` of nav links stays the same — just remove the demo link from `{/* Actions */}`).

- [ ] **Step 4: Update Hero.tsx — replace CTAs**

Replace the `{/* CTAs */}` div and the note below it:

```tsx
{/* CTAs */}
<div className="flex flex-wrap items-center gap-[10px] mb-4">
  <Link
    href="/early-access"
    className="px-6 py-[10px] text-[15px] font-medium text-white bg-accent border border-accent
      rounded hover:opacity-90 transition-opacity duration-150 no-underline"
  >
    Request early access →
  </Link>
  <Link
    href="#features"
    className="px-6 py-[10px] text-[15px] font-medium text-ink-2 bg-surface border border-border-2
      rounded hover:bg-page transition-colors duration-150 hidden sm:inline-flex no-underline"
  >
    See how it works
  </Link>
</div>

{/* Note */}
<p className="text-[13px] text-muted-2">
  Limited early access · STSMA compliant · Built for South Africa
</p>
```

- [ ] **Step 5: Update CTASection.tsx — replace CTAs**

Replace the entire section content:

```tsx
import Link from 'next/link'

export default function CTASection() {
  return (
    <section className="bg-surface border-t border-border">
      <div className="max-w-container mx-auto px-container">
        <div className="reveal max-w-[600px] mx-auto text-center py-[clamp(56px,9vh,80px)]">
          <h2 className="font-serif text-clamp-cta font-bold tracking-[-0.02em] text-ink leading-[1.2] mb-4">
            Ready to bring order to your scheme?
          </h2>
          <p className="text-[16px] text-ink-2 mb-8 leading-[1.65]">
            We&apos;re onboarding a limited number of schemes. Request early access and we&apos;ll be in touch.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-[10px] mb-[14px]">
            <Link
              href="/early-access"
              className="px-7 py-3 text-[15px] font-medium text-white bg-accent border border-accent
                rounded hover:opacity-90 transition-opacity duration-150 no-underline"
            >
              Request early access →
            </Link>
          </div>
          <p className="text-[13px] text-muted-2">
            Limited spots · STSMA compliant · No credit card needed
          </p>
        </div>
      </div>
    </section>
  )
}
```

- [ ] **Step 6: Verify build**

```bash
npm run build 2>&1 | tail -20
```

Expected: no errors (warnings about demo links being gone are fine)

- [ ] **Step 7: Commit**

```bash
git add app/page.tsx components/Nav.tsx components/Hero.tsx components/CTASection.tsx
git rm app/demo/layout.tsx app/demo/page.tsx components/DemoTeaser.tsx
git commit -m "feat(landing): remove demo, replace CTAs with early access"
```

---

## Task 8: Redirect register page to early access

**Files:**
- Modify: `app/auth/register/page.tsx`

- [ ] **Step 1: Replace register page with redirect**

```tsx
// app/auth/register/page.tsx
import { redirect } from 'next/navigation'

export default function RegisterPage() {
  redirect('/early-access')
}
```

- [ ] **Step 2: Commit**

```bash
git add app/auth/register/page.tsx
git commit -m "feat(auth): redirect /auth/register to /early-access"
```

---

## Task 9: Early access form page

**Files:**
- Create: `lib/early-access-actions.ts`
- Create: `app/early-access/page.tsx`
- Create: `app/early-access/success/page.tsx`

- [ ] **Step 1: Write lib/early-access-actions.ts**

```ts
// lib/early-access-actions.ts
'use server'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

export type EarlyAccessSubmitInput = {
  full_name: string
  email: string
  scheme_name: string
  unit_count: number
}

export async function submitEarlyAccessRequest(
  data: EarlyAccessSubmitInput,
): Promise<{ ok: true } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/early-access`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    return { error: 'Failed to submit request — please try again' }
  }
  return { ok: true }
}
```

- [ ] **Step 2: Write app/early-access/page.tsx**

```tsx
// app/early-access/page.tsx
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'
import { submitEarlyAccessRequest } from '@/lib/early-access-actions'

export default function EarlyAccessPage() {
  const router = useRouter()
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [schemeName, setSchemeName] = useState('')
  const [unitCount, setUnitCount] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const count = parseInt(unitCount, 10)
    if (isNaN(count) || count <= 0) {
      setError('Unit count must be a positive number')
      return
    }
    setLoading(true)
    const result = await submitEarlyAccessRequest({
      full_name: fullName,
      email,
      scheme_name: schemeName,
      unit_count: count,
    })
    setLoading(false)
    if ('error' in result) {
      setError(result.error)
      return
    }
    router.push('/early-access/success')
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <Link href="/" className="flex items-center gap-2 mb-8 no-underline">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </Link>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
          Request early access
        </h1>
        <p className="text-muted text-sm mb-8">
          We&apos;re onboarding schemes selectively. Fill in your details and we&apos;ll review your request.
        </p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="full_name" className="block text-sm font-medium text-ink mb-1">
              Full name
            </label>
            <input
              id="full_name"
              type="text"
              autoComplete="name"
              required
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="Jane Smith"
            />
          </div>

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-ink mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="scheme_name" className="block text-sm font-medium text-ink mb-1">
              Scheme name
            </label>
            <input
              id="scheme_name"
              type="text"
              required
              value={schemeName}
              onChange={(e) => setSchemeName(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="Rosewood Estate"
            />
          </div>

          <div>
            <label htmlFor="unit_count" className="block text-sm font-medium text-ink mb-1">
              Number of units
            </label>
            <input
              id="unit_count"
              type="number"
              min="1"
              required
              value={unitCount}
              onChange={(e) => setUnitCount(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="24"
            />
          </div>

          {error && <p className="text-sm text-red">{error}</p>}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
          >
            {loading ? 'Submitting…' : 'Submit request'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-muted">
          Already have an account?{' '}
          <Link href="/auth/login" className="text-accent hover:underline font-medium">
            Log in
          </Link>
        </p>
      </div>
    </main>
  )
}
```

- [ ] **Step 3: Write app/early-access/success/page.tsx**

```tsx
// app/early-access/success/page.tsx
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'

export default function EarlyAccessSuccessPage() {
  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12 text-center">
        <Link href="/" className="flex items-center justify-center gap-2 mb-10 no-underline">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </Link>

        <div className="w-14 h-14 rounded-full bg-accent-dim flex items-center justify-center mx-auto mb-6">
          <svg
            viewBox="0 0 24 24"
            className="w-7 h-7 fill-none stroke-accent"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
          Request received
        </h1>
        <p className="text-muted text-sm leading-relaxed mb-8">
          We&apos;ll review your request and send you an email when your access is ready. This usually takes 1–2 business days.
        </p>

        <Link
          href="/"
          className="inline-flex items-center gap-2 rounded border border-border bg-surface px-5 py-2.5 text-sm font-medium text-ink hover:bg-border transition-colors no-underline"
        >
          Back to home
        </Link>
      </div>
    </main>
  )
}
```

- [ ] **Step 4: Verify build**

```bash
npm run build 2>&1 | tail -20
```

Expected: clean

- [ ] **Step 5: Commit**

```bash
git add lib/early-access-actions.ts app/early-access/
git commit -m "feat(early-access): add request form and success page"
```

---

## Task 10: Admin early access review page

**Files:**
- Create: `lib/early-access-api.ts`
- Create: `app/admin/early-access/page.tsx`

- [ ] **Step 1: Write lib/early-access-api.ts**

```ts
// lib/early-access-api.ts
'use server'

import { cookies } from 'next/headers'
import { readApiData } from './api-contract'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

export type EarlyAccessRequest = {
  id: string
  full_name: string
  email: string
  scheme_name: string
  unit_count: number
  status: 'pending' | 'approved' | 'rejected'
  created_at: string
  reviewed_at?: string
}

async function getAccessToken(): Promise<string | null> {
  const cookieStore = await cookies()
  return cookieStore.get('sh_access')?.value ?? null
}

export async function listEarlyAccessRequests(): Promise<EarlyAccessRequest[]> {
  const token = await getAccessToken()
  if (!token) return []
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: 'no-store',
  })
  if (!res.ok) return []
  return readApiData<EarlyAccessRequest[]>(res)
}

export async function approveEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const token = await getAccessToken()
  if (!token) return { error: 'Not authenticated' }
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access/${id}/approve`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return { error: 'Failed to approve' }
  return { ok: true }
}

export async function rejectEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const token = await getAccessToken()
  if (!token) return { error: 'Not authenticated' }
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access/${id}/reject`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return { error: 'Failed to reject' }
  return { ok: true }
}
```

- [ ] **Step 2: Write app/admin/early-access/page.tsx**

```tsx
// app/admin/early-access/page.tsx
'use client'

import { useEffect, useState } from 'react'
import {
  listEarlyAccessRequests,
  approveEarlyAccessRequest,
  rejectEarlyAccessRequest,
  type EarlyAccessRequest,
} from '@/lib/early-access-api'

const STATUS_COLORS = {
  pending: 'text-amber bg-yellowbg',
  approved: 'text-green bg-green-bg',
  rejected: 'text-muted bg-page',
}

export default function AdminEarlyAccessPage() {
  const [requests, setRequests] = useState<EarlyAccessRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [working, setWorking] = useState<string | null>(null)

  async function load() {
    setLoading(true)
    const data = await listEarlyAccessRequests()
    setRequests(data)
    setLoading(false)
  }

  useEffect(() => { load() }, [])

  async function handleApprove(id: string) {
    setWorking(id)
    await approveEarlyAccessRequest(id)
    await load()
    setWorking(null)
  }

  async function handleReject(id: string) {
    setWorking(id)
    await rejectEarlyAccessRequest(id)
    await load()
    setWorking(null)
  }

  if (loading) {
    return (
      <div className="p-8 text-sm text-muted">Loading…</div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-10">
      <h1 className="font-serif text-2xl font-semibold text-ink mb-6">
        Early access requests
      </h1>

      {requests.length === 0 ? (
        <p className="text-sm text-muted">No requests yet.</p>
      ) : (
        <div className="space-y-3">
          {requests.map((req) => (
            <div
              key={req.id}
              className="bg-surface border border-border rounded-lg px-5 py-4 flex items-start justify-between gap-4"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-semibold text-ink">{req.full_name}</span>
                  <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_COLORS[req.status]}`}>
                    {req.status}
                  </span>
                </div>
                <p className="text-sm text-muted truncate">{req.email}</p>
                <p className="text-xs text-muted-2 mt-1">
                  {req.scheme_name} · {req.unit_count} units ·{' '}
                  {new Date(req.created_at).toLocaleDateString('en-ZA', {
                    day: 'numeric', month: 'short', year: 'numeric',
                  })}
                </p>
              </div>

              {req.status === 'pending' && (
                <div className="flex items-center gap-2 flex-shrink-0">
                  <button
                    onClick={() => handleReject(req.id)}
                    disabled={working === req.id}
                    className="px-3 py-1.5 text-xs font-medium text-muted border border-border rounded hover:bg-page transition-colors disabled:opacity-50"
                  >
                    Reject
                  </button>
                  <button
                    onClick={() => handleApprove(req.id)}
                    disabled={working === req.id}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-accent rounded hover:opacity-90 transition-opacity disabled:opacity-50"
                  >
                    {working === req.id ? 'Approving…' : 'Approve'}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: Verify build**

```bash
npm run build 2>&1 | tail -20
```

Expected: clean

- [ ] **Step 4: Commit**

```bash
git add lib/early-access-api.ts app/admin/early-access/
git commit -m "feat(admin): add early access review page"
```

---

## Spec Coverage Check

- Submit early access request (public form) → Task 9
- Store requests in DB → Tasks 1, 2, 4
- Admin views requests → Tasks 5, 10
- Admin approves → creates user account, sends password-setup email → Task 4 (`Approve`)
- Admin rejects → Task 4 (`Reject`)
- Remove demo from app → Task 7
- Replace "Get started" / "Start free trial" CTAs → Task 7
- Redirect `/auth/register` to early access → Task 8
- Email notification on approval → Tasks 3, 4

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 0 | — | — |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

**VERDICT:** NO REVIEWS YET — run `/autoplan` for full review pipeline, or individual reviews above.
