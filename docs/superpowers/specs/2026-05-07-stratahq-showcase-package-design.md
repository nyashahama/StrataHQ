# StrataHQ Showcase Package Design

## Purpose

Create a complete StrataHQ evaluation package for GitHub visitors, technical reviewers, and portfolio readers. The work should make the project easy to understand, run, inspect, and present without adding new product surface area to the application.

The package will include:

- A clean live demo section with role-based demo credentials
- Architecture diagram and explanation
- API documentation
- Screenshots of core workflows
- Database schema overview
- A "Run locally in 5 minutes" path
- A `v0.1.0-alpha` release tag plan
- A portable portfolio case study
- A short demo video script/storyboard
- An "Engineering decisions" section

## Scope Decision

Use **Repo plus portfolio case study** as the scope.

This means the repository becomes the source of truth for the public showcase:

- `README.md` acts as the front door for evaluators.
- `docs/` contains deeper technical docs and portfolio-ready narrative material.
- Screenshot assets are committed to the repo when they can be generated from a seeded local app.
- The portfolio case study is created inside this repo as a portable Markdown artifact, not published directly into a separate portfolio repository.

Out of scope for this pass:

- Building a dedicated in-app showcase page
- Creating a new marketing site
- Publishing to an external portfolio repository
- Recording and hosting the final video file
- Adding new product features to support the demo

## Live Demo

The README should document the current live frontend:

- Demo URL: `https://strata-hq-blue.vercel.app`
- Repository URL: `https://github.com/nyashahama/StrataHQ`

The demo will expose three role-based logins using seeded fake data only:

| Role | Email | Password |
| --- | --- | --- |
| Managing agent | `agent@demo.stratahq.test` | `StrataDemo!2026` |
| Trustee | `trustee@demo.stratahq.test` | `StrataDemo!2026` |
| Resident | `resident@demo.stratahq.test` | `StrataDemo!2026` |

The docs must clearly state that these accounts are for a seeded demo environment and must never contain real resident, owner, scheme, financial, or contact data.

The implementation plan should include a production-demo checklist:

1. Set `SEED_DEMO_PASSWORD=StrataDemo!2026` in the backend demo environment.
2. Run the existing backend seed flow against the demo database.
3. Confirm all three users can log in.
4. Confirm the demo data is fake and representative.
5. Add a periodic reset/rotation note so public demo state can be refreshed.

## README Structure

Update the README so a reviewer can evaluate the project quickly.

Recommended order:

1. Project summary and product scope
2. Live demo URL and credentials
3. Screenshot gallery
4. Core workflows
5. Architecture overview with link to detailed docs
6. API documentation link
7. Database schema overview link
8. Run locally in 5 minutes
9. Tech stack
10. Engineering decisions
11. Release status, including `v0.1.0-alpha`
12. Testing and verification

The README should stay concise. Deeper detail belongs in dedicated docs files.

## Documentation Files

Create these docs:

| File | Purpose |
| --- | --- |
| `docs/demo.md` | Demo URL, credentials, role walkthroughs, reset checklist, video script/storyboard |
| `docs/architecture.md` | System architecture diagram, request flow, deployment shape, auth/session flow |
| `docs/api.md` | API groups and endpoint table generated from existing Go route definitions |
| `docs/database.md` | Database schema overview grouped by domain from migrations |
| `docs/engineering-decisions.md` | Concise explanation of key implementation decisions and tradeoffs |
| `docs/case-study.md` | Portable portfolio case study for StrataHQ |

These docs should be useful independently, but the README should link to all of them.

## Architecture Diagram

Use Mermaid in Markdown so the diagram is version-controlled and viewable on GitHub.

The architecture doc should show:

- Browser and Next.js App Router frontend
- Next.js API proxy/session routes
- Go backend API
- Go background worker
- PostgreSQL
- Redis
- External services: Stripe, Resend, Twilio WhatsApp, AI provider
- Vercel-style frontend deployment and containerized backend deployment

The diagram should reflect the current repo shape:

- Frontend: `app/`, `components/`, `lib/`
- Backend: `backend/cmd/server`, `backend/cmd/worker`, `backend/internal/*`
- Data: `backend/db/migrations`, `backend/db/queries`, generated sqlc code

## API Documentation

The API doc should be grounded in the current route files under `backend/internal/*/routes.go`.

Group endpoints by domain:

- Health and platform
- Auth and account
- Schemes, units, and members
- Invitations
- Levies, reconciliation, bank statement imports, and collection follow-up
- Maintenance
- AGM
- Documents
- Financials
- Communications
- Compliance
- WhatsApp
- Contractors
- Billing
- Early access
- AI copilot
- Audit
- Integrations and open API

Each endpoint row should include:

- Method
- Path
- Auth requirement
- Short purpose

The doc should mention the response envelope:

```json
{ "data": {}, "meta": {} }
```

and error envelope:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "..." } }
```

## Database Schema Overview

The database overview should summarize tables by product domain rather than listing every column in every migration.

Required groups:

- Identity and organizations: `users`, `orgs`, `org_memberships`, `refresh_tokens`
- Schemes and membership: `schemes`, `units`, `scheme_memberships`
- Levy and collections: `levy_periods`, `levy_accounts`, `levy_payments`, collection event/reminder tables, bank statement import tables
- Maintenance
- AGM
- Documents
- Financials
- Communications
- Compliance
- WhatsApp maintenance inbox
- Billing
- Integrations and open API clients
- Audit/resource audit events
- Background jobs

The doc should call out important implementation choices:

- UUID primary keys
- Foreign keys with cascade behavior for owned domain records and `SET NULL` for optional links such as unit-bound memberships
- `updated_at` trigger function
- sqlc-generated typed query layer
- Goose migrations as the source of truth

## Screenshots

Generate actual screenshots from the local seeded app when the app runs cleanly.

Target screenshots:

1. Landing page
2. Login page with demo credential context
3. Managing-agent portfolio dashboard
4. Scheme overview
5. Levy dashboard and bank reconciliation/import workflow
6. Maintenance dashboard
7. AGM workflow
8. Documents or compliance workflow
9. Audit or WhatsApp maintenance inbox, if seeded data makes it meaningful

Store screenshots in:

```text
docs/assets/screenshots/
```

Use descriptive, stable filenames such as:

```text
docs/assets/screenshots/agent-portfolio-dashboard.png
docs/assets/screenshots/scheme-overview.png
docs/assets/screenshots/levy-reconciliation.png
```

If the local seeded app cannot produce meaningful authenticated screenshots in the current environment, the implementation should still create `docs/demo.md` with exact capture instructions and avoid committing broken placeholder images.

## Run Locally In 5 Minutes

The local setup should be a short, copy-pasteable path from a fresh clone:

1. Install frontend dependencies with `npm ci`.
2. Create `.env.local`.
3. Start backend dependencies with Docker Compose.
4. Configure `backend/.env`.
5. Run migrations and seed demo data.
6. Start the Go backend.
7. Start the Next.js frontend.
8. Log in with the three demo users.

The docs should be honest about prerequisites:

- Node.js 22+
- npm
- Go 1.25.9+
- Docker and Docker Compose
- `goose`
- `sqlc`

## Release Tag

The docs should identify the current showcase milestone as:

```text
v0.1.0-alpha
```

The implementation should not create the tag until after the documentation and screenshots are committed and verified. The release checklist should include:

1. Run frontend checks.
2. Run backend compile/unit checks where relevant.
3. Confirm demo URL and credentials.
4. Confirm README links.
5. Create an annotated tag:

```bash
git tag -a v0.1.0-alpha -m "StrataHQ v0.1.0-alpha showcase release"
git push origin v0.1.0-alpha
```

## Portfolio Case Study

Create `docs/case-study.md` as a polished, portable case study.

Recommended sections:

1. Title and one-line summary
2. Problem
3. Users and workflows
4. What was built
5. Architecture
6. Engineering decisions
7. Security and reliability considerations
8. Demo walkthrough
9. Results and current status
10. What I would improve next

The case study should be written for a technical hiring manager or senior engineer reviewing the project. It should avoid hype and focus on concrete engineering choices.

## Demo Video

Do not commit a video file in this pass.

Add a short script/storyboard to `docs/demo.md` covering a 90-120 second video:

1. Login with managing-agent demo user
2. Show portfolio dashboard
3. Open a scheme
4. Show levy reconciliation/import workflow
5. Show maintenance or AGM workflow
6. Show docs/API/schema/architecture briefly
7. Close with local run path and engineering decisions

Leave a clear slot for a future hosted video link without implying the video is already recorded:

```markdown
Demo video: not yet recorded
```

## Engineering Decisions

The engineering decisions doc should explain decisions that make the project credible:

- Next.js frontend with server routes and a backend proxy
- Go backend with Chi, pgx, sqlc, and explicit domain packages
- PostgreSQL as the source of truth and Redis for session/cache/rate-limit needs
- JWT access/refresh token model with server-side refresh token storage
- Goose migrations and sqlc query generation
- Role-based access across managing agents, trustees, residents, and owners
- Security headers, CSRF handling, CORS, rate limiting, and audit logging
- Background worker separation for asynchronous work
- External provider boundaries for Stripe, Resend, Twilio, and AI provider integration

Each decision should include the tradeoff, not just the chosen technology.

## Testing And Verification

Implementation verification should include:

```bash
npm run lint
npm run typecheck
npm test
```

For backend documentation accuracy, also run a compile/no-op test when backend files or generated docs depend on Go route references:

```bash
cd backend
go test ./... -run TestNonExistent -count=1
```

Screenshot generation should be verified by opening the image files and confirming they are not blank, cropped incorrectly, or showing private data.

## Risks

- Live demo credentials can become a security issue if connected to real data. Mitigation: use seeded fake data only and document reset/rotation.
- Screenshot generation may fail if local auth/session flow depends on services not available in the environment. Mitigation: create capture instructions and only commit real screenshots that are verified.
- API docs can drift if manually maintained. Mitigation: derive the first version from current `routes.go` files and keep docs grouped by domain.
- README can become too long. Mitigation: keep README as overview and link out to deeper docs.

## Acceptance Criteria

The work is complete when:

- README contains live demo, credentials, screenshots, local setup, doc links, release status, and engineering decisions summary.
- `docs/demo.md`, `docs/architecture.md`, `docs/api.md`, `docs/database.md`, `docs/engineering-decisions.md`, and `docs/case-study.md` exist and contain concrete project-specific content.
- Screenshots are either committed under `docs/assets/screenshots/` or `docs/demo.md` explains exactly how to capture them once the local seeded app is available.
- Release tagging instructions for `v0.1.0-alpha` are documented.
- Verification commands pass, or any unavailable checks are documented clearly.
