# Mock Data Layer — Design Spec

**Date:** 2026-03-27
**Status:** Approved
**Scope:** All 8 app modules + agent portfolio, for all 3 user roles

---

## Overview

Replace all hardcoded inline values across the app with a structured mock data layer. The data shapes match the eventual Supabase/Golang backend schema so that swapping mock → real API requires only changing the data source, not the components.

---

## Data Layer Architecture

**Approach: per-module mock files** under `lib/mock/`

Each file exports:
1. **TypeScript interfaces** — match the backend schema exactly
2. **A typed mock constant** — realistic SA strata data ready to consume

| File | Exports |
|------|---------|
| `lib/mock/scheme.ts` | `Scheme`, `Unit` types + `mockScheme`, `mockUnits[]` |
| `lib/mock/levy.ts` | `LevyPeriod`, `LevyAccount`, `LevyPayment` + mock data |
| `lib/mock/maintenance.ts` | `MaintenanceRequest` + mock data |
| `lib/mock/agm.ts` | `AgmMeeting`, `AgmResolution` + mock data |
| `lib/mock/members.ts` | `Member` + mock data |
| `lib/mock/financials.ts` | `BudgetLine`, `ReserveFund` + mock data |
| `lib/mock/communications.ts` | `Notice` + mock data |
| `lib/mock/documents.ts` | `SchemeDocument` + mock data |

`scheme.ts` is the shared base — all other modules reference its `Unit.id` and `Unit.identifier` values for consistency (e.g. levy accounts reference real unit IDs from the mock units array).

---

## TypeScript Interfaces

```ts
// lib/mock/scheme.ts
export interface Scheme {
  id: string
  name: string
  org_id: string
  unit_count: number
  address: string
  created_at: string
}

export interface Unit {
  id: string
  scheme_id: string
  identifier: string        // e.g. "1A", "4B"
  owner_name: string
  floor: number
  section_value: number     // participation quota percentage
}

// lib/mock/levy.ts
export interface LevyPeriod {
  id: string
  scheme_id: string
  amount_cents: number
  due_date: string          // ISO date
  label: string             // e.g. "October 2025"
}

export interface LevyAccount {
  id: string
  unit_id: string
  unit_identifier: string
  owner_name: string
  period_id: string
  amount_cents: number
  paid_cents: number
  status: 'paid' | 'partial' | 'overdue' | 'pending'
  due_date: string
  paid_date: string | null
}

export interface LevyPayment {
  id: string
  levy_account_id: string
  amount_cents: number
  date: string
  reference: string
  bank_ref: string | null
}

// lib/mock/maintenance.ts
export interface MaintenanceRequest {
  id: string
  scheme_id: string
  unit_id: string | null        // null = common property
  title: string
  description: string
  category: 'plumbing' | 'electrical' | 'structural' | 'garden' | 'pool' | 'other'
  status: 'open' | 'in_progress' | 'pending_approval' | 'resolved'
  contractor_name: string | null
  contractor_phone: string | null
  sla_hours: number
  created_at: string
  resolved_at: string | null
  submitted_by_unit: string | null  // unit identifier if resident-submitted
}

// lib/mock/agm.ts
export interface AgmMeeting {
  id: string
  scheme_id: string
  date: string
  quorum_required: number   // number of owners
  quorum_present: number
  status: 'upcoming' | 'in_progress' | 'closed'
}

export interface AgmResolution {
  id: string
  meeting_id: string
  title: string
  description: string
  votes_for: number
  votes_against: number
  total_eligible: number
  status: 'open' | 'passed' | 'failed'
}

// lib/mock/members.ts
export interface Member {
  id: string
  scheme_id: string
  unit_id: string
  unit_identifier: string
  name: string
  role: 'owner' | 'trustee' | 'resident'
  email: string
  phone: string | null
  is_trustee_committee: boolean
}

// lib/mock/financials.ts
export interface BudgetLine {
  id: string
  scheme_id: string
  category: string
  budgeted_cents: number
  actual_cents: number
  period_label: string
}

export interface ReserveFund {
  scheme_id: string
  balance_cents: number
  target_cents: number
  last_updated: string
}

// lib/mock/communications.ts
export interface Notice {
  id: string
  scheme_id: string
  title: string
  body: string
  sent_at: string
  sent_by_name: string
  type: 'general' | 'urgent' | 'agm' | 'levy'
}

// lib/mock/documents.ts
export interface SchemeDocument {
  id: string
  scheme_id: string
  name: string
  file_type: 'pdf' | 'docx' | 'xlsx' | 'jpg' | 'png'
  category: 'rules' | 'minutes' | 'insurance' | 'financial' | 'other'
  uploaded_at: string
  uploaded_by_name: string
  size_bytes: number
}
```

---

## Mock Data Content

All mock data is set in the scheme **"Sunridge Heights"** (schemeId: `scheme-001`), matching the login page mock. 8 units used consistently across all modules (`1A`, `2B`, `3A`, `4B`, `5A`, `6C`, `7B`, `8A` — 24 total units but 8 shown for table brevity).

Data is realistic South African strata context:
- Levy amounts in Rands (stored as cents)
- South African names for owners
- SA addresses and contractor names
- STSMA-relevant document categories

---

## Module Pages — UI

Each module page (`app/app/[schemeId]/<module>/page.tsx`) reads `user.role` from `useMockAuth()` and renders the appropriate view. The page component is the only place role-gating logic lives — no conditional rendering buried in sub-components.

### Overview (`/app/[schemeId]`)
| Role | Stat cards | Main content |
|------|-----------|-------------|
| Agent / Trustee | Units · levy collection % · open maintenance · days to AGM | Scheme health indicators · recent activity feed · quick links to overdue items |
| Resident | My levy status · open requests · next AGM date | Welcome card · my unit details · recent notices |

### Levy & Payments
| Role | View |
|------|------|
| Agent | 6-month collection rate chart · full levy roll (all units, status pills) · overdue list |
| Trustee | Same as agent, read-only (no action buttons) |
| Resident | My levy card (current period) · payment history (6 months) · statement download button |

### Maintenance
| Role | View |
|------|------|
| Agent | Stat strip (open · SLA breaches · avg resolution) · all work orders with contractor + SLA |
| Trustee | Same, read-only — pending approvals highlighted |
| Resident | My submitted requests · Submit new request button · request status timeline |

### AGM & Voting
| Role | View |
|------|------|
| Agent / Trustee | AGM meeting card · all resolutions with vote bars · quorum tracker |
| Resident | Resolutions list · vote buttons (if status: open) · results when closed |

### Members
| Role | View |
|------|------|
| Agent | Full roster: owners + trustees + residents · contact details · unit mapping |
| Trustee | Full roster, read-only |
| Resident | Trustee committee contacts only |

### Financials
| Role | View |
|------|------|
| Agent / Trustee | Stat cards (budget · spent · reserve · surplus) · budget vs actual table by category · reserve fund bar |
| Resident | Reserve fund health bar + brief plain-language summary only |

### Communications
| Role | View |
|------|------|
| Agent | Sent notices list with type badges · Compose notice button |
| Trustee / Resident | Received notices list, expandable body |

### Documents
| Role | View |
|------|------|
| Agent | Documents grouped by category · Upload document button |
| Trustee / Resident | Same list, read-only (download only) |

---

## Agent Portfolio Pages

### `/agent` — Portfolio Overview
- Stat cards: active schemes · total units managed · open maintenance jobs across portfolio · avg collection rate
- Scheme list table: name · units · collection rate · open jobs · health indicator (green/amber/red)

### `/agent/schemes` — All Schemes
- Full scheme cards with: name · unit count · address · collection rate · open maintenance count · "View →" link

---

## Role-Gating Pattern

```tsx
// Standard pattern used in every module page
const { user } = useMockAuth()

if (user?.role === 'resident') {
  return <ResidentView data={...} unitIdentifier={user.unitIdentifier} />
}
return <AgentTrusteeView data={...} canEdit={user?.role === 'agent'} />
```

Trustee and agent share the same view component; `canEdit` prop controls whether action buttons render.

---

## What Does NOT Change

- `lib/mock-auth.tsx` — unchanged, auth layer is separate from data layer
- `components/Sidebar.tsx` — unchanged
- `components/AppShell.tsx` — unchanged
- All marketing/landing page components — unchanged
- Existing `AGMMockPanel`, `LevyMockPanel`, `MaintenanceMockPanel` in `/demo` — unchanged (they're demo components, not app pages)

---

## Future Backend Integration

To swap a module from mock → real API:
1. Replace the mock constant import with a `fetch()` / Supabase query that returns the same type
2. The TypeScript interface stays — it becomes the API response contract
3. No component changes required

Recommended swap order (matches backend build priority from TODOS.md):
1. Levy & Payments (highest agent value)
2. Maintenance
3. AGM & Voting
4. Members
5. Financials
6. Communications
7. Documents
