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
1. Exchange Supabase code for session
2. Query `memberships WHERE user_id = me`
3. No memberships + registered as managing agent → redirect `/agent/setup`
4. No memberships + registered via invite → redirect `/auth/pending`
5. Has memberships → branch on `memberships[0].role`:
   - `agent` → `/agent`
   - `trustee` / `resident` → `/app/[memberships[0].scheme_id]`

### `/auth/pending`
Static holding page: "Your access is being set up. You'll receive an email when you're ready to log in." No action required from user.

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

---

## Route Guards — `middleware.ts`

- Unauthenticated request to any non-`/auth/*` route → redirect `/auth/login`
- Authenticated request to `/agent/*` where role ≠ `agent` → redirect `/app/[their_scheme_id]`
- Authenticated request to `/app/[schemeId]/*` where user has no membership on that scheme → 404

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
