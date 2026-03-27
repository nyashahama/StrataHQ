# Mock Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a typed mock data layer under `lib/mock/` and build out all 8 "coming soon" module pages for all 3 user roles (agent, trustee, resident).

**Architecture:** Per-module mock files export TypeScript interfaces + typed constants. Pages read `user.role` from `useMockAuth()` and render the appropriate role view inline. No new component files — all view logic stays in the page file.

**Tech Stack:** Next.js 14 App Router, TypeScript, Tailwind CSS (tokens: `ink`, `muted`, `accent`, `green`, `red`, `yellowbg`, `border`, `page`)

---

## File Map

**Create:**
- `lib/mock/scheme.ts` — Scheme + Unit types and mock data (shared base)
- `lib/mock/levy.ts` — LevyPeriod, LevyAccount, LevyPayment types + mock data
- `lib/mock/maintenance.ts` — MaintenanceRequest type + mock data
- `lib/mock/agm.ts` — AgmMeeting, AgmResolution types + mock data
- `lib/mock/members.ts` — Member type + mock data
- `lib/mock/financials.ts` — BudgetLine, ReserveFund types + mock data
- `lib/mock/communications.ts` — Notice type + mock data
- `lib/mock/documents.ts` — SchemeDocument type + mock data

**Modify:**
- `app/app/[schemeId]/page.tsx` — Overview, role-aware
- `app/app/[schemeId]/levy/page.tsx` — Levy roll (agent/trustee) or my levy (resident)
- `app/app/[schemeId]/maintenance/page.tsx` — Work orders (agent/trustee) or my requests (resident)
- `app/app/[schemeId]/agm/page.tsx` — Resolutions + quorum (agent/trustee/resident)
- `app/app/[schemeId]/members/page.tsx` — Full roster (agent/trustee) or trustees only (resident)
- `app/app/[schemeId]/financials/page.tsx` — Budget table (agent/trustee) or summary (resident)
- `app/app/[schemeId]/communications/page.tsx` — Notices list, compose button for agent only
- `app/app/[schemeId]/documents/page.tsx` — Documents list, upload button for agent only
- `app/agent/page.tsx` — Portfolio overview with per-scheme stats
- `app/agent/schemes/page.tsx` — Scheme cards with health indicators

---

## Task 1: lib/mock/scheme.ts

**Files:**
- Create: `lib/mock/scheme.ts`

- [ ] **Create the file with full content:**

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
  identifier: string      // e.g. "1A", "4B"
  owner_name: string
  floor: number
  section_value: number   // participation quota as percentage e.g. 4.17
}

export const mockScheme: Scheme = {
  id: 'scheme-001',
  name: 'Sunridge Heights',
  org_id: 'org-001',
  unit_count: 24,
  address: '14 Sunridge Drive, Claremont, Cape Town, 7708',
  created_at: '2024-01-15T08:00:00Z',
}

// 8 representative units used across all module mock data
export const mockUnits: Unit[] = [
  { id: 'unit-1a', scheme_id: 'scheme-001', identifier: '1A', owner_name: 'Henderson, T.', floor: 1, section_value: 4.17 },
  { id: 'unit-2b', scheme_id: 'scheme-001', identifier: '2B', owner_name: 'Molefe, S.',     floor: 2, section_value: 4.17 },
  { id: 'unit-3a', scheme_id: 'scheme-001', identifier: '3A', owner_name: 'van der Berg, L.', floor: 3, section_value: 4.17 },
  { id: 'unit-4b', scheme_id: 'scheme-001', identifier: '4B', owner_name: 'Naidoo, R.',    floor: 4, section_value: 4.17 },
  { id: 'unit-5a', scheme_id: 'scheme-001', identifier: '5A', owner_name: 'Khumalo, B.',   floor: 5, section_value: 4.17 },
  { id: 'unit-6c', scheme_id: 'scheme-001', identifier: '6C', owner_name: 'Abrahams, J.', floor: 6, section_value: 4.17 },
  { id: 'unit-7b', scheme_id: 'scheme-001', identifier: '7B', owner_name: 'Petersen, M.', floor: 7, section_value: 4.17 },
  { id: 'unit-8a', scheme_id: 'scheme-001', identifier: '8A', owner_name: 'Dlamini, S.',  floor: 8, section_value: 4.17 },
]

// Agent portfolio — all 3 schemes managed by org-001
export interface PortfolioScheme {
  id: string
  name: string
  unit_count: number
  address: string
  levy_collection_pct: number   // e.g. 91
  open_maintenance_count: number
  health: 'good' | 'fair' | 'poor'
}

export const mockPortfolio: PortfolioScheme[] = [
  {
    id: 'scheme-001',
    name: 'Sunridge Heights',
    unit_count: 24,
    address: 'Claremont, Cape Town',
    levy_collection_pct: 91,
    open_maintenance_count: 7,
    health: 'good',
  },
  {
    id: 'scheme-002',
    name: 'Bayside Manor',
    unit_count: 12,
    address: 'Sea Point, Cape Town',
    levy_collection_pct: 75,
    open_maintenance_count: 4,
    health: 'fair',
  },
  {
    id: 'scheme-003',
    name: 'The Palms Estate',
    unit_count: 16,
    address: 'Kenilworth, Cape Town',
    levy_collection_pct: 58,
    open_maintenance_count: 12,
    health: 'poor',
  },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/scheme.ts
git commit -m "feat(mock): add scheme and unit mock data"
```

---

## Task 2: lib/mock/levy.ts

**Files:**
- Create: `lib/mock/levy.ts`

- [ ] **Create the file with full content:**

```ts
// lib/mock/levy.ts

export interface LevyPeriod {
  id: string
  scheme_id: string
  amount_cents: number    // standard levy amount e.g. 245000 = R2,450
  due_date: string        // ISO date
  label: string           // e.g. "October 2025"
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

export const mockLevyPeriod: LevyPeriod = {
  id: 'period-oct-2025',
  scheme_id: 'scheme-001',
  amount_cents: 245000,
  due_date: '2025-10-01',
  label: 'October 2025',
}

export const mockLevyRoll: LevyAccount[] = [
  { id: 'la-001', unit_id: 'unit-1a', unit_identifier: '1A', owner_name: 'Henderson, T.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-28' },
  { id: 'la-002', unit_id: 'unit-2b', unit_identifier: '2B', owner_name: 'Molefe, S.',        period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 120000, status: 'partial', due_date: '2025-10-01', paid_date: '2025-10-03' },
  { id: 'la-003', unit_id: 'unit-3a', unit_identifier: '3A', owner_name: 'van der Berg, L.', period_id: 'period-oct-2025', amount_cents: 310000, paid_cents: 0,      status: 'overdue', due_date: '2025-10-01', paid_date: null },
  { id: 'la-004', unit_id: 'unit-4b', unit_identifier: '4B', owner_name: 'Naidoo, R.',       period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-30' },
  { id: 'la-005', unit_id: 'unit-5a', unit_identifier: '5A', owner_name: 'Khumalo, B.',      period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-10-01' },
  { id: 'la-006', unit_id: 'unit-6c', unit_identifier: '6C', owner_name: 'Abrahams, J.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 0,      status: 'overdue', due_date: '2025-10-01', paid_date: null },
  { id: 'la-007', unit_id: 'unit-7b', unit_identifier: '7B', owner_name: 'Petersen, M.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-29' },
  { id: 'la-008', unit_id: 'unit-8a', unit_identifier: '8A', owner_name: 'Dlamini, S.',     period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-10-02' },
]

// 6-month collection trend (May–Oct 2025)
export const mockCollectionTrend: { month: string; pct: number }[] = [
  { month: 'May', pct: 87 },
  { month: 'Jun', pct: 89 },
  { month: 'Jul', pct: 88 },
  { month: 'Aug', pct: 91 },
  { month: 'Sep', pct: 92 },
  { month: 'Oct', pct: 94 },
]

// Payment history for Unit 4B (resident view)
export const mockUnit4BPayments: LevyPayment[] = [
  { id: 'pay-001', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-09-30', reference: 'SH-4B-OCT25', bank_ref: 'FNB-9283471' },
  { id: 'pay-002', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-08-29', reference: 'SH-4B-SEP25', bank_ref: 'FNB-9274312' },
  { id: 'pay-003', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-07-31', reference: 'SH-4B-AUG25', bank_ref: 'FNB-9265193' },
  { id: 'pay-004', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-06-28', reference: 'SH-4B-JUL25', bank_ref: 'FNB-9256044' },
  { id: 'pay-005', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-05-30', reference: 'SH-4B-JUN25', bank_ref: 'FNB-9246875' },
  { id: 'pay-006', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-04-29', reference: 'SH-4B-MAY25', bank_ref: 'FNB-9237706' },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/levy.ts
git commit -m "feat(mock): add levy mock data"
```

---

## Task 3: lib/mock/maintenance.ts

**Files:**
- Create: `lib/mock/maintenance.ts`

- [ ] **Create the file with full content:**

```ts
// lib/mock/maintenance.ts

export interface MaintenanceRequest {
  id: string
  scheme_id: string
  unit_id: string | null        // null = common property job
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

export const mockMaintenanceRequests: MaintenanceRequest[] = [
  {
    id: 'mr-001',
    scheme_id: 'scheme-001',
    unit_id: 'unit-2b',
    title: 'Shower drain blocked — Unit 2B',
    description: 'Resident reports shower draining very slowly, likely hair blockage.',
    category: 'plumbing',
    status: 'in_progress',
    contractor_name: 'Rapid Plumbing Co.',
    contractor_phone: '021 555 0123',
    sla_hours: 48,
    created_at: '2025-10-14T09:00:00Z',
    resolved_at: null,
    submitted_by_unit: '2B',
  },
  {
    id: 'mr-002',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Parking bay lights not working',
    description: 'Three overhead lights in basement parking bay B not functioning.',
    category: 'electrical',
    status: 'open',
    contractor_name: null,
    contractor_phone: null,
    sla_hours: 24,
    created_at: '2025-10-15T14:30:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-003',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Pool pump replacement',
    description: 'Pool pump has failed and requires full replacement. Quote obtained.',
    category: 'pool',
    status: 'pending_approval',
    contractor_name: 'AquaFix Pool Services',
    contractor_phone: '021 555 0456',
    sla_hours: 72,
    created_at: '2025-10-12T11:00:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-004',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Garden service — monthly',
    description: 'Scheduled monthly garden maintenance and lawn cutting.',
    category: 'garden',
    status: 'resolved',
    contractor_name: 'GreenThumb Gardens',
    contractor_phone: '021 555 0789',
    sla_hours: 8,
    created_at: '2025-10-10T07:00:00Z',
    resolved_at: '2025-10-10T11:00:00Z',
    submitted_by_unit: null,
  },
  {
    id: 'mr-005',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Lift service certificate renewal',
    description: 'Annual lift inspection due. Booking with certified inspector.',
    category: 'structural',
    status: 'in_progress',
    contractor_name: 'Cape Lift Services',
    contractor_phone: '021 555 0321',
    sla_hours: 96,
    created_at: '2025-10-08T10:00:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-006',
    scheme_id: 'scheme-001',
    unit_id: 'unit-4b',
    title: 'Leaking tap in kitchen — Unit 4B',
    description: 'Persistent drip from kitchen mixer tap.',
    category: 'plumbing',
    status: 'resolved',
    contractor_name: 'Rapid Plumbing Co.',
    contractor_phone: '021 555 0123',
    sla_hours: 48,
    created_at: '2025-09-20T13:00:00Z',
    resolved_at: '2025-09-22T10:00:00Z',
    submitted_by_unit: '4B',
  },
  {
    id: 'mr-007',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Intercom system fault — Block A',
    description: 'Intercom for units 1A–4B not ringing. Possible wiring fault.',
    category: 'electrical',
    status: 'open',
    contractor_name: null,
    contractor_phone: null,
    sla_hours: 24,
    created_at: '2025-10-16T08:45:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/maintenance.ts
git commit -m "feat(mock): add maintenance mock data"
```

---

## Task 4: lib/mock/agm.ts

**Files:**
- Create: `lib/mock/agm.ts`

- [ ] **Create the file with full content:**

```ts
// lib/mock/agm.ts

export interface AgmMeeting {
  id: string
  scheme_id: string
  date: string              // ISO date
  quorum_required: number   // minimum owners required
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

export const mockAgmMeeting: AgmMeeting = {
  id: 'agm-2025',
  scheme_id: 'scheme-001',
  date: '2025-11-14',
  quorum_required: 25,
  quorum_present: 38,
  status: 'closed',
}

export const mockAgmResolutions: AgmResolution[] = [
  {
    id: 'res-001',
    meeting_id: 'agm-2025',
    title: 'Approval of 2026 maintenance budget',
    description: 'Proposed total maintenance budget of R485,000 for the financial year 2026, covering all scheduled and reactive maintenance.',
    votes_for: 31,
    votes_against: 7,
    total_eligible: 48,
    status: 'passed',
  },
  {
    id: 'res-002',
    meeting_id: 'agm-2025',
    title: 'Levy increase — 6% from January 2026',
    description: 'Proposed increase of standard levy from R2,450 to R2,597 per month effective 1 January 2026, in line with CPI.',
    votes_for: 26,
    votes_against: 12,
    total_eligible: 48,
    status: 'passed',
  },
  {
    id: 'res-003',
    meeting_id: 'agm-2025',
    title: 'Appointment of trustees for 2025–2026',
    description: 'Re-appointment of Henderson, T. (Chair), Molefe, S., and van der Berg, L. as trustees for the 2025–2026 term.',
    votes_for: 35,
    votes_against: 3,
    total_eligible: 48,
    status: 'passed',
  },
]

// Upcoming AGM for next year
export const mockUpcomingAgm: AgmMeeting = {
  id: 'agm-2026',
  scheme_id: 'scheme-001',
  date: '2026-11-20',
  quorum_required: 25,
  quorum_present: 0,
  status: 'upcoming',
}
```

- [ ] **Commit:**

```bash
git add lib/mock/agm.ts
git commit -m "feat(mock): add AGM mock data"
```

---

## Task 5: lib/mock/members.ts

**Files:**
- Create: `lib/mock/members.ts`

- [ ] **Create the file with full content:**

```ts
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

export const mockMembers: Member[] = [
  { id: 'mem-001', scheme_id: 'scheme-001', unit_id: 'unit-1a', unit_identifier: '1A', name: 'Henderson, T.',    role: 'trustee',  email: 'thenderson@email.co.za',  phone: '082 555 0101', is_trustee_committee: true },
  { id: 'mem-002', scheme_id: 'scheme-001', unit_id: 'unit-2b', unit_identifier: '2B', name: 'Molefe, S.',        role: 'trustee',  email: 'smolefe@email.co.za',      phone: '083 555 0202', is_trustee_committee: true },
  { id: 'mem-003', scheme_id: 'scheme-001', unit_id: 'unit-3a', unit_identifier: '3A', name: 'van der Berg, L.', role: 'trustee',  email: 'lvanderberg@email.co.za',  phone: '084 555 0303', is_trustee_committee: true },
  { id: 'mem-004', scheme_id: 'scheme-001', unit_id: 'unit-4b', unit_identifier: '4B', name: 'Naidoo, R.',       role: 'owner',    email: 'rnaidoo@email.co.za',      phone: '071 555 0404', is_trustee_committee: false },
  { id: 'mem-005', scheme_id: 'scheme-001', unit_id: 'unit-5a', unit_identifier: '5A', name: 'Khumalo, B.',      role: 'owner',    email: 'bkhumalo@email.co.za',     phone: '072 555 0505', is_trustee_committee: false },
  { id: 'mem-006', scheme_id: 'scheme-001', unit_id: 'unit-6c', unit_identifier: '6C', name: 'Abrahams, J.',    role: 'owner',    email: 'jabrahams@email.co.za',    phone: null,           is_trustee_committee: false },
  { id: 'mem-007', scheme_id: 'scheme-001', unit_id: 'unit-7b', unit_identifier: '7B', name: 'Petersen, M.',    role: 'resident', email: 'mpetersen@email.co.za',    phone: '073 555 0707', is_trustee_committee: false },
  { id: 'mem-008', scheme_id: 'scheme-001', unit_id: 'unit-8a', unit_identifier: '8A', name: 'Dlamini, S.',     role: 'owner',    email: 'sdlamini@email.co.za',     phone: '074 555 0808', is_trustee_committee: false },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/members.ts
git commit -m "feat(mock): add members mock data"
```

---

## Task 6: lib/mock/financials.ts

**Files:**
- Create: `lib/mock/financials.ts`

- [ ] **Create the file with full content:**

```ts
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

export const mockBudgetLines: BudgetLine[] = [
  { id: 'bl-001', scheme_id: 'scheme-001', category: 'Administration',      budgeted_cents: 4800000,  actual_cents: 4620000,  period_label: '2025' },
  { id: 'bl-002', scheme_id: 'scheme-001', category: 'Cleaning',            budgeted_cents: 3600000,  actual_cents: 3600000,  period_label: '2025' },
  { id: 'bl-003', scheme_id: 'scheme-001', category: 'Maintenance',         budgeted_cents: 18500000, actual_cents: 21340000, period_label: '2025' },
  { id: 'bl-004', scheme_id: 'scheme-001', category: 'Insurance',           budgeted_cents: 9600000,  actual_cents: 9600000,  period_label: '2025' },
  { id: 'bl-005', scheme_id: 'scheme-001', category: 'Electricity (common)', budgeted_cents: 7200000, actual_cents: 6890000,  period_label: '2025' },
  { id: 'bl-006', scheme_id: 'scheme-001', category: 'Landscaping',         budgeted_cents: 2400000,  actual_cents: 2280000,  period_label: '2025' },
  { id: 'bl-007', scheme_id: 'scheme-001', category: 'Pool maintenance',    budgeted_cents: 1800000,  actual_cents: 3960000,  period_label: '2025' },
  { id: 'bl-008', scheme_id: 'scheme-001', category: 'Reserve levy',        budgeted_cents: 600000,   actual_cents: 546000,   period_label: '2025' },
]

export const mockReserveFund: ReserveFund = {
  scheme_id: 'scheme-001',
  balance_cents: 18450000,    // R184,500
  target_cents: 36000000,     // R360,000 (10-year maintenance plan target)
  last_updated: '2025-10-01T00:00:00Z',
}
```

- [ ] **Commit:**

```bash
git add lib/mock/financials.ts
git commit -m "feat(mock): add financials mock data"
```

---

## Task 7: lib/mock/communications.ts

**Files:**
- Create: `lib/mock/communications.ts`

- [ ] **Create the file with full content:**

```ts
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

export const mockNotices: Notice[] = [
  {
    id: 'notice-001',
    scheme_id: 'scheme-001',
    title: 'Annual General Meeting — 14 November 2025',
    body: 'Dear Owners,\n\nYou are hereby notified that the Annual General Meeting of Sunridge Heights Body Corporate will be held on Thursday, 14 November 2025 at 18:30 in the complex communal room.\n\nAgenda: (1) Approval of 2026 maintenance budget, (2) Levy increase, (3) Trustee elections.\n\nProxy forms must be submitted by 12 November 2025.',
    sent_at: '2025-10-24T10:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'agm',
  },
  {
    id: 'notice-002',
    scheme_id: 'scheme-001',
    title: 'Urgent: Water supply interruption — 18 October 2025',
    body: 'Dear Residents,\n\nPlease be advised that the City of Cape Town will be conducting maintenance on the main water supply line on Saturday 18 October 2025 from 08:00–14:00. All units will experience no water supply during this period.\n\nPlease make alternative arrangements.',
    sent_at: '2025-10-16T14:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'urgent',
  },
  {
    id: 'notice-003',
    scheme_id: 'scheme-001',
    title: 'October levy reminder',
    body: 'Dear Owners,\n\nThis is a reminder that October levies of R2,450 were due on 1 October 2025. If you have not yet made payment, please do so immediately to avoid additional administration fees.\n\nPayment reference: SH-[UNIT]-OCT25',
    sent_at: '2025-10-07T09:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'levy',
  },
  {
    id: 'notice-004',
    scheme_id: 'scheme-001',
    title: 'Pool pump replacement — update',
    body: 'Dear Residents,\n\nThe pool pump has been confirmed as beyond repair. A replacement quote of R8,400 (inclusive) has been received from AquaFix Pool Services. The trustees are reviewing the quote for approval.\n\nThe pool will remain closed until the replacement is complete. We apologise for the inconvenience.',
    sent_at: '2025-10-13T11:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'general',
  },
  {
    id: 'notice-005',
    scheme_id: 'scheme-001',
    title: 'Year-end building inspection — 5 December 2025',
    body: 'Dear Residents,\n\nPlease be advised that our annual building inspection will take place on Friday, 5 December 2025. The inspector will require access to all units between 09:00 and 15:00. If you are unable to be present, please make arrangements with the managing agent by 28 November 2025.',
    sent_at: '2025-10-01T08:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'general',
  },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/communications.ts
git commit -m "feat(mock): add communications mock data"
```

---

## Task 8: lib/mock/documents.ts

**Files:**
- Create: `lib/mock/documents.ts`

- [ ] **Create the file with full content:**

```ts
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

export const mockDocuments: SchemeDocument[] = [
  { id: 'doc-001', scheme_id: 'scheme-001', name: 'Conduct Rules — Sunridge Heights',  file_type: 'pdf',  category: 'rules',     uploaded_at: '2024-02-01T10:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 524288  },
  { id: 'doc-002', scheme_id: 'scheme-001', name: 'AGM Minutes — November 2025',       file_type: 'pdf',  category: 'minutes',   uploaded_at: '2025-11-21T14:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 286720  },
  { id: 'doc-003', scheme_id: 'scheme-001', name: 'AGM Minutes — November 2024',       file_type: 'pdf',  category: 'minutes',   uploaded_at: '2024-11-18T14:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 311296  },
  { id: 'doc-004', scheme_id: 'scheme-001', name: 'Insurance Certificate 2025–2026',   file_type: 'pdf',  category: 'insurance', uploaded_at: '2025-01-05T09:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 204800  },
  { id: 'doc-005', scheme_id: 'scheme-001', name: 'Approved Budget 2026',              file_type: 'xlsx', category: 'financial', uploaded_at: '2025-11-21T14:30:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 45056   },
  { id: 'doc-006', scheme_id: 'scheme-001', name: 'Management Agreement',              file_type: 'pdf',  category: 'other',     uploaded_at: '2024-01-20T09:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 638976  },
  { id: 'doc-007', scheme_id: 'scheme-001', name: '10-Year Maintenance Plan',          file_type: 'pdf',  category: 'financial', uploaded_at: '2024-06-10T11:00:00Z', uploaded_by_name: 'Acme Property Management', size_bytes: 1048576 },
]
```

- [ ] **Commit:**

```bash
git add lib/mock/documents.ts
git commit -m "feat(mock): add documents mock data"
```

---

## Task 9: Levy & Payments page

**Files:**
- Modify: `app/app/[schemeId]/levy/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend, mockUnit4BPayments } from '@/lib/mock/levy'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

const STATUS_STYLES: Record<string, string> = {
  paid:    'bg-green-bg text-green',
  partial: 'bg-yellowbg text-[#92400e]',
  overdue: 'bg-red-bg text-red',
  pending: 'bg-accent-bg text-accent',
}

export default function LevyPaymentsPage() {
  const { user } = useMockAuth()

  if (user?.role === 'resident') {
    const myAccount = mockLevyRoll.find(a => a.unit_identifier === user.unitIdentifier)
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › My Levy</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Levy</h1>
        <p className="text-[14px] text-muted mb-8">Levy account for Unit {user.unitIdentifier}.</p>

        {/* Current levy card */}
        <div className="bg-white border border-border rounded-lg px-6 py-5 mb-6 flex items-center justify-between">
          <div>
            <p className="text-[12px] text-muted mb-1">{mockLevyPeriod.label} · due {new Date(mockLevyPeriod.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}</p>
            <p className="font-serif text-[32px] font-semibold text-ink leading-none">{formatRand(mockLevyPeriod.amount_cents)}</p>
          </div>
          {myAccount && (
            <span className={`text-[12px] font-semibold px-3 py-1 rounded-full ${STATUS_STYLES[myAccount.status]}`}>
              {myAccount.status.charAt(0).toUpperCase() + myAccount.status.slice(1)}
            </span>
          )}
        </div>

        {/* Payment history */}
        <h2 className="text-[14px] font-semibold text-ink mb-3">Payment history</h2>
        <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
          {mockUnit4BPayments.map((p, i) => (
            <div key={p.id} className={`flex items-center justify-between px-5 py-3 text-[13px] ${i < mockUnit4BPayments.length - 1 ? 'border-b border-border' : ''}`}>
              <div>
                <span className="font-medium text-ink">{formatRand(p.amount_cents)}</span>
                <span className="text-muted ml-3">{new Date(p.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}</span>
              </div>
              <span className="text-[11px] text-muted font-mono">{p.reference}</span>
            </div>
          ))}
        </div>

        <button className="text-[12px] text-accent font-medium border border-accent rounded px-4 py-2 hover:bg-accent-dim transition-colors">
          Download statement (PDF)
        </button>
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const collected = mockLevyRoll.filter(a => a.status === 'paid').length
  const overdue = mockLevyRoll.filter(a => a.status === 'overdue').length
  const totalCollected = mockLevyRoll.reduce((sum, a) => sum + a.paid_cents, 0)
  const latestPct = mockCollectionTrend[mockCollectionTrend.length - 1].pct

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Levy & Payments</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Levy & Payments</h1>
      <p className="text-[14px] text-muted mb-8">Levy collection, statements, and payment history.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Collection rate', value: `${latestPct}%` },
          { label: 'Total collected', value: formatRand(totalCollected) },
          { label: 'Overdue', value: String(overdue) },
          { label: 'Monthly levy', value: formatRand(mockLevyPeriod.amount_cents) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Collection trend chart */}
      <div className="bg-white border border-border rounded-lg px-6 py-5 mb-6">
        <p className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-4">Collection rate — 6 months</p>
        <div className="flex items-end gap-2 h-[56px] mb-1">
          {mockCollectionTrend.map((d, i) => {
            const isLast = i === mockCollectionTrend.length - 1
            const heightPct = ((d.pct - 80) / 20) * 100
            return (
              <div key={d.month} className="flex-1 flex flex-col items-center justify-end gap-1">
                <span className="text-[9px] font-semibold leading-none" style={{ color: isLast ? '#2B6CB0' : '#A8A49E' }}>{d.pct}%</span>
                <div className="w-full rounded-[2px]" style={{ height: `${Math.max(heightPct, 8)}%`, background: isLast ? '#2B6CB0' : '#E3E2DF' }} />
              </div>
            )
          })}
        </div>
        <div className="flex">
          {mockCollectionTrend.map(d => (
            <span key={d.month} className="flex-1 text-center text-[9px] text-muted">{d.month}</span>
          ))}
        </div>
      </div>

      {/* Levy roll */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Levy Roll — {mockLevyPeriod.label}</span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">{mockLevyRoll.length} shown</span>
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Unit</span><span>Amount</span><span>Due</span><span>Status</span>
          </div>
          {mockLevyRoll.map((account) => (
            <div key={account.id} className="grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 border-b border-border last:border-b-0 text-[13px]">
              <div>
                <div className="font-semibold text-ink">Unit {account.unit_identifier}</div>
                <div className="text-[12px] text-muted">{account.owner_name}</div>
              </div>
              <span className="font-semibold text-ink tabular-nums">{formatRand(account.amount_cents)}</span>
              <span className="text-[12px] text-muted">{new Date(account.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}</span>
              <div className="flex items-center gap-2">
                <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[account.status]}`}>
                  {account.status.charAt(0).toUpperCase() + account.status.slice(1)}
                </span>
                {canEdit && account.status === 'overdue' && (
                  <button className="text-[11px] text-accent font-medium hover:underline">Remind</button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Login as **agent** at `http://localhost:3000/auth/login` → navigate to `/app/scheme-001/levy`
  - Should see: 4 stat cards, bar chart, full levy roll with status pills and "Remind" buttons on overdue rows
  - Login as **trustee** → same page, "Remind" buttons should be absent
  - Login as **resident** (unit 4B) → should see "My Levy" heading, current levy card showing Paid, 6-month payment history

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/levy/page.tsx
git commit -m "feat(levy): build levy page with role-aware views"
```

---

## Task 10: Maintenance page

**Files:**
- Modify: `app/app/[schemeId]/maintenance/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMaintenanceRequests, type MaintenanceRequest } from '@/lib/mock/maintenance'

const STATUS_STYLES: Record<string, string> = {
  open:             'bg-red-bg text-red',
  in_progress:      'bg-yellowbg text-[#92400e]',
  pending_approval: 'bg-accent-bg text-accent',
  resolved:         'bg-green-bg text-green',
}

const STATUS_LABELS: Record<string, string> = {
  open:             'Open',
  in_progress:      'In progress',
  pending_approval: 'Pending',
  resolved:         'Resolved',
}

const CATEGORY_ICONS: Record<string, string> = {
  plumbing:   '🚿',
  electrical: '💡',
  structural: '🏗️',
  garden:     '🌿',
  pool:       '🏊',
  other:      '🔧',
}

function isSlaBreached(req: MaintenanceRequest): boolean {
  if (req.status === 'resolved') return false
  const created = new Date(req.created_at).getTime()
  const now = new Date('2025-10-16T12:00:00Z').getTime() // fixed "now" for mock
  const hoursElapsed = (now - created) / (1000 * 60 * 60)
  return hoursElapsed > req.sla_hours
}

export default function MaintenancePage() {
  const { user } = useMockAuth()

  // Resident: show only their unit's requests
  if (user?.role === 'resident') {
    const myRequests = mockMaintenanceRequests.filter(
      r => r.submitted_by_unit === user.unitIdentifier
    )
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
        <p className="text-[14px] text-muted mb-8">Your maintenance requests for Unit {user.unitIdentifier}.</p>

        <div className="flex items-center justify-between mb-6">
          <span className="text-[13px] text-muted">{myRequests.length} request{myRequests.length !== 1 ? 's' : ''}</span>
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Submit request
          </button>
        </div>

        {myRequests.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
            No maintenance requests submitted yet.
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {myRequests.map(req => (
              <div key={req.id} className="bg-white border border-border rounded-lg px-5 py-4 flex gap-3">
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[16px]">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[13px] font-semibold text-ink">{req.title}</span>
                    <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_STYLES[req.status]}`}>
                      {STATUS_LABELS[req.status]}
                    </span>
                  </div>
                  <div className="text-[12px] text-muted">{req.contractor_name ?? 'No contractor assigned'}</div>
                  <div className="text-[11px] text-muted mt-1">
                    Submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                    {req.resolved_at && ` · Resolved ${new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const open = mockMaintenanceRequests.filter(r => r.status !== 'resolved')
  const breached = open.filter(isSlaBreached)
  const pendingApproval = open.filter(r => r.status === 'pending_approval')
  const resolvedCount = mockMaintenanceRequests.filter(r => r.status === 'resolved').length

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
      <p className="text-[14px] text-muted mb-8">Log and track maintenance requests.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Open jobs',         value: String(open.length) },
          { label: 'SLA breaches',      value: String(breached.length) },
          { label: 'Pending approval',  value: String(pendingApproval.length) },
          { label: 'Resolved this month', value: String(resolvedCount) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Work orders */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Work Orders</span>
          {canEdit && (
            <button className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors">
              + New job
            </button>
          )}
        </div>
        <div className="px-5 py-3 flex flex-col gap-0">
          {mockMaintenanceRequests.map((req, i) => {
            const breached = isSlaBreached(req)
            return (
              <div key={req.id} className={`flex gap-3 items-start py-4 ${i < mockMaintenanceRequests.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[15px] mt-0.5">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold text-ink mb-[2px]">{req.title}</div>
                  <div className="text-[11px] text-muted">
                    {req.contractor_name ? `${req.contractor_name}` : 'No contractor assigned'}
                    {req.submitted_by_unit && ` · Unit ${req.submitted_by_unit}`}
                  </div>
                  {breached && (
                    <div className="text-[11px] text-red font-medium mt-[2px]">⚠ SLA breached</div>
                  )}
                  {!breached && req.status !== 'resolved' && (
                    <div className="text-[11px] text-muted mt-[2px]">
                      SLA: {req.sla_hours}h · submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                  {req.status === 'resolved' && req.resolved_at && (
                    <div className="text-[11px] text-green mt-[2px]">
                      Resolved {new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[req.status]}`}>
                    {STATUS_LABELS[req.status]}
                  </span>
                  {canEdit && req.status === 'pending_approval' && (
                    <button className="text-[11px] text-accent font-medium hover:underline">Approve</button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Login as **agent** → `/app/scheme-001/maintenance`: 4 stat cards, all 7 work orders, SLA breach markers, "Approve" button on pending_approval row, "+ New job" button
  - Login as **trustee** → same list, no "+ New job", no "Approve" buttons
  - Login as **resident** (unit 4B) → "Your maintenance requests" heading, sees mr-001 (Unit 2B — wait, this was submitted_by_unit '2B'), and mr-006 (Unit 4B). Only mr-006 shows. Submit button visible.

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/maintenance/page.tsx
git commit -m "feat(maintenance): build maintenance page with role-aware views"
```

---

## Task 11: AGM & Voting page

**Files:**
- Modify: `app/app/[schemeId]/agm/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockAgmMeeting, mockAgmResolutions, mockUpcomingAgm } from '@/lib/mock/agm'

export default function AgmVotingPage() {
  const { user } = useMockAuth()

  const meeting = mockAgmMeeting
  const upcoming = mockUpcomingAgm

  const quorumPct = Math.round((meeting.quorum_present / (meeting.quorum_required * 2)) * 100)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › AGM & Voting</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">AGM & Voting</h1>
      <p className="text-[14px] text-muted mb-8">Annual general meetings and trustee resolutions.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Last AGM', value: new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Next AGM', value: new Date(upcoming.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Quorum', value: `${meeting.quorum_present}/${meeting.quorum_required * 2}` },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Meeting summary */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">
            AGM — {new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
          </span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">Closed</span>
        </div>
        {/* Quorum bar */}
        <div className="px-5 py-4 border-b border-border">
          <div className="flex justify-between text-[11px] text-muted mb-2">
            <span className="font-semibold text-ink">Quorum reached ✓</span>
            <span>{meeting.quorum_present} of {meeting.quorum_required * 2} owners present</span>
          </div>
          <div className="h-[6px] bg-border rounded-full overflow-hidden">
            <div className="h-full bg-green rounded-full" style={{ width: `${Math.min(quorumPct, 100)}%` }} />
          </div>
        </div>

        {/* Resolutions */}
        <div className="px-5 py-4 flex flex-col gap-4">
          {mockAgmResolutions.map((res, i) => {
            const forPct = Math.round((res.votes_for / res.total_eligible) * 100)
            return (
              <div key={res.id} className={`${i < mockAgmResolutions.length - 1 ? 'pb-4 border-b border-border' : ''}`}>
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div>
                    <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {i + 1} OF {mockAgmResolutions.length}</div>
                    <div className="text-[13px] font-semibold text-ink">{res.title}</div>
                    <div className="text-[12px] text-muted mt-1">{res.description}</div>
                  </div>
                  <span className={`flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full ${res.status === 'passed' ? 'bg-green-bg text-green' : 'bg-red-bg text-red'}`}>
                    {res.status.charAt(0).toUpperCase() + res.status.slice(1)}
                  </span>
                </div>
                <div className="flex justify-between text-[11px] text-muted mb-1">
                  <span>In favour · {res.votes_for} votes</span>
                  <span>Against · {res.votes_against}</span>
                </div>
                <div className="h-[6px] bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-accent rounded-full" style={{ width: `${forPct}%` }} />
                </div>
                {user?.role === 'resident' && res.status === 'open' && (
                  <div className="flex gap-2 mt-3">
                    <button className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:bg-[#245a96] transition-colors">Vote in favour</button>
                    <button className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors">Vote against</button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - All 3 roles → `/app/scheme-001/agm`: quorum bar, 3 resolutions with vote bars, passed badges
  - Resident: if any resolution had status `'open'`, vote buttons would appear (none do in mock since all are 'passed' — this is correct)

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/agm/page.tsx
git commit -m "feat(agm): build AGM page with resolutions and quorum"
```

---

## Task 12: Members page

**Files:**
- Modify: `app/app/[schemeId]/members/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMembers } from '@/lib/mock/members'

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  owner:    'bg-[#f0efe9] text-muted',
  resident: 'bg-green-bg text-green',
}

export default function MembersPage() {
  const { user } = useMockAuth()

  // Resident: trustees only
  if (user?.role === 'resident') {
    const trustees = mockMembers.filter(m => m.is_trustee_committee)
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Trustee Committee</h1>
        <p className="text-[14px] text-muted mb-8">Contact the trustees for scheme-related matters.</p>
        <div className="flex flex-col gap-3">
          {trustees.map(m => (
            <div key={m.id} className="bg-white border border-border rounded-lg px-5 py-4 flex items-center justify-between">
              <div>
                <div className="text-[14px] font-semibold text-ink">{m.name}</div>
                <div className="text-[12px] text-muted mt-0.5">Unit {m.unit_identifier} · {m.email}</div>
              </div>
              {m.phone && <span className="text-[12px] text-muted">{m.phone}</span>}
            </div>
          ))}
        </div>
      </div>
    )
  }

  // Agent / Trustee: full roster
  const canEdit = user?.role === 'agent'
  const trustees = mockMembers.filter(m => m.is_trustee_committee)
  const owners = mockMembers.filter(m => !m.is_trustee_committee)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Members</h1>
      <p className="text-[14px] text-muted mb-8">Owners, trustees, and contact information.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Total members', value: String(mockMembers.length) },
          { label: 'Trustees', value: String(trustees.length) },
          { label: 'Owners', value: String(owners.length) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Members table */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">All members</span>
          {canEdit && (
            <button className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors">
              + Invite member
            </button>
          )}
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[auto_1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Unit</span><span>Name</span><span>Contact</span><span>Role</span>
          </div>
          {mockMembers.map((m, i) => (
            <div key={m.id} className={`grid grid-cols-[auto_1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockMembers.length - 1 ? 'border-b border-border' : ''}`}>
              <span className="font-semibold text-ink w-8">{m.unit_identifier}</span>
              <span className="text-ink">{m.name}</span>
              <span className="text-[12px] text-muted">{m.phone ?? '—'}</span>
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[m.role]}`}>
                {m.is_trustee_committee ? 'Trustee' : m.role.charAt(0).toUpperCase() + m.role.slice(1)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Agent → full roster with stat cards and "+ Invite member" button
  - Trustee → full roster, no invite button
  - Resident → "Trustee Committee" heading with 3 trustee cards only

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/members/page.tsx
git commit -m "feat(members): build members page with role-aware views"
```

---

## Task 13: Financials page

**Files:**
- Modify: `app/app/[schemeId]/financials/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockBudgetLines, mockReserveFund } from '@/lib/mock/financials'

function formatRand(cents: number): string {
  return `R ${(cents / 100).toLocaleString('en-ZA', { minimumFractionDigits: 0 })}`
}

export default function FinancialsPage() {
  const { user } = useMockAuth()

  const totalBudgeted = mockBudgetLines.reduce((s, l) => s + l.budgeted_cents, 0)
  const totalActual   = mockBudgetLines.reduce((s, l) => s + l.actual_cents, 0)
  const surplus = totalBudgeted - totalActual
  const reservePct = Math.round((mockReserveFund.balance_cents / mockReserveFund.target_cents) * 100)

  // Resident: simplified summary only
  if (user?.role === 'resident') {
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Financials</h1>
        <p className="text-[14px] text-muted mb-8">Scheme financial health summary.</p>

        {/* Reserve fund */}
        <div className="bg-white border border-border rounded-lg px-6 py-5 mb-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[13px] font-semibold text-ink">Reserve Fund</span>
            <span className="text-[12px] text-muted">{reservePct}% of target</span>
          </div>
          <div className="h-3 bg-border rounded-full overflow-hidden mb-2">
            <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
          </div>
          <div className="flex justify-between text-[12px] text-muted">
            <span>Balance: <strong className="text-ink">{formatRand(mockReserveFund.balance_cents)}</strong></span>
            <span>Target: {formatRand(mockReserveFund.target_cents)}</span>
          </div>
        </div>

        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-5 text-[13px] text-ink leading-relaxed">
          The scheme's total approved budget for 2025 is <strong>{formatRand(totalBudgeted)}</strong>.
          Expenditure to date is <strong>{formatRand(totalActual)}</strong>.
          The reserve fund stands at {reservePct}% of the 10-year maintenance plan target.
        </div>
      </div>
    )
  }

  // Agent / Trustee: full financial view
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Financials</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Financials</h1>
      <p className="text-[14px] text-muted mb-8">Budget, expenditure, and reserve fund.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Total budget',  value: formatRand(totalBudgeted) },
          { label: 'Spent to date', value: formatRand(totalActual) },
          { label: 'Reserve fund',  value: formatRand(mockReserveFund.balance_cents) },
          { label: surplus >= 0 ? 'Surplus' : 'Deficit', value: formatRand(Math.abs(surplus)) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Reserve fund bar */}
      <div className="bg-white border border-border rounded-lg px-6 py-5 mb-6">
        <div className="flex items-center justify-between mb-3">
          <span className="text-[13px] font-semibold text-ink">Reserve Fund — {reservePct}% of target</span>
          <span className="text-[12px] text-muted">Target: {formatRand(mockReserveFund.target_cents)}</span>
        </div>
        <div className="h-3 bg-border rounded-full overflow-hidden">
          <div className="h-full bg-accent rounded-full" style={{ width: `${reservePct}%` }} />
        </div>
      </div>

      {/* Budget vs actual table */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Budget vs Actual — 2025</span>
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Category</span><span>Budgeted</span><span>Actual</span><span>Variance</span>
          </div>
          {mockBudgetLines.map((line, i) => {
            const variance = line.budgeted_cents - line.actual_cents
            const over = variance < 0
            return (
              <div key={line.id} className={`grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockBudgetLines.length - 1 ? 'border-b border-border' : ''}`}>
                <span className="text-ink">{line.category}</span>
                <span className="tabular-nums text-muted">{formatRand(line.budgeted_cents)}</span>
                <span className="tabular-nums text-ink font-medium">{formatRand(line.actual_cents)}</span>
                <span className={`tabular-nums text-[12px] font-semibold ${over ? 'text-red' : 'text-green'}`}>
                  {over ? '+' : '-'}{formatRand(Math.abs(variance))}
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Agent/Trustee → 4 stat cards, reserve fund bar at ~51%, 8-row budget table with variance column (maintenance and pool in red, others in green)
  - Resident → simplified summary paragraph + reserve fund bar only

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/financials/page.tsx
git commit -m "feat(financials): build financials page with role-aware views"
```

---

## Task 14: Communications page

**Files:**
- Modify: `app/app/[schemeId]/communications/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockNotices, type Notice } from '@/lib/mock/communications'

const TYPE_STYLES: Record<Notice['type'], string> = {
  general: 'bg-[#f0efe9] text-muted',
  urgent:  'bg-red-bg text-red',
  agm:     'bg-accent-bg text-accent',
  levy:    'bg-yellowbg text-[#92400e]',
}

const TYPE_LABELS: Record<Notice['type'], string> = {
  general: 'General',
  urgent:  'Urgent',
  agm:     'AGM',
  levy:    'Levy',
}

export default function CommunicationsPage() {
  const { user } = useMockAuth()
  const [expanded, setExpanded] = useState<string | null>(null)
  const canCompose = user?.role === 'agent'

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Communications</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Communications</h1>
      <p className="text-[14px] text-muted mb-8">Notices, announcements, and correspondence.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{mockNotices.length} notices</span>
        {canCompose && (
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Compose notice
          </button>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {mockNotices.map(notice => (
          <div key={notice.id} className="bg-white border border-border rounded-lg overflow-hidden">
            <button
              className="w-full px-5 py-4 flex items-start justify-between gap-4 text-left hover:bg-page transition-colors"
              onClick={() => setExpanded(expanded === notice.id ? null : notice.id)}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`text-[10px] font-semibold px-2 py-[2px] rounded-full ${TYPE_STYLES[notice.type]}`}>
                    {TYPE_LABELS[notice.type]}
                  </span>
                  <span className="text-[11px] text-muted">
                    {new Date(notice.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                  </span>
                </div>
                <div className="text-[13px] font-semibold text-ink">{notice.title}</div>
                <div className="text-[12px] text-muted mt-0.5">{notice.sent_by_name}</div>
              </div>
              <span className="text-muted text-[12px] flex-shrink-0 mt-1">{expanded === notice.id ? '▲' : '▼'}</span>
            </button>
            {expanded === notice.id && (
              <div className="px-5 pb-4 border-t border-border">
                <p className="text-[13px] text-ink leading-relaxed whitespace-pre-line pt-4">{notice.body}</p>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Agent → sees 5 notices with "+ Compose notice" button. Click a notice to expand body.
  - Trustee / Resident → same notices list, no compose button

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/communications/page.tsx
git commit -m "feat(communications): build communications page with expandable notices"
```

---

## Task 15: Documents page

**Files:**
- Modify: `app/app/[schemeId]/documents/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockDocuments, type SchemeDocument } from '@/lib/mock/documents'

const CATEGORY_LABELS: Record<SchemeDocument['category'], string> = {
  rules:     'Conduct Rules',
  minutes:   'AGM Minutes',
  insurance: 'Insurance',
  financial: 'Financial',
  other:     'Other',
}

const FILE_TYPE_STYLES: Record<string, string> = {
  pdf:  'bg-red-bg text-red',
  xlsx: 'bg-green-bg text-green',
  docx: 'bg-accent-bg text-accent',
}

function formatBytes(bytes: number): string {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// Group documents by category
function groupByCategory(docs: SchemeDocument[]): Record<string, SchemeDocument[]> {
  return docs.reduce((acc, doc) => {
    const key = CATEGORY_LABELS[doc.category]
    if (!acc[key]) acc[key] = []
    acc[key].push(doc)
    return acc
  }, {} as Record<string, SchemeDocument[]>)
}

export default function DocumentsPage() {
  const { user } = useMockAuth()
  const canUpload = user?.role === 'agent'
  const grouped = groupByCategory(mockDocuments)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Documents</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Documents</h1>
      <p className="text-[14px] text-muted mb-8">Scheme rules, minutes, and shared files.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{mockDocuments.length} documents</span>
        {canUpload && (
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Upload document
          </button>
        )}
      </div>

      <div className="flex flex-col gap-6">
        {Object.entries(grouped).map(([category, docs]) => (
          <div key={category}>
            <h2 className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-3">{category}</h2>
            <div className="bg-white border border-border rounded-lg overflow-hidden">
              {docs.map((doc, i) => (
                <div key={doc.id} className={`flex items-center gap-4 px-5 py-3 text-[13px] ${i < docs.length - 1 ? 'border-b border-border' : ''}`}>
                  <span className={`text-[10px] font-bold px-[6px] py-[2px] rounded uppercase ${FILE_TYPE_STYLES[doc.file_type] ?? 'bg-page text-muted'}`}>
                    {doc.file_type}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-ink truncate">{doc.name}</div>
                    <div className="text-[11px] text-muted mt-0.5">
                      {new Date(doc.uploaded_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                      {' · '}{formatBytes(doc.size_bytes)}
                    </div>
                  </div>
                  <button className="text-[12px] text-accent font-medium hover:underline flex-shrink-0">
                    Download
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Agent → grouped document list with "+ Upload document" button, PDF/XLSX badges
  - Trustee / Resident → same list, no upload button

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/documents/page.tsx
git commit -m "feat(documents): build documents page with grouped file list"
```

---

## Task 16: Overview page

**Files:**
- Modify: `app/app/[schemeId]/page.tsx`

- [ ] **Replace file content:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockScheme } from '@/lib/mock/scheme'
import { mockLevyRoll, mockCollectionTrend } from '@/lib/mock/levy'
import { mockMaintenanceRequests } from '@/lib/mock/maintenance'
import { mockAgmMeeting, mockUpcomingAgm } from '@/lib/mock/agm'
import { mockNotices } from '@/lib/mock/communications'

function daysUntil(dateStr: string): number {
  const now = new Date('2025-10-16')
  const target = new Date(dateStr)
  return Math.ceil((target.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

export default function SchemeOverviewPage() {
  const { user } = useMockAuth()

  const openMaintenance = mockMaintenanceRequests.filter(r => r.status !== 'resolved')
  const collectionPct = mockCollectionTrend[mockCollectionTrend.length - 1].pct
  const daysToAgm = daysUntil(mockUpcomingAgm.date)

  // Resident view
  if (user?.role === 'resident') {
    const myLevyAccount = mockLevyRoll.find(a => a.unit_identifier === user.unitIdentifier)
    const myRequests = mockMaintenanceRequests.filter(
      r => r.submitted_by_unit === user.unitIdentifier && r.status !== 'resolved'
    )
    const recentNotices = mockNotices.slice(0, 3)

    return (
      <div className="px-8 py-8 max-w-[900px]">
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
          {mockScheme.name}
        </h1>
        <p className="text-[14px] text-muted mb-8">Unit {user.unitIdentifier} · Welcome back.</p>

        <div className="grid grid-cols-3 gap-4 mb-8">
          {[
            { label: 'My levy', value: myLevyAccount ? (myLevyAccount.status === 'paid' ? 'Paid ✓' : 'Due') : '—' },
            { label: 'Open requests', value: String(myRequests.length) },
            { label: 'Days to AGM', value: String(daysToAgm) },
          ].map(({ label, value }) => (
            <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
              <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
              <div className="text-[12px] text-muted">{label}</div>
            </div>
          ))}
        </div>

        <h2 className="text-[13px] font-semibold text-ink mb-3">Recent notices</h2>
        <div className="flex flex-col gap-2">
          {recentNotices.map(n => (
            <div key={n.id} className="bg-white border border-border rounded-lg px-5 py-3 flex items-center justify-between">
              <div>
                <div className="text-[13px] font-medium text-ink">{n.title}</div>
                <div className="text-[11px] text-muted mt-0.5">
                  {new Date(n.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                </div>
              </div>
              <span className="text-[12px] text-accent font-medium">Read →</span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  // Agent / Trustee view
  const slaBreaches = openMaintenance.filter(r => {
    const created = new Date(r.created_at).getTime()
    const now = new Date('2025-10-16T12:00:00Z').getTime()
    return (now - created) / (1000 * 60 * 60) > r.sla_hours
  })

  const overdueLevy = mockLevyRoll.filter(a => a.status === 'overdue')

  const recentActivity = [
    { text: `Unit 1A levy received — R2,450`,                     time: '2 hours ago',  type: 'levy' },
    { text: `Pool pump replacement approved — AquaFix R8,400`,    time: '4 hours ago',  type: 'maintenance' },
    { text: `Water supply notice sent to all residents`,           time: '1 day ago',    type: 'comms' },
    { text: `Unit 6C levy overdue — reminder sent`,               time: '2 days ago',   type: 'levy' },
    { text: `Parking bay lights — job opened`,                    time: '2 days ago',   type: 'maintenance' },
  ]

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        {mockScheme.name}
      </h1>
      <p className="text-[14px] text-muted mb-8">Scheme at a glance.</p>

      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Total units',          value: String(mockScheme.unit_count) },
          { label: 'Levies collected',     value: `${collectionPct}%` },
          { label: 'Open maintenance',     value: String(openMaintenance.length) },
          { label: 'Days to AGM',          value: String(daysToAgm) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Activity feed */}
        <div>
          <h2 className="text-[13px] font-semibold text-ink mb-3">Recent activity</h2>
          <div className="bg-white border border-border rounded-lg overflow-hidden">
            {recentActivity.map((a, i) => (
              <div key={i} className={`px-5 py-3 ${i < recentActivity.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="text-[12px] text-ink">{a.text}</div>
                <div className="text-[11px] text-muted mt-0.5">{a.time}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Attention items */}
        <div>
          <h2 className="text-[13px] font-semibold text-ink mb-3">Needs attention</h2>
          <div className="flex flex-col gap-2">
            {slaBreaches.length > 0 && (
              <div className="bg-red-bg border border-red/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-red">{slaBreaches.length} SLA breach{slaBreaches.length > 1 ? 'es' : ''}</div>
                <div className="text-[11px] text-red/70 mt-0.5">Maintenance jobs past SLA</div>
              </div>
            )}
            {overdueLevy.length > 0 && (
              <div className="bg-yellowbg border border-[#92400e]/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-[#92400e]">{overdueLevy.length} overdue lev{overdueLevy.length > 1 ? 'ies' : 'y'}</div>
                <div className="text-[11px] text-[#92400e]/70 mt-0.5">
                  Units: {overdueLevy.map(a => a.unit_identifier).join(', ')}
                </div>
              </div>
            )}
            {slaBreaches.length === 0 && overdueLevy.length === 0 && (
              <div className="bg-green-bg border border-green/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-green">All clear</div>
                <div className="text-[11px] text-green/70 mt-0.5">No urgent items</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Agent/Trustee → 4 stat cards (24 units, 94%, 5 open jobs, days to AGM), activity feed, "Needs attention" panel showing SLA breach + overdue levy alerts
  - Resident (unit 4B) → 3 stat cards (Paid ✓, 0 open requests, days to AGM), recent notices list

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/page.tsx
git commit -m "feat(overview): build scheme overview with role-aware dashboard"
```

---

## Task 17: Agent portfolio pages

**Files:**
- Modify: `app/agent/page.tsx`
- Modify: `app/agent/schemes/page.tsx`

- [ ] **Replace `app/agent/page.tsx`:**

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockPortfolio } from '@/lib/mock/scheme'

const HEALTH_STYLES = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-[#92400e]',
  poor: 'bg-red-bg text-red',
}

export default function AgentPortfolioPage() {
  const { user } = useMockAuth()

  const totalUnits = mockPortfolio.reduce((s, p) => s + p.unit_count, 0)
  const totalMaintenance = mockPortfolio.reduce((s, p) => s + p.open_maintenance_count, 0)
  const avgCollection = Math.round(
    mockPortfolio.reduce((s, p) => s + p.levy_collection_pct, 0) / mockPortfolio.length
  )

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Portfolio overview</h1>
      <p className="text-[14px] text-muted mb-8">
        {user?.orgName}. {mockPortfolio.length} schemes under management.
      </p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Active schemes',     value: String(mockPortfolio.length) },
          { label: 'Units managed',      value: String(totalUnits) },
          { label: 'Open maintenance',   value: String(totalMaintenance) },
          { label: 'Avg collection rate', value: `${avgCollection}%` },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Scheme list */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">All schemes</span>
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Scheme</span><span>Units</span><span>Collection</span><span>Maintenance</span><span>Health</span>
          </div>
          {mockPortfolio.map((scheme, i) => (
            <div key={scheme.id} className={`grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockPortfolio.length - 1 ? 'border-b border-border' : ''}`}>
              <div>
                <div className="font-semibold text-ink">{scheme.name}</div>
                <div className="text-[12px] text-muted">{scheme.address}</div>
              </div>
              <span className="text-muted">{scheme.unit_count}</span>
              <span className="font-medium text-ink">{scheme.levy_collection_pct}%</span>
              <span className="text-muted">{scheme.open_maintenance_count} open</span>
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Replace `app/agent/schemes/page.tsx`:**

```tsx
import { mockPortfolio } from '@/lib/mock/scheme'

const HEALTH_STYLES = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-[#92400e]',
  poor: 'bg-red-bg text-red',
}

export default function SchemesPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">All schemes</h1>
      <p className="text-[14px] text-muted mb-8">Schemes managed by your organisation.</p>
      <div className="flex flex-col gap-3">
        {mockPortfolio.map(scheme => (
          <div key={scheme.id} className="bg-white border border-border rounded-lg px-5 py-4 flex items-center justify-between">
            <div>
              <div className="text-[14px] font-semibold text-ink">{scheme.name}</div>
              <div className="text-[12px] text-muted mt-0.5">
                {scheme.unit_count} units · {scheme.address}
              </div>
              <div className="text-[12px] text-muted mt-0.5">
                {scheme.levy_collection_pct}% collected · {scheme.open_maintenance_count} open jobs
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
              </span>
              <a href={`/app/${scheme.id}`} className="text-[12px] text-accent font-medium">View →</a>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Verify in browser:**
  - Login as agent → `/agent`: 4 stat cards (3 schemes, 52 units, 23 open maintenance, 75% avg), scheme table with health badges
  - `/agent/schemes`: 3 scheme cards with collection rate, open jobs, health badges, View → links

- [ ] **Commit:**

```bash
git add app/agent/page.tsx app/agent/schemes/page.tsx
git commit -m "feat(agent): build portfolio overview and schemes list with mock data"
```

---

## Self-Review

**Spec coverage check:**
- ✓ All 8 mock files created with correct TypeScript interfaces
- ✓ All 7 scheme module pages built with role-aware views
- ✓ Agent portfolio + schemes pages built
- ✓ All 3 roles (agent, trustee, resident) handled in every page
- ✓ `canEdit` / `canUpload` / `canCompose` guards on action buttons
- ✓ Mock data references consistent unit IDs and identifiers across modules
- ✓ `lib/mock-auth.tsx` and all layout/sidebar/shell components untouched

**Type consistency check:**
- `LevyAccount.status` values: `'paid' | 'partial' | 'overdue' | 'pending'` — STATUS_STYLES keys match
- `MaintenanceRequest.status` values: `'open' | 'in_progress' | 'pending_approval' | 'resolved'` — STATUS_LABELS keys match
- `mockUnit4BPayments` correctly references `levy_account_id: 'la-004'` which is the Unit 4B account
- `submitted_by_unit` for resident requests uses unit identifier string (e.g. `'4B'`), matched against `user.unitIdentifier`

**Note on resident maintenance view:** Task 10 verifies that resident sees only their requests filtered by `submitted_by_unit === user.unitIdentifier`. Unit 4B resident sees mr-006 only (mr-001 has `submitted_by_unit: '2B'`, not '4B').
