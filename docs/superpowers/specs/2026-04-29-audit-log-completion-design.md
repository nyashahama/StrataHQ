# Audit Log Completion Design

**Goal:** Complete StrataHQ's scheme-scoped business audit log so significant write actions are recorded consistently and visible in a scheme audit timeline for agents and trustees.

**Architecture:** Keep the existing request telemetry table (`audit_events`) separate from the business audit table (`resource_audit_events`). Reuse the current `audit.ResourceService` as the write sink, expand the remaining domain services to emit resource audit events after successful mutations, and add a scheme-scoped UI that reads from the existing audit endpoint. Retention/archival is intentionally out of scope for this phase.

**Tech Stack:** Go 1.24, PostgreSQL 17, sqlc, Chi, Next.js App Router, existing JWT/RBAC/session helpers, existing backend proxy and scheme shell UI.

---

## 1. Requirements and Scoping

### Functional Requirements

- Record high-value scheme write actions as immutable business audit rows with actor, role, resource type, resource ID, action, before-state, after-state, and metadata.
- Preserve the current scheme-scoped audit read API and use it to render a readable audit timeline.
- Cover the remaining major write paths that are not yet emitting resource audit events:
  - scheme create/update/delete
  - unit create/update
  - member updates
  - maintenance request create/assign/resolve
  - invitation create/resend/revoke
  - notice creation/publishing
  - WhatsApp broadcast creation/sending
- Keep the already-audited document, levy, AGM, proxy, vote, and collection flows intact.
- Show only agents/managing staff and trustees in the audit UI; residents must not see the page or the data.
- Present event details in a way that makes it obvious what changed without exposing secrets such as invitation tokens or internal storage credentials.

### Non-Functional Requirements

- Audit recording must remain best-effort and must not block the primary business mutation in the first release.
- The read path must stay fast enough for a recent-event timeline page, with a default page size around 50 events.
- Audit rows must be append-only and scheme-scoped so they can support compliance and support debugging.
- Audit data must be easy to explain and inspect during STSMA-related reviews and customer support escalations.
- The implementation should avoid broad schema churn because the underlying audit table already exists and is working in production-like tests.

### Capacity Estimate

- Early production should expect hundreds of audit events per scheme over time, with higher churn on levy, AGM, maintenance, and document actions.
- A scheme timeline page will usually inspect only the most recent 25 to 100 events.
- The write overhead should be small compared with the primary mutation latency because the audit payload is JSON-based and the table is append-only.

### Scope Boundary

- In scope:
  - missing audit hooks in the remaining scheme-domain services
  - a scheme audit timeline UI
  - integration tests for representative audited actions and access control
  - event naming conventions and snapshot conventions for new hooks
- Out of scope:
  - retention or archival jobs
  - exporting audit logs to PDF or CSV
  - cross-scheme admin analytics dashboards
  - a second audit storage system or event-sourcing rewrite

## 2. Current State

StrataHQ already has the core audit foundation in place:

- `backend/db/migrations/00020_resource_audit_log.sql` created the `resource_audit_events` table.
- `backend/db/queries/resource_audit_events.sql` provides create/list/count queries.
- `backend/internal/audit/service.go` exposes `ResourceService` and `ListSchemeEvents`.
- `backend/internal/audit/handler.go` and `backend/internal/audit/routes.go` expose a protected scheme audit read endpoint.
- Document, levy, and AGM services already emit resource audit events for major actions.

The remaining gap is completeness:

- Several write-heavy domains still do not emit resource audit rows.
- There is no scheme audit page in the frontend.
- There is no consistent event coverage checklist that tells future contributors which changes must be audited.

## 3. Design Options

### Option A: Backend-only completion

Record the missing audit events in the domain services and stop there.

Pros:
- Smallest code change.
- Lowest UI and routing risk.

Cons:
- The audit data exists but is hard to discover.
- Support and compliance users still need to query the database or backend directly.

### Option B: Backend completion plus minimal scheme audit page

Record the missing events and add a scheme-scoped audit timeline page with basic details and no complex filters.

Pros:
- Delivers the actual customer-facing value of the feature.
- Keeps the page simple enough to ship in one plan.
- Matches the existing scheme shell and backend proxy patterns.

Cons:
- No advanced search, export, or retention in this phase.
- Large histories still require loading a recent slice rather than a full data explorer.

### Option C: Backend completion plus full audit explorer

Add backend hooks, advanced filtering, full pagination, export, and retention controls in the same plan.

Pros:
- Very complete from day one.

Cons:
- Too broad for one implementation cycle.
- Adds avoidable complexity before the audit feature has proved useful.

### Chosen Approach

Use **Option B**.

It is the smallest scope that produces a real audit product: the important writes are captured, and users can inspect them in the app without needing database access. Retention stays as a separate follow-up.

## 4. Data Model

No new core table is required for this phase.

The existing `resource_audit_events` table remains the source of truth:

- `scheme_id` scopes the event to one scheme.
- `org_id` ties the event back to the managing-agent organization.
- `actor_user_id` and `actor_role` identify who caused the change.
- `resource_type`, `resource_id`, and `action` define what changed.
- `before_state`, `after_state`, and `metadata` store JSON snapshots.
- `occurred_at` preserves append order.

### Event Taxonomy

Keep the current dotted naming style and use stable names for all new events.

Existing names that stay unchanged:

- `document.uploaded`
- `document.deleted`
- `levy.period.created`
- `levy.reconciled`
- `agm.proxy_assigned`
- `agm.vote_cast`

New names to standardize in the missing hooks:

- `scheme.created`
- `scheme.updated`
- `scheme.deleted`
- `unit.created`
- `unit.updated`
- `member.updated`
- `maintenance.request_created`
- `maintenance.request_assigned`
- `maintenance.request_resolved`
- `invitation.created`
- `invitation.resent`
- `invitation.revoked`
- `notice.created`
- `notice.published`
- `whatsapp.broadcast_created`
- `whatsapp.broadcast_sent`

### Snapshot Rules

- `before_state` should contain the pre-change public state when available.
- `after_state` should contain the post-change public state when available.
- `metadata` should carry contextual data that should not be treated as the core object snapshot.
- Tokens, secrets, upload credentials, and provider-side signed URLs must never be written into audit snapshots.

## 5. API Design

### Read API

Keep the current scheme audit endpoint and its role checks.

- `GET /audit/schemes/{schemeId}/events`

Behavior:

- Admin/managing-agent users can read events for schemes in their organization.
- Trustees can read events for schemes they are members of as trustees.
- Residents cannot read the audit timeline.
- The default page size stays small enough for a timeline view.

### Write API

No new public write endpoint is needed.

Audit rows are created only from server-side domain services after a successful mutation. The frontend never writes audit events directly.

### UI Contract

The scheme audit page will consume the existing JSON envelope returned by the read endpoint and render:

- timestamp
- actor
- action
- resource type
- resource summary
- expandable before/after details

## 6. High-Level Architecture

### Backend Flow

1. A domain service performs the business mutation.
2. The service builds a resource audit event from the inputs and the saved row.
3. The service calls `audit.ResourceRecorder.RecordResourceEvent(...)`.
4. The audit service writes a row to `resource_audit_events`.
5. The scheme audit page reads the latest events through the existing audit handler.

### Service Boundaries

Keep the audit recording responsibility at the service layer rather than the handler layer.

- `backend/internal/scheme/service.go` owns scheme, unit, and member changes.
- `backend/internal/maintenance/service.go` owns maintenance request changes.
- `backend/internal/invitation/service.go` owns invitation lifecycle changes.
- `backend/internal/communications/service.go` owns notice creation/publishing.
- `backend/internal/whatsapp/service.go` owns WhatsApp broadcast actions.

The existing document, levy, and AGM services remain the examples to follow for event structure and helper naming.

### Frontend Flow

1. The scheme shell exposes an `Audit log` nav item to agent and trustee roles.
2. The audit page fetches the latest audit events for the active scheme.
3. The page renders a compact event list with expandable detail blocks.
4. The page shows an empty state when there are no events yet.

## 7. Frontend Design

### Route

Add a scheme-scoped page at:

- `app/app/[schemeId]/audit/page.tsx`

### Navigation

Add an `Audit log` item to the scheme sidebar for:

- `agent-scheme`
- `trustee`

Do not add it to the resident sidebar.

### Layout

The page should follow the existing scheme page patterns:

- short breadcrumb or section label at the top
- a title that explains this is the scheme activity trail
- a concise explanatory subtitle
- a recent-events list in the same visual language as the other scheme pages

### Event Display

Each row should show:

- action label
- resource type
- actor name or role
- timestamp
- a short summary from the snapshots

When a row is expanded, show:

- before state
- after state
- metadata

If a field is missing, show a compact placeholder such as `-` rather than inventing values.

### Empty and Error States

- Empty state: “No audit events yet.”
- Access denied: let the existing auth/session layer handle it.
- Load failure: show the standard retry state used by other scheme pages.

## 8. Observability and Security

### Observability

- Audit recording failures should be visible in application logs, but they must not interrupt the main user flow in this release.
- The audit page should be simple enough that a failed request is obvious and retryable.

### Security

- Keep scheme access checks in the service layer.
- Do not expose audit data to residents.
- Do not include secrets or provider tokens in snapshots.
- Keep the request audit table separate from the business audit table so operational telemetry and compliance data do not get mixed.

## 9. Testing Strategy

### Unit Tests

- Verify the audit event builders produce the expected `resource_type`, `action`, and JSON snapshots.
- Verify nil/invalid recorder inputs fail safely.
- Verify the service rejects events with missing scheme, org, resource type, or action fields.

### Integration Tests

- Create a scheme and exercise a representative audited action from each remaining domain.
- Verify the corresponding `resource_audit_events` rows are written.
- Verify the audit page or audit handler rejects residents and allows trustees/agents.
- Verify the existing audited actions still emit rows after the new work lands.

### Manual Smoke Tests

- Create or update a scheme, maintenance request, invitation, and notice in a local environment.
- Confirm the audit timeline page updates without requiring database inspection.

## 10. Implementation Notes

- Preserve the current `audit_events` request telemetry table unchanged.
- Preserve the current `resource_audit_events` schema unless a specific bug forces a change.
- Keep event builder helpers close to the domain they describe so snapshots stay easy to reason about.
- Prefer small, explicit payloads over generic `map[string]any` blobs in the services.
- Keep the audit page lean; this is an operational timeline, not a full analytics product.

## 11. Open Questions Resolved

- Retention/cleanup is deferred to a later plan.
- Advanced search and export are deferred to a later plan.
- No new public audit write endpoint is needed.

