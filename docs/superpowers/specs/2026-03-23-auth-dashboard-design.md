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
Trustees and residents are invited via Supabase's `auth.admin.inviteUserByEmail()` called from the wizard step 5 and from `/agent/invitations`. The invite email contains a token. When the invited user clicks the link:
- Supabase handles the token exchange automatically and redirects to `/auth/callback`
- The callback reads `user_metadata` set by the invite (role + scheme_id + unit_id, stored when the invite was created)
- A `memberships` row is created server-side in the callback before redirecting
- Expired/invalid token: Supabase returns an error code; redirect to `/auth/login?error=invalid_invite`

### Form error states
- **Login — wrong password / user not found**: inline error below password field: "Incorrect email or password."
- **Login — email not confirmed**: "Please confirm your email before logging in."
- **Login — network error**: "Something went wrong. Please try again."
- **Register — email already in use**: "An account with this email already exists. Log in instead."
- **Register — success**: replace form with: "Check your email — we've sent you a confirmation link."
- All errors use Supabase error codes; do not expose raw error messages to the user.

---

## Setup Wizard — `/agent/setup`

Shown only to new managing agents (no memberships). Saves data incrementally — each step commits before advancing so closing mid-wizard doesn't lose work.

| Step | Fields | Notes |
|------|--------|-------|
| 1 — Firm | Company name, contact email, phone | Creates `organisations` row |
| 2 — Scheme | Scheme name, physical address, scheme number (SS XX/YYYY) | Creates `schemes` row |
| 3 — Units | Unit count, unit identifiers (e.g. 1A, 1B, 2A) | Creates `units` rows |
| 4 — Levies | Base levy amount, admin levy, levy period | Stored on scheme for now |
| 5 — Invite | Email addresses for trustees + residents, assign unit per resident | Sends Supabase invite emails; skippable |

Progress bar shows current step. "Back" is always available. "Skip for now" available on step 5.

On completion → redirect to `/agent`.

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
Role-aware nav list. Reads role from Supabase session JWT claims. Renders the correct nav items and scheme/org name in header. Handles scheme switcher UI when trustee has 2+ schemes.

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

Valid `memberships.role` values: `'agent'` | `'trustee'` | `'resident'` — enforced via CHECK constraint.

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

Middleware uses `@supabase/ssr` `createServerClient` and calls `supabase.auth.getUser()` on every request to refresh the session token. It does **not** trust `getSession()` alone.

**Role × route matrix:**

| Request path | Unauthenticated | Agent | Trustee/Resident | No membership yet |
|---|---|---|---|---|
| `/auth/*` | allow | allow | allow | allow |
| `/agent/setup` | → `/auth/login` | allow | → `/app/[scheme_id]` | allow (new agent) |
| `/agent/*` (other) | → `/auth/login` | allow | → `/app/[scheme_id]` | → `/agent/setup` |
| `/app/[schemeId]/*` | → `/auth/login` | allow (agent can view any scheme they manage) | allow if membership exists on that scheme | → `/auth/pending` |
| `/app/[schemeId]/*` where user has no membership on that scheme | — | 404 | 404 | — |
| `/auth/pending` | allow | allow | allow | allow |

The middleware reads `user_metadata.registered_as` and queries memberships to determine routing. For performance, memberships are checked once per request and the result is passed via request headers to layouts.

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
