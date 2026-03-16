# TODOS

Items deferred from the V1 roadmap review. Each item has enough context to be picked up cold.

---

## V1 / Testing Infrastructure

### TypeScript Factory Functions for Test Data

**What:** TypeScript factory functions (like FactoryBot/factory_boy) that create DB rows for tests and return typed objects. Replaces/supplements `supabase/seed.sql` + raw SQL in `beforeEach`.

**Why:** As the codebase grows beyond 3 modules, seeding test data via SQL becomes hard to maintain. A factory for `createScheme()`, `createUnit()`, `createLevyAccount()` makes test setup composable and readable.

**Pros:** Tests become self-documenting. Complex relationship scenarios (scheme → 10 units → levy period → overdue accounts → maintenance requests) are expressed in TypeScript, not SQL. Easy to add to any test without touching seed.sql.

**Cons:** More code to maintain. Must stay in sync with schema changes. Overkill while the codebase has fewer than 3 modules with test coverage.

**Context:** Seed.sql + beforeEach is the right approach for PR 2. Revisit when the Maintenance or AGM module tests are being written and you're writing the same SQL insert patterns repeatedly. The factories should live in `tests/factories/` and use the Supabase admin client (bypasses RLS, for test setup only).

**Effort:** M
**Priority:** P3
**Depends on:** At least 3 modules with integration test coverage

---

## V2 — High Priority

### EFT Auto-Reconciliation

**What:** Upload a SA bank statement CSV and StrataHQ auto-matches EFT payments to levy accounts by reference number.

**Why:** Managing agents currently reconcile 100+ EFT payments manually in Excel every month. This is the single highest-friction workflow in scheme management and a strong reason for agents to switch to StrataHQ.

**Pros:** Eliminates the most painful manual task for managing agents. Strong differentiation — no other SA tool does this well.

**Cons:** SA banks (FNB, ABSA, Nedbank, Standard Bank) each export statements in different CSV formats. Requires building and maintaining 4+ parsers.

**Context:** The data model is ready — `levy_payments.reference` and `levy_payments.bank_ref` columns should be added in V1 even if auto-matching is deferred. The matching algorithm is: normalize EFT reference → fuzzy match against unit number patterns → flag ambiguous matches for manual review. Start with FNB (most common among managing agents).

**Effort:** L
**Priority:** P1
**Depends on:** Levy & Payments module (V1), `levy_payments.bank_ref` column added in V1

---

### Audit Log

**What:** Every significant action in the system (levy created, vote cast, document uploaded, user invited, payment reconciled) is recorded with actor + before/after state.

**Why:** STSMA compliance audits require a paper trail. Enterprise managing agents will not sign contracts without audit logging. This also dramatically reduces support burden ("why did this levy amount change?").

**Pros:** Required for enterprise sales. Makes debugging production issues trivial. Covers STSMA compliance requirements.

**Cons:** Adds a write to every Server Action. Needs careful schema design to handle JSON snapshots at scale.

**Context:** Add `audit_log(id, scheme_id, resource_type, resource_id, action, actor_id, before_state jsonb, after_state jsonb, created_at)` table in V1 (zero cost), add inserts to Server Actions in V2, build the UI in V2. The table should have a Supabase RLS policy: managing agents and trustees can read their scheme's log, residents cannot.

**Effort:** M
**Priority:** P1
**Depends on:** All V1 modules complete (audit log inserts go into every Server Action)

---

## V2 — Medium Priority

### WhatsApp Maintenance Inbox

**What:** Residents send a WhatsApp message (photo + description) to a dedicated number, and StrataHQ auto-creates a maintenance request ticket.

**Why:** The landing page explicitly calls out "Maintenance disappears into WhatsApp" as the #1 pain point. SA has ~95% WhatsApp penetration. Residents will never log into a web portal to log an issue — they'll WhatsApp it.

**Pros:** Massive increase in resident adoption. Differentiator in SA market. Makes the maintenance module actually reflect how residents communicate.

**Cons:** Requires WhatsApp Business API (Meta partner program, approval process). Adds Twilio or 360dialog as a dependency. Incoming media handling (photos) needs server-side storage.

**Context:** Design the maintenance request creation API to accept an external webhook payload from day 1 — even if the WhatsApp integration itself is V2. The endpoint `/api/maintenance/inbound` should be designed in V1 (not necessarily implemented). WhatsApp Business API → Twilio → webhook → parse message + media → create maintenance_request row.

**Effort:** M
**Priority:** P2
**Depends on:** Maintenance module (V1), WhatsApp Business API approval

---

### STSMA Compliance Pulse

**What:** Monthly auto-assessment of each scheme's compliance with the Sectional Titles Schemes Management Act — AGM held on time, minutes distributed within 30 days, insurance policy current, reserve fund maintained at prescribed level, trustees elected within term limits.

**Why:** No other SA body corporate tool automates STSMA compliance checking. Managing agents currently track this manually per scheme, across a portfolio of 50+. This turns StrataHQ into a compliance tool, not just management software — significant moat.

**Pros:** Strong differentiator. Creates a recurring engagement loop (monthly compliance email). Gives trustees peace of mind. Strong enterprise/multi-scheme selling point.

**Cons:** Requires encoding STSMA legal requirements as code. STSMA gets amended — need a process for updating rules. Some compliance items (insurance policy expiry) require manual data entry.

**Context:** Phase 1: rule engine checks data already in StrataHQ (AGM dates from agm_meetings, minutes distribution from documents). Phase 2: add insurance policy + reserve fund fields to scheme settings. Phase 3: auto-generated compliance PDF report. The compliance score feeds into the scheme health score (see below).

**Effort:** L
**Priority:** P2
**Depends on:** AGM & Voting module, Document Vault (V1)

---

### Scheme Health Score

**What:** A single computed number (0–100) per scheme, visible in the managing agent portfolio view. Combines: levy collection rate (40%), open SLA-breached maintenance jobs (20%), days since last AGM vs required frequency (20%), document vault completeness (20%).

**Why:** Agents managing 50+ schemes can't review each one individually every morning. A health score makes the portfolio view actionable: sort by health score, click into the worst 3. Turns the portfolio from a list into a triage tool.

**Pros:** High impact on agent UX. Low technical effort (computed SQL view). Makes the portfolio view genuinely useful.

**Cons:** The formula is opinionated — agents may disagree with the weightings. Should be configurable in V3.

**Context:** Implement as a Postgres materialized view refreshed nightly: `scheme_health_scores(scheme_id, score, levy_collection_pct, open_sla_breach_count, days_since_agm, doc_completeness_pct, computed_at)`. Surface in the `/agent/dashboard` as a sortable column. Add a "needs attention" filter (score < 60). Feeds into STSMA compliance pulse.

**Effort:** S
**Priority:** P2
**Depends on:** Agent portfolio view (V1), Levy & Payments + Maintenance + AGM modules

---

### Predictive Levy Analytics

**What:** "At the current collection rate, your reserve fund will be depleted by August. Suggested levy increase: R45/unit/month." Forward-looking projections based on historical levy collection + expenditure patterns.

**Why:** Trustees make levy increase decisions based on gut feel. Giving them a data-driven projection makes StrataHQ feel like a financial advisor, not just a ledger. Strong differentiator.

**Pros:** Agents and trustees will demo this to potential new schemes. Creates recurring engagement (check the projection monthly).

**Cons:** Requires 6+ months of historical data to be meaningful. V1 schemes won't have data for predictions.

**Context:** Build the data model to collect clean historical data from V1 launch. Implement predictions in V2 once real scheme data exists. Start simple: linear projection of expenditure vs levy income, no ML. Display as a chart on the financial reporting page with a "recommended levy increase" call-out. Use the `levy_periods` + `levy_payments` + expense records already in the data model.

**Effort:** M
**Priority:** P2
**Depends on:** Financial Reporting module (V1), 6+ months of production data

---

## V3 — Future Vision

### Open API for Integrators

**What:** Public REST API allowing managing agents' accounting software (Pastel, Sage, QuickBooks) to pull levy data, payments, and scheme financials automatically.

**Why:** Managing agents run their accounting in separate software. Without an API, they manually re-enter StrataHQ data. An API eliminates double-entry and makes StrataHQ the authoritative data source.

**Pros:** Unlocks a partnership/integration ecosystem. Reduces churn (agents are more locked in when accounting software reads from StrataHQ).

**Cons:** API versioning, backwards compatibility, OAuth implementation, developer documentation. Significant ongoing maintenance commitment.

**Context:** Design data models in V1 with stable, meaningful IDs (already using UUIDs). In V3: OAuth 2.0 client credentials flow, per-scheme API keys as intermediate step, OpenAPI spec auto-generated from Server Actions. Start with read-only endpoints (levy accounts, payments, schemes) before write endpoints.

**Effort:** XL
**Priority:** P3
**Depends on:** All V1 modules, stable data model, auth infrastructure

---

### Contractor Pre-Vetting Marketplace

**What:** When assigning a maintenance job, agents pick from a curated directory of pre-vetted contractors by trade (plumbing, electrical, roofing) and suburb, with ratings from other StrataHQ schemes.

**Why:** Sourcing trustworthy contractors is a major pain point for agents managing schemes in new areas. A curated marketplace removes this friction and becomes a revenue opportunity (referral fees or premium contractor listings).

**Pros:** High value to agents. Revenue opportunity. Network effects (more schemes → more contractor ratings → better marketplace).

**Cons:** Requires contractor onboarding, vetting process, review system. Two-sided marketplace dynamics — needs contractor supply before it's useful. Risk: contractors game reviews.

**Context:** In V1, use a free-text contractor name + phone number on the maintenance_requests table. In V2, add a scheme-level `contractors` address book (agent's known contractors, not public). In V3, open the marketplace with public contractor profiles and cross-scheme ratings. This is a separate product build, not just a feature.

**Effort:** XL
**Priority:** P3
**Depends on:** Maintenance module (V1), significant scheme volume for network effects
