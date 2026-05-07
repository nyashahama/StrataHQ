# StrataHQ Engineering Decisions

This document records the main engineering choices in StrataHQ and the tradeoffs they introduce. For the current system layout, see [architecture.md](architecture.md).

## 1. Next.js frontend with a backend proxy

StrataHQ uses a Next.js App Router frontend for the product UI and keeps core business logic in the Go backend. Browser-originated product requests do not call the Go API directly. They go through Next.js route handlers such as `app/api/proxy/[...path]/route.ts`, `app/api/session/route.ts`, `app/api/session/refresh/route.ts`, and `app/api/copilot/route.ts`.

Why this choice:

- It keeps backend URLs and token handling out of most browser code.
- It lets the frontend own cookie writes, session hydration, and CSRF checks at the browser boundary.
- It fits the repo's mix of server components, client components, and React Query data fetching.

Tradeoffs:

- There are two HTTP layers to reason about for many requests.
- Debugging auth or latency issues requires looking at both Next.js handlers and backend handlers.
- The proxy needs an explicit allowlist so it does not become a generic tunnel to the backend.

## 2. Go backend organized by Chi routes and domain packages

The backend is a Go service built around Chi routing and domain packages under `backend/internal/`. Product modules such as `auth`, `scheme`, `levy`, `maintenance`, `agm`, `documents`, `financials`, `communications`, `compliance`, `whatsapp`, `billing`, `invitation`, `contractors`, `integrations`, `ai`, and `audit` each own handlers and service logic for a bounded area.

Why this choice:

- The product is broad enough that a single package or route file would become hard to maintain.
- Go packages give clear ownership boundaries without adding a heavier framework.
- Chi keeps HTTP composition explicit, including auth, rate limiting, and audit middleware placement.

Tradeoffs:

- Cross-cutting workflows can still span multiple domain packages.
- Domain boundaries need ongoing discipline as new features are added.
- Some shared concepts, such as scheme access and audit recording, appear across modules and must stay consistent by convention.

## 3. PostgreSQL as the source of truth, with sqlc for typed access

Durable state lives in PostgreSQL. Schema changes are defined in `backend/db/migrations/`. SQL queries live in `backend/db/queries/` and generate typed Go code into `backend/db/gen/` through `sqlc`.

Why this choice:

- The product is relationship-heavy: orgs, schemes, units, memberships, levy accounts, maintenance records, AGM votes, documents, billing, and audit history all fit naturally in a relational model.
- Hand-written SQL keeps joins, constraints, and indexes visible.
- `sqlc` gives compile-time query types without introducing an ORM layer that hides SQL behavior.

Tradeoffs:

- Query updates need a generation step.
- Complex query behavior remains the team's responsibility; there is less framework abstraction to fall back on.
- Schema and query design need care up front because many product areas depend on shared tenancy and membership relationships.

## 4. Redis as a support store, not the system of record

Redis is present as supporting infrastructure rather than the primary data store. In the current codebase it backs rate limiting, health checks, and auth-related service dependencies, while PostgreSQL remains the durable record for product data.

Why this choice:

- Rate limiting and short-lived coordination data do not belong in PostgreSQL tables.
- Redis is a better fit for low-latency counters and ephemeral support concerns.
- Keeping Redis optional for support flows avoids coupling durable business records to an in-memory store.

Tradeoffs:

- The runtime stack is more complex because both Postgres and Redis must be available.
- Operators need to think about degraded behavior when Redis is unavailable.
- Some future uses of Redis, such as broader caching, must be introduced carefully to avoid stale authorization or product state.

## 5. JWT access tokens with server-side refresh tokens

StrataHQ uses short-lived JWT access tokens for API authorization and stores refresh tokens server-side in PostgreSQL. The frontend keeps access, refresh, session, and CSRF cookies, and the Next.js proxy can refresh the session when an upstream request returns `401`.

Why this choice:

- JWT access tokens keep backend authorization checks simple and stateless for the common path.
- Server-side refresh tokens allow revocation and rotation behavior that pure self-contained tokens do not provide.
- The proxy-refresh flow lets browser clients recover from expired access tokens without pushing refresh logic into every UI callsite.

Tradeoffs:

- The auth model is more complex than a single session cookie.
- Cookie handling, refresh behavior, CSRF checks, and logout flows need to stay aligned across frontend and backend code.
- Refresh tokens remain a sensitive persistence concern and deserve continued hardening.

## 6. Role-based product boundaries tied to org and scheme memberships

Authorization is not only "logged in or not." The product uses organization memberships and scheme memberships to decide what a user can view or mutate. In practice the main user roles are managing-agent admins, trustees, and residents, with additional checks at org scope, scheme scope, and in some endpoints unit scope.

Why this choice:

- Sectional-title operations are inherently multi-tenant and role-sensitive.
- A managing agent needs portfolio-wide visibility that a trustee or resident should not have.
- Many workflows, such as levy follow-up, member management, or contractor reviews, only make sense for non-resident or admin roles.

Tradeoffs:

- Authorization logic appears in many services and handlers.
- Role names alone are not enough; access checks often need org, scheme, and unit context.
- The more product areas that exist, the easier it is for a route to drift from the intended boundary if tests do not keep up.

## 7. Provider boundaries behind service clients

External systems are integrated behind backend service clients instead of from the browser. Current provider boundaries include Stripe for billing, Resend for email, Twilio WhatsApp for messaging, and an OpenAI-compatible AI provider configured in the backend.

Why this choice:

- It keeps credentials and signing logic on the server.
- It makes domain services the place where provider behavior is translated into product behavior.
- It allows no-op or test-friendly implementations in local and development flows.

Tradeoffs:

- Provider outages still affect product workflows, even if the boundaries are cleaner.
- Each provider adds configuration, webhook handling, and failure-mode complexity.
- Swapping providers is easier than if the logic lived in the frontend, but still not free because data models and workflow assumptions remain.

## 8. Separate background worker for asynchronous jobs

StrataHQ runs a separate Go worker process from `backend/cmd/worker`. Background jobs are stored in PostgreSQL and processed outside the request path by the `jobs` package. Current registered work includes collection reminder email and WhatsApp delivery plus bank statement import handling.

Why this choice:

- Request handlers stay focused on immediate API responses.
- Long-running or retryable work can be retried and observed separately.
- The same language and data access model are reused for both API and worker code.

Tradeoffs:

- Deployment now requires more than one backend process.
- Job design needs idempotency, leasing, and retry discipline.
- Operational visibility matters more because failures can happen after the original user request has already succeeded.

## 9. Security and audit posture as built-in platform concerns

The repository treats security and auditability as platform-level concerns rather than afterthoughts. Current examples include:

- backend auth middleware validating bearer tokens
- CSRF checks in the Next.js proxy/session refresh path for mutating browser requests
- rate limiting at both global and per-endpoint levels
- security headers middleware
- audit event recording middleware and resource-level audit event services
- webhook signature verification for Stripe and Twilio-facing integrations
- row-level-security hardening called out in [database.md](database.md)

Why this choice:

- The system handles identity, scheme governance, payment-related workflows, and resident communications.
- Portfolio software needs change history for operational accountability, not just debugging.
- Security controls are easier to sustain when they live in shared middleware and platform packages.

Tradeoffs:

- Security checks add friction to local development and integration work.
- Audit logging creates additional storage and review requirements.
- The posture is only as good as the consistency of its application; broad products accumulate exceptions unless they are reviewed regularly.

## 10. Open API and product API kept separate

The backend exposes product routes under `/api/v1` and integration-facing routes under `/api/open/v1`. The public OpenAPI document is separate from the authenticated product app surface.

Why this choice:

- It separates user-session workflows from API-key integration workflows.
- It allows clearer rate limiting and authorization models for machine clients.
- It reduces pressure to distort product endpoints into generic integration endpoints.

Tradeoffs:

- Similar data may need to be represented in two different API shapes.
- Integration scope management becomes its own product surface.
- Docs and tests must keep both surfaces aligned with the underlying domain model.

## Summary

The repo favors explicit boundaries over hidden framework magic: a Next.js UI layer, a Go domain backend, PostgreSQL as the durable source of truth, Redis for support concerns, typed SQL access with `sqlc`, and separate worker and provider boundaries. That choice keeps the architecture understandable and portable, but it also means the team must be disciplined about auth, audit, and cross-module consistency as the product grows.
