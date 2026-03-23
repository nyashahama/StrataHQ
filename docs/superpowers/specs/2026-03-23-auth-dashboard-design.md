# Auth Flow + Role-Based Dashboard Shells — Design Spec

**Date:** 2026-03-23
**Project:** StrataHQ
**Stack:** Next.js 16 App Router · React 19 · TypeScript · Tailwind · Supabase Auth

---

## Overview

Build the complete auth flow and role-based dashboard shells for StrataHQ's three user types: Managing Agent, Trustee, and Resident/Owner. This is the first authenticated layer of the application — every route behind login uses these shells. Module pages (levy, maintenance, etc.) are placeholder shells; their content is out of scope for this spec.

---

## User Roles

| Role | Entry path | Scope |
|------|-----------|-------|
| Managing Agent | Registers directly | Portfolio view across all schemes they manage |
| Trustee | Invited by agent | Single scheme (switcher if on 2+) |
| Resident/Owner | Invited by agent | Single unit within a scheme |

---

## Auth Pages

### `/auth/login`
- Email + password form
- "Forgot password" link (no-op placeholder for V1)
- "Don't have an account?" → `/auth/register`
- On success: POST to Supabase, redirect to `/auth/callback`

### `/auth/register`
- Full name, email, password
- Role selector: **"I'm a managing agent"** or **"I was invited"**
  - "I was invited" hides the role — invitation token handles role assignment
- On success: POST to Supabase, redirect to `/auth/callback`

### `/auth/callback/route.ts`
Server-side route handler:
1. Exchange Supabase code for session via `supabase.auth.exchangeCodeForSession(code)`
2. Call `supabase.auth.getUser()` — use this, not `getSession()`, to get the verified user
3. Query `memberships WHERE user_id = me ORDER BY created_at ASC`
4. **No memberships + user registered as managing agent** → redirect `/agent/setup`
   - How to detect: store `registered_as` in `user_metadata` during `signUp` call (`{ data: { registered_as: 'agent' | 'invited' } }`)
5. **No memberships + user registered as invited** → redirect `/auth/pending`
6. **Has memberships** → pick first membership, branch on `role`:
   - `'agent'` → `/agent`
   - `'trustee'` → `/app/[scheme_id]`
   - `'resident'` → `/app/[scheme_id]`
   - Any other value → redirect `/auth/login?error=unknown_role` (future-proof fallback)
7. **Error exchanging code** → redirect `/auth/login?error=auth_failed`

Note: A user can only hold one role in V1. `memberships[0]` is always the primary. Multi-role support is out of scope.

### `/auth/pending`
Static holding page shown to invited users who have not yet been accepted.

- Copy: "Your access is being set up. You'll receive an email once your account is activated."
- Shows a "Refresh" button that re-queries memberships and redirects if they now exist
- The agent activates the user by completing their invitation in `/agent/invitations` (separate flow, out of scope for this spec — the endpoint will exist, the page is a placeholder)
- If user revisits `/auth/login` after being activated, the callback will route them normally

### Invitation flow
Trustees and residents are invited via Supabase's `auth.admin.inviteUserByEmail()` (called from wizard step 5 server action using the service role key). The call embeds the membership data in the invite:

```ts
await supabaseAdmin.auth.admin.inviteUserByEmail(email, {
  data: { role: 'trustee' | 'resident', scheme_id: '...', unit_id: '...' | null }
})
```

When the invited user clicks the email link, Supabase exchanges the token and redirects to `/auth/callback`. Invited users **do not** go through `/auth/register` — the invite link takes them directly to the callback. The `registered_as` metadata check in the callback is only relevant for users who signed up via the register form.

The callback then:
1. Reads `user.user_metadata` (contains `role`, `scheme_id`, `unit_id` from the invite)
2. Inserts a `memberships` row using the **Supabase service role client** (bypasses RLS — the invited user has no existing membership to satisfy the INSERT policy)
3. Redirects to `/app/[scheme_id]`

Expired/invalid token: Supabase returns an error; redirect to `/auth/login?error=invalid_invite`.

The "I was invited" option on the register form is shown purely to handle users who arrive at `/auth/register` without a valid invite link. On submit, instead of creating an account, show: "You need an invitation to join StrataHQ. Contact your managing agent." No account is created.

### Form error states
- **Login — wrong password / user not found**: inline error below password field: "Incorrect email or password."
- **Login — email not confirmed**: "Please confirm your email before logging in."
- **Login — network error**: "Something went wrong. Please try again."
- **Register — email already in use**: "An account with this email already exists. Log in instead."
- **Register — success**: replace form with: "Check your email — we've sent you a confirmation link."
- All errors use Supabase error codes; do not expose raw error messages to the user.

---

## Setup Wizard — `/agent/setup`

Shown only to new managing agents (no memberships). Saves data incrementally — each step calls a server action before advancing so closing mid-wizard doesn't lose work.

**RLS note:** The setup wizard server actions run using the **service role key** (via `supabaseAdmin`), because the agent has no `memberships` row yet and therefore cannot satisfy row-level security on `organisations` or `schemes`. After wizard completion the agent's membership row exists and all subsequent operations use the standard anon key + RLS.

If the wizard is abandoned mid-way, the partially-created org/scheme is cleaned up on next login (the callback sees no memberships, detects `registered_as: 'agent'`, and redirects back to `/agent/setup` which resumes from the last completed step — stored in `wizard_progress` field on the `organisations` row, values: `'firm' | 'scheme' | 'units' | 'levies' | 'invite' | 'complete'`).

| Step | Fields | Server action | Notes |
|------|--------|--------------|-------|
| 1 — Firm | Company name, contact email, phone | `createOrganisation()` | Creates `organisations` row; stores `wizard_progress: 'firm'` |
| 2 — Scheme | Scheme name, physical address, scheme number (SS XX/YYYY) | `createScheme()` | Creates `schemes` row linked to org |
| 3 — Units | Unit count; list of unit identifiers (e.g. 1A, 1B, 2A) | `createUnits()` | Creates one `units` row per identifier |
| 4 — Levies | Base levy (ZAR), admin levy (ZAR), levy period | `updateSchemeLevies()` | Levy period values: `'monthly' \| 'quarterly' \| 'bi-annual' \| 'annual'` |
| 5 — Invite | List of { email, role, unit_id? } | `sendInvites()` | Calls `supabaseAdmin.auth.admin.inviteUserByEmail`; skippable |

On completion: set `wizard_progress: 'complete'`, create agent's own `memberships` row (`role: 'agent'`, `scheme_id` of first scheme, `unit_id: null`), then redirect to `/agent`.

Progress bar shows current step (1–5). "Back" is always available. "Skip for now" on step 5 only.

---

## Dashboard Layout

**Layout C — Full-width labeled sidebar (~200px)**

- Sidebar always visible with icon + text labels
- Scheme name / organisation name in sidebar header
- Active item highlighted with left border accent
- Settings / profile link pinned to bottom

### Managing Agent — Two sidebar contexts

**Portfolio view** (`/agent/*`):
```
[Acme Property Management]
─────────────────────────
⊞  Portfolio overview      ← /agent
⊟  All schemes             ← /agent/schemes
✉  Invitations             ← /agent/invitations
─────────────────────────
⚙  Settings
```

**Scheme view** (`/app/[schemeId]/*`):
```
[Sunridge Heights ▾]       ← scheme switcher
─────────────────────────
⊞  Overview
💳  Levy & Payments
🔧  Maintenance
🗳  AGM & Voting
📢  Communications
📁  Documents
📊  Financials
─────────────────────────
👥  Members
```

### Trustee sidebar
Same as agent scheme view minus "Members". Scheme switcher shown if trustee belongs to 2+ schemes.

### Resident sidebar
```
[Unit 4B · Sunridge Heights]
─────────────────────────
⊞  Overview
💳  My Levy
🔧  Maintenance
📣  Notices
📁  Documents
─────────────────────────
⚙  My Profile
```
No financials tab. No member management. Intentionally simpler.

---

## Route Structure

```
app/
  auth/
    login/page.tsx
    register/page.tsx
    callback/route.ts
    pending/page.tsx

  agent/
    layout.tsx           ← agent-only AppShell (portfolio sidebar)
    page.tsx             ← portfolio overview
    setup/page.tsx       ← setup wizard
    schemes/page.tsx
    invitations/page.tsx

  app/
    [schemeId]/
      layout.tsx         ← scheme-scoped AppShell (role-aware sidebar)
      page.tsx           ← scheme overview
      levy/page.tsx
      maintenance/page.tsx
      agm/page.tsx
      communications/page.tsx
      documents/page.tsx
      financials/page.tsx
      members/page.tsx   ← agent + trustee only

middleware.ts            ← route guards
```

---

## Components

### `components/AppShell.tsx`
Layout wrapper: sidebar + main content area. Accepts `sidebar` slot and `children`. Used by both agent and scheme layouts.

### `components/Sidebar.tsx`
Role-aware nav list. Receives `role`, `orgName`/`schemeName`, and `allMemberships` as props from the parent layout server component (which fetches these from Supabase). Does **not** read from JWT claims — role comes from the DB via props. Renders the correct nav items and name in header. Handles scheme switcher dropdown when trustee has 2+ memberships.

### `components/SetupWizard.tsx`
Client component. Step state managed locally (1–5). Each "Next" calls a server action to persist the step's data before advancing. Shows linear progress bar. Renders step-specific form fields.

### `components/auth/LoginForm.tsx`
Client component. Email + password fields. Calls Supabase `signInWithPassword`. Handles error display inline.

### `components/auth/RegisterForm.tsx`
Client component. Name, email, password, role selector. Calls Supabase `signUp`. On success shows "Check your email" confirmation.

---

## Database Schema (V1 minimal)

```sql
organisations (
  id          uuid primary key,
  name        text not null,
  type        text default 'managing_agent'
)

schemes (
  id             uuid primary key,
  name           text not null,
  address        text,
  scheme_number  text,
  org_id         uuid references organisations(id),
  base_levy      numeric,
  admin_levy     numeric,
  levy_period    text
)

units (
  id          uuid primary key,
  scheme_id   uuid references schemes(id),
  identifier  text not null   -- e.g. "1A", "4B"
)

memberships (
  id          uuid primary key,
  user_id     uuid references auth.users(id),
  scheme_id   uuid references schemes(id),
  role        text not null,  -- 'agent' | 'trustee' | 'resident'
  unit_id     uuid references units(id)  -- nullable (agents have no unit)
)
```

RLS enabled on all tables. Policies filter by `scheme_id` matching user's memberships.

Valid `memberships.role` values: `'agent'` | `'trustee'` | `'resident'` — enforced via `CHECK (role IN ('agent','trustee','resident'))`.

Add `wizard_progress` to `organisations`:
```sql
wizard_progress text default 'firm'
  check (wizard_progress in ('firm','scheme','units','levies','invite','complete'))
```

All tables include `created_at timestamptz default now()` and `updated_at timestamptz default now()`.

**RLS policy pattern (example for `schemes`):**
```sql
-- Users can read schemes they have a membership on
create policy "scheme_select" on schemes for select
  using (id in (select scheme_id from memberships where user_id = auth.uid()));
```
Each table follows this same pattern. Agents additionally have insert/update access on tables scoped to their org's schemes.

---

## Route Guards — `middleware.ts`

Middleware uses `@supabase/ssr` `createServerClient` and calls `supabase.auth.getUser()` on every request to refresh the session token. Does **not** trust `getSession()` alone.

Middleware queries `memberships WHERE user_id = me` once per request. Results are set as request headers for layouts to consume without a second DB call:
- `x-user-role`: `'agent' | 'trustee' | 'resident' | ''`
- `x-user-scheme-id`: primary `scheme_id` (first membership) or `''`
- `x-user-org-name`: organisation name (agents only) or `''`

**Role × route matrix:**

| Request path | Unauthenticated | Agent | Trustee/Resident | Invited, no membership |
|---|---|---|---|---|
| `/auth/*` | allow | allow | allow | allow |
| `/agent/setup` | → `/auth/login` | → `/agent` if wizard complete; allow if not | → `/app/[scheme_id]` | allow (new agent) |
| `/agent/*` (other) | → `/auth/login` | allow | → `/app/[scheme_id]` | → `/agent/setup` |
| `/app/[schemeId]/*` | → `/auth/login` | allow if agent's org manages that scheme | allow if has membership on that schemeId | → `/auth/pending` |
| `/app/[schemeId]/*` (wrong scheme) | — | 404 | 404 | — |
| `/auth/pending` | allow | allow | allow | allow |

"Wizard complete" detected by `wizard_progress = 'complete'` on the agent's `organisations` row.

### Scheme switcher (trustees on 2+ schemes)
When a trustee has memberships on multiple schemes, the sidebar header shows a dropdown: scheme name + "▾". Clicking it reveals a list of schemes by name. Selecting one navigates to `/app/[schemeId]`. The switcher is rendered by `Sidebar.tsx` and receives `allMemberships` as a prop from the scheme layout server component. In V1, up to 10 schemes per trustee. This UI is implemented as part of the sidebar, not as a separate page.

---

## Module Placeholder Pages

Each module page (levy, maintenance, agm, communications, documents, financials, members) renders:
- Page title (h1) matching the nav label
- Breadcrumb: Scheme name → Module name
- Placeholder content block with muted description ("Levy management — coming soon")
- Sidebar remains fully active

This ensures all nav links are live and functional from day one.

---

## Out of Scope (V1)

- Google / social OAuth
- Forgot password flow (link present, no-op)
- Module page content (levy tables, maintenance tickets, etc.)
- Email template customisation
- Multi-org support (one managing agent org per account)
