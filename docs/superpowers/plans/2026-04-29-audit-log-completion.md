# Audit Log Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the scheme-scoped business audit log so the remaining write-heavy product actions are recorded and visible in a scheme activity timeline for agents and trustees.

**Architecture:** Keep the existing request telemetry table (`audit_events`) separate from the business audit table (`resource_audit_events`). Reuse the current `audit.ResourceService` write path, extend the remaining domain services so they emit best-effort audit events after successful mutations, then add a simple scheme-scoped audit timeline in the frontend that reads the existing audit endpoint. Retention, archival, export, and cross-scheme analytics stay out of scope for this phase.

**Tech Stack:** Go 1.25, PostgreSQL 17, Chi, sqlc, Next.js App Router, React Query, Vitest, the existing JWT/RBAC/session helpers, and the existing backend proxy.

---

## Scope

This plan completes the current audit-log foundation that already exists in the repo:

- `backend/db/migrations/00020_resource_audit_log.sql`
- `backend/db/queries/resource_audit_events.sql`
- `backend/internal/audit/service.go`
- `backend/internal/audit/handler.go`
- `backend/internal/audit/routes.go`

The plan focuses on the remaining product write paths that are not yet consistently audited:

- scheme create/update/delete
- unit create/update
- member updates
- maintenance create/assign/resolve
- invitation create/resend/revoke
- notice creation
- WhatsApp broadcast creation and send completion
- a scheme audit timeline page in the app shell

Retention/archival is intentionally not part of this plan.

---

## File Structure

- Modify: `backend/internal/scheme/service.go`
  - Add audit recording for scheme, unit, and member mutations.
- Modify: `backend/internal/maintenance/service.go`
  - Add audit recording for maintenance request lifecycle mutations.
- Modify: `backend/internal/invitation/service.go`
  - Add audit recording for invitation lifecycle mutations.
- Modify: `backend/internal/communications/service.go`
  - Add audit recording for notice creation.
- Modify: `backend/internal/whatsapp/service.go`
  - Add audit recording for broadcast creation and send completion.
- Modify: `backend/cmd/server/main.go`
  - Wire the audit recorder into the services that need it.
- Modify: `backend/internal/scheme/service_test.go`
- Modify: `backend/internal/maintenance/service_test.go`
- Modify: `backend/internal/invitation/service_test.go`
- Modify: `backend/internal/communications/service_test.go`
- Modify: `backend/internal/whatsapp/service_test.go`
  - Add focused unit coverage for the new audit events and the best-effort failure path.
- Create: `lib/audit.ts`
  - Shared frontend audit response types.
- Create: `lib/audit-api.ts`
  - Browser API helper for the scheme audit endpoint.
- Modify: `lib/query-keys.ts`
  - Add an audit query key family for cache invalidation.
- Modify: `lib/query-keys.test.ts`
  - Assert the new query key shape.
- Modify: `components/Sidebar.tsx`
  - Add the scheme audit navigation item for agents and trustees only.
- Create: `components/Sidebar.test.tsx`
  - Assert the audit nav item is visible for agent/trustee roles and hidden for residents.
- Create: `app/app/[schemeId]/audit/page.tsx`
  - Render the scheme audit timeline page.
- Create: `app/app/[schemeId]/audit/page.test.tsx`
  - Cover loading, error, empty, and populated states.
- Modify: `backend/tests/integration/audit_test.go`
  - Expand the audit read-path integration coverage.
- Modify: `backend/internal/audit/service_test.go`
  - Add any remaining audit read-path unit coverage that the integration test does not cover cleanly.

---

### Task 1: Audit the Scheme and Maintenance Write Paths

**Files:**
- Modify: `backend/internal/scheme/service.go`
- Modify: `backend/internal/maintenance/service.go`
- Modify: `backend/internal/scheme/service_test.go`
- Modify: `backend/internal/maintenance/service_test.go`

- [ ] **Step 1: Write the failing tests**

Add a small in-memory auditor in both test files:

```go
type recordingAuditor struct {
	events []audit.ResourceEvent
	err    error
}

func (r *recordingAuditor) RecordResourceEvent(_ context.Context, event audit.ResourceEvent) error {
	r.events = append(r.events, event)
	return r.err
}
```

Add representative tests that assert these actions are emitted:

```go
func TestCreateSchemeRecordsAuditEvent(t *testing.T)
func TestUpdateSchemeRecordsAuditEvent(t *testing.T)
func TestDeleteSchemeRecordsAuditEvent(t *testing.T)
func TestCreateUnitRecordsAuditEvent(t *testing.T)
func TestUpdateUnitRecordsAuditEvent(t *testing.T)
func TestUpdateMemberRecordsAuditEvent(t *testing.T)
func TestCreateMaintenanceRequestRecordsAuditEvent(t *testing.T)
func TestAssignMaintenanceRequestRecordsAuditEvent(t *testing.T)
func TestResolveMaintenanceRequestRecordsAuditEvent(t *testing.T)
```

Each test should assert:

- `ResourceType` matches the domain object (`scheme`, `unit`, `member`, `maintenance_request`)
- `Action` matches the dotted event name
- `BeforeState` and `AfterState` contain public state only
- the mutation still succeeds when the auditor returns an error

Example expectations:

```go
if got := auditor.events[0].Action; got != "scheme.created" {
	t.Fatalf("action = %q, want scheme.created", got)
}
if got := auditor.events[0].ResourceType; got != "scheme" {
	t.Fatalf("resource_type = %q, want scheme", got)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
cd backend && go test ./internal/scheme ./internal/maintenance -run 'Test.*Audit' -count=1
```

Expected: fail because the services still do not emit the new audit events.

- [ ] **Step 3: Add the audit plumbing and event builders**

In each service, add the same pattern used by documents, AGM, and levy:

```go
type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	db      *database.Pool
	auditor resourceAuditor
}

func NewService(db *database.Pool) *Service {
	return NewServiceWithAudit(db, nil)
}

func NewServiceWithAudit(db *database.Pool, auditor resourceAuditor) *Service {
	return &Service{db: db, auditor: auditor}
}
```

Emit audit rows after each successful mutation:

- `scheme.created` after a scheme is inserted
- `scheme.updated` after the scheme row is updated
- `scheme.deleted` before deleting, using the last public state and the counts that are about to disappear
- `unit.created` and `unit.updated` after the unit row is saved
- `member.updated` after the membership row is updated
- `maintenance.request_created`, `maintenance.request_assigned`, and `maintenance.request_resolved` after the maintenance row transition succeeds

Keep the payloads safe and concise:

- do not include tokens, secrets, or provider credentials
- do not include internal IDs that are not already visible in the API response
- use `before_state` and `after_state` only for the public fields that explain the change

The audit call must remain best-effort:

```go
if s.auditor != nil {
	_ = s.auditor.RecordResourceEvent(ctx, audit.ResourceEvent{...})
}
```

- [ ] **Step 4: Run the targeted tests and lint**

Run:

```bash
cd backend && go test ./internal/scheme ./internal/maintenance -count=1
cd backend && make lint
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/scheme/service.go backend/internal/maintenance/service.go backend/internal/scheme/service_test.go backend/internal/maintenance/service_test.go
git commit -m "feat: audit scheme and maintenance mutations"
```

---

### Task 2: Audit Invitations, Notices, and WhatsApp Broadcasts

**Files:**
- Modify: `backend/internal/invitation/service.go`
- Modify: `backend/internal/communications/service.go`
- Modify: `backend/internal/whatsapp/service.go`
- Modify: `backend/internal/invitation/service_test.go`
- Modify: `backend/internal/communications/service_test.go`
- Modify: `backend/internal/whatsapp/service_test.go`

- [ ] **Step 1: Write the failing tests**

Add an in-memory auditor and representative tests for the remaining user-facing writes:

```go
func TestCreateInvitationRecordsAuditEvent(t *testing.T)
func TestResendInvitationRecordsAuditEvent(t *testing.T)
func TestRevokeInvitationRecordsAuditEvent(t *testing.T)
func TestCreateNoticeRecordsAuditEvent(t *testing.T)
func TestCreateBroadcastRecordsAuditEvent(t *testing.T)
```

The invitation tests must assert that the token is not written into the audit payload.

The notice test should record the current notice creation mutation as:

- `ResourceType: "notice"`
- `Action: "notice.created"`

There is no explicit publish mutation in the current service, so do not invent a fake publish step for this plan.

The WhatsApp test should assert both:

- a broadcast row is audited as `whatsapp.broadcast_created`
- the send completion is audited as `whatsapp.broadcast_sent` with recipient counts in metadata

The best-effort rule still applies:

- a sender failure logs the provider error
- the broadcast mutation still completes
- the audit row is still recorded

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
cd backend && go test ./internal/invitation ./internal/communications ./internal/whatsapp -run 'Test.*Audit' -count=1
```

Expected: fail because the new audit events are not emitted yet.

- [ ] **Step 3: Add the audit plumbing and event builders**

Mirror the same constructor pattern used in Task 1:

```go
type resourceAuditor interface {
	RecordResourceEvent(ctx context.Context, event audit.ResourceEvent) error
}

type Service struct {
	db      *database.Pool
	auditor resourceAuditor
}

func NewService(db *database.Pool, ...) *Service {
	return &Service{db: db, ...}
}

func NewServiceWithAudit(db *database.Pool, ..., auditor resourceAuditor) *Service {
	return &Service{db: db, ..., auditor: auditor}
}
```

Add event builders that snapshot only public state:

- invitation: email, role, scheme_id, unit_id, status, expires_at
- notice: title, body, type, sent_at
- WhatsApp: broadcast type, message, recipient_count, sent_at, send result summary

Use these event names:

- `invitation.created`
- `invitation.resent`
- `invitation.revoked`
- `notice.created`
- `whatsapp.broadcast_created`
- `whatsapp.broadcast_sent`

Do not write invite tokens, webhooks, or phone numbers into the audit snapshots.

- [ ] **Step 4: Run the targeted tests and lint**

Run:

```bash
cd backend && go test ./internal/invitation ./internal/communications ./internal/whatsapp -count=1
cd backend && make lint
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/invitation/service.go backend/internal/communications/service.go backend/internal/whatsapp/service.go backend/internal/invitation/service_test.go backend/internal/communications/service_test.go backend/internal/whatsapp/service_test.go
git commit -m "feat: audit invitations notices and whatsapp"
```

---

### Task 3: Wire the Audit Recorder Through the Server Entry Point

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Write the failing build check**

Once Tasks 1 and 2 add the new constructors, `main.go` will still call the old constructors. First confirm the compile failure by running:

```bash
cd backend && go build ./cmd/server ./cmd/worker
```

Expected: fail with missing constructor arguments or missing `WithAudit` calls.

- [ ] **Step 2: Update the wiring**

Create the `resourceAuditService` once and inject it into every audited domain service:

```go
resourceAuditService := audit.NewResourceService(db.Q)

schemeService := scheme.NewServiceWithAudit(db, resourceAuditService)
maintenanceService := maintenance.NewServiceWithAudit(db, resourceAuditService)
invitationService := invitation.NewServiceWithAudit(db, emailClient, cfg.AppBaseURL, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry, resourceAuditService)
communicationsService := communications.NewServiceWithAudit(db, resourceAuditService)
whatsAppService := whatsapp.NewServiceWithAudit(db, sender, logger, resourceAuditService)
```

Keep the existing request-audit service, handler wiring, and router mounting unchanged.

- [ ] **Step 3: Run the build and tests**

Run:

```bash
cd backend && go build ./cmd/server ./cmd/worker
cd backend && go test ./internal/... -count=1
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "fix: wire resource audit recorder into services"
```

---

### Task 4: Add the Scheme Audit Timeline UI

**Files:**
- Create: `lib/audit.ts`
- Create: `lib/audit-api.ts`
- Modify: `lib/query-keys.ts`
- Modify: `lib/query-keys.test.ts`
- Modify: `components/Sidebar.tsx`
- Create: `components/Sidebar.test.tsx`
- Create: `app/app/[schemeId]/audit/page.tsx`
- Create: `app/app/[schemeId]/audit/page.test.tsx`

- [ ] **Step 1: Write the failing frontend tests**

Add a query-key test:

```ts
expect(schemeKeys.audit("scheme-1")).toEqual([
  "scheme",
  "scheme-1",
  "audit",
]);
```

Add a sidebar test that proves:

- `agent-scheme` and `trustee` see `Audit log`
- `resident` does not see `Audit log`

Add a page test modeled after the existing scheme page tests:

```ts
const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ user: { role: "trustee" } })),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));
```

The page test should cover:

- loading state
- retry state on query failure
- empty state with no events
- a populated event row with expandable details

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
npm test -- lib/query-keys.test.ts components/Sidebar.test.tsx app/app/[schemeId]/audit/page.test.tsx
```

Expected: fail because the audit client API, query key, sidebar item, and page do not exist yet.

- [ ] **Step 3: Implement the audit client types, API, and page**

Add shared types in `lib/audit.ts`:

```ts
export interface AuditEventInfo {
  id: string;
  scheme_id: string;
  org_id: string;
  actor_user_id: string | null;
  actor_role: string;
  resource_type: string;
  resource_id: string | null;
  action: string;
  before_state: Record<string, unknown> | null;
  after_state: Record<string, unknown> | null;
  metadata: Record<string, unknown> | null;
  occurred_at: string;
}

export interface AuditEventsResponse {
  events: AuditEventInfo[];
  total: number;
  limit: number;
}
```

Add the browser helper in `lib/audit-api.ts`:

```ts
export async function getSchemeAuditEvents(
  schemeId: string,
  limit = 50,
): Promise<AuditEventsResponse>
```

Update `schemeKeys` in `lib/query-keys.ts` with:

```ts
audit: (schemeId: string) => ["scheme", schemeId, "audit"] as const,
```

Build `app/app/[schemeId]/audit/page.tsx` as a client page that:

- uses `useParams()` to read `schemeId`
- uses `useAuth()` to block residents
- uses `useAuthenticatedQuery` with `schemeKeys.audit(schemeId)`
- renders `RetryState` on error
- renders an empty state when `events.length === 0`
- renders each event in a compact timeline row with `<details>` for `before_state`, `after_state`, and `metadata`

Render action labels and actor labels directly from the API response so the page does not need a second mapping layer.

Update the sidebar to add `Audit log` under the scheme nav for agent and trustee roles only.

- [ ] **Step 4: Run the frontend tests and type check**

Run:

```bash
npm test
npm run lint
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add lib/audit.ts lib/audit-api.ts lib/query-keys.ts lib/query-keys.test.ts components/Sidebar.tsx components/Sidebar.test.tsx app/app/[schemeId]/audit/page.tsx app/app/[schemeId]/audit/page.test.tsx
git commit -m "feat: add scheme audit timeline ui"
```

---

### Task 5: Expand Audit Read-Path Coverage and Verify End-to-End Behavior

**Files:**
- Modify: `backend/tests/integration/audit_test.go`
- Modify: `backend/internal/audit/service_test.go`

- [ ] **Step 1: Write the failing integration and service tests**

Expand the audit integration test so it proves all three of these behaviors:

```go
func TestAudit_ListSchemeEvents_ReturnsNewestFirst(t *testing.T)
func TestAudit_ListSchemeEvents_RejectsResidents(t *testing.T)
func TestAudit_ListSchemeEvents_UsesDefaultLimit(t *testing.T)
```

Seed a few `resource_audit_events` rows through `audit.NewResourceService(testPool.Q)` so the endpoint sees real database rows and real JSON snapshots.

Add one unit-level read-path assertion in `backend/internal/audit/service_test.go` if needed to cover the default limit and forbidden resident path without depending on the integration harness.

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
cd backend && go test -tags=integration ./tests/integration -run 'TestAudit' -count=1
```

Expected: fail because the expanded assertions are not in place yet.

- [ ] **Step 3: Implement the read-path coverage**

Keep the existing audit endpoint and service behavior, but strengthen the tests so they assert:

- newest events come back first
- the `limit` default is applied when the query parameter is empty or invalid
- residents receive `403`
- trustees and agents can read events for their scheme

If any event mapping helpers need to change, keep the output envelope compatible with the current frontend page.

- [ ] **Step 4: Run the full verification suite**

Run:

```bash
cd backend && go test ./internal/... -count=1
cd backend && go test -tags=integration ./tests/integration -count=1
npm test
npm run lint
cd backend && go build ./cmd/server ./cmd/worker
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add backend/tests/integration/audit_test.go backend/internal/audit/service_test.go
git commit -m "test: expand audit read-path coverage"
```

---

## Self-Review

Before handing the plan to an implementation worker:

1. Confirm each remaining write-heavy service has a task and a test file.
2. Confirm the plan does not introduce a retention/archival requirement.
3. Confirm the frontend task uses the same query-key and retry-state patterns as the rest of the app.
4. Confirm the backend audit read API remains unchanged and is only being covered more thoroughly.

