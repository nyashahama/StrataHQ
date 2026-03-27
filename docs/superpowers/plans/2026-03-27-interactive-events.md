# Interactive Events Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up all buttons, modals, and state mutations across the 8 app module pages so the app is fully interactive before backend integration.

**Architecture:** Shared Modal component + Toast context (scheme layout) + per-page useState initialized from mock data. Mutations update local state only — no persistence across page navigations. This is intentional for the mock stage.

**Tech Stack:** React 19 useState/useContext, Next.js 14 Link, Tailwind CSS custom tokens

---

## Shared patterns

### Role-gating recap (unchanged)
- `canEdit = user?.role === 'agent'` — controls action buttons on levy, maintenance, members
- `canCompose = user?.role === 'agent'` — compose notice button
- `canUpload = user?.role === 'agent'` — upload document button

### Toast types
- `'success'` → green tokens (`green-bg`, `green`)
- `'info'` → accent tokens (`accent-bg`, `accent`)
- `'error'` → red tokens (`red-bg`, `red`)

### Modal pattern
Every modal: Escape key + backdrop click to close. Max-w-md centered. Rendered inline in the page component (not a portal — high z-index handles stacking).

---

## Task 1: Shared infrastructure — Modal + Toast

**Files:**
- Create: `components/Modal.tsx`
- Create: `lib/toast.tsx`
- Modify: `app/app/[schemeId]/layout.tsx`

- [ ] **Create `components/Modal.tsx`:**

```tsx
'use client'
import { useEffect, type ReactNode } from 'react'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
}

export default function Modal({ open, onClose, title, children }: ModalProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-ink/40" onClick={onClose} />
      <div className="relative bg-white rounded-xl shadow-lg w-full max-w-md">
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 className="text-[16px] font-semibold text-ink">{title}</h2>
          <button
            onClick={onClose}
            className="text-muted hover:text-ink text-[22px] leading-none w-8 h-8 flex items-center justify-center rounded hover:bg-page transition-colors"
            aria-label="Close"
          >
            ×
          </button>
        </div>
        <div className="px-6 py-5">{children}</div>
      </div>
    </div>
  )
}
```

- [ ] **Create `lib/toast.tsx`:**

```tsx
'use client'
import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

interface Toast {
  id: number
  message: string
  type: 'success' | 'info' | 'error'
}

interface ToastContextValue {
  addToast: (message: string, type?: Toast['type']) => void
}

const ToastContext = createContext<ToastContextValue>({ addToast: () => {} })

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback((message: string, type: Toast['type'] = 'success') => {
    const id = Date.now()
    setToasts(prev => [...prev, { id, message, type }])
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 3000)
  }, [])

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <div className="fixed bottom-5 right-5 z-[60] flex flex-col gap-2 pointer-events-none">
        {toasts.map(toast => (
          <div
            key={toast.id}
            className={`
              px-4 py-3 rounded-lg shadow-lg text-[13px] font-medium pointer-events-auto
              animate-in slide-in-from-right-4 duration-200
              ${toast.type === 'success' ? 'bg-green-bg border border-green/20 text-green' : ''}
              ${toast.type === 'info'    ? 'bg-accent-bg border border-accent/20 text-accent' : ''}
              ${toast.type === 'error'   ? 'bg-red-bg border border-red/20 text-red' : ''}
            `}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  return useContext(ToastContext)
}
```

NOTE: The `animate-in slide-in-from-right-4` classes require `tailwindcss-animate`. If not installed, remove those two classes — the toast will still work without animation.

- [ ] **Read `app/app/[schemeId]/layout.tsx`** to understand its current structure before modifying.

- [ ] **Modify `app/app/[schemeId]/layout.tsx`** — wrap children with ToastProvider:

Add to imports:
```tsx
import { ToastProvider } from '@/lib/toast'
```

Wrap the AppShell return with ToastProvider:
```tsx
return (
  <ToastProvider>
    <AppShell
      sidebar={
        <Sidebar
          role={sidebarRole}
          headerLabel={headerLabel}
          schemeId={schemeId}
        />
      }
    >
      {children}
    </AppShell>
  </ToastProvider>
)
```

- [ ] **Check if `tailwindcss-animate` is installed** — run `cat package.json | grep animate`. If not present, remove `animate-in slide-in-from-right-4 duration-200` from the toast div in `lib/toast.tsx`.

- [ ] **Commit:**

```bash
git add components/Modal.tsx lib/toast.tsx app/app/\[schemeId\]/layout.tsx
git commit -m "feat(ui): add Modal component and Toast system"
```

---

## Task 2: Overview page — "Read →" navigation

**Files:**
- Modify: `app/app/[schemeId]/page.tsx`

- [ ] **Read `app/app/[schemeId]/page.tsx`** to confirm current structure.

- [ ] **Update the resident notices section** — replace the `<span>Read →</span>` with a `<Link>`:

Add to imports at top of file:
```tsx
import Link from 'next/link'
```

Find the notices map in the resident view (the `<span className="text-[12px] text-accent font-medium">Read →</span>` at the end of each notice row) and replace the wrapping div with a Link:

Before:
```tsx
<div key={n.id} className="bg-white border border-border rounded-lg px-5 py-3 flex items-center justify-between">
  <div>
    <div className="text-[13px] font-medium text-ink">{n.title}</div>
    <div className="text-[11px] text-muted mt-0.5">
      {new Date(n.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
    </div>
  </div>
  <span className="text-[12px] text-accent font-medium">Read →</span>
</div>
```

After:
```tsx
<Link
  key={n.id}
  href={`/app/${mockScheme.id}/communications`}
  className="bg-white border border-border rounded-lg px-5 py-3 flex items-center justify-between hover:bg-page transition-colors"
>
  <div>
    <div className="text-[13px] font-medium text-ink">{n.title}</div>
    <div className="text-[11px] text-muted mt-0.5">
      {new Date(n.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
    </div>
  </div>
  <span className="text-[12px] text-accent font-medium">Read →</span>
</Link>
```

Note: remove `key` from the Link — it should be on the outermost element in the map. The `key` is already on the Link via JSX, which is correct.

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/page.tsx
git commit -m "feat(overview): wire Read links to communications page"
```

---

## Task 3: Levy page — Remind + Download actions

**Files:**
- Modify: `app/app/[schemeId]/levy/page.tsx`

Pattern: no modal needed — both actions are instant toasts. `useToast` from `lib/toast`.

- [ ] **Read `app/app/[schemeId]/levy/page.tsx`** first.

- [ ] **Replace the entire file** with this updated version:

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockLevyRoll, mockLevyPeriod, mockCollectionTrend, mockUnit4BPayments } from '@/lib/mock/levy'
import { useToast } from '@/lib/toast'

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
  const { addToast } = useToast()

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

        <button
          onClick={() => addToast('Statement download started — check your downloads folder.', 'info')}
          className="text-[12px] text-accent font-medium border border-accent rounded px-4 py-2 hover:bg-accent-dim transition-colors"
        >
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
                  <button
                    onClick={() => addToast(`Reminder sent to ${account.owner_name}`, 'info')}
                    className="text-[11px] text-accent font-medium hover:underline"
                  >
                    Remind
                  </button>
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

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/levy/page.tsx
git commit -m "feat(levy): wire Remind and Download statement actions"
```

---

## Task 4: Maintenance page — New job modal + Approve + Submit request

**Files:**
- Modify: `app/app/[schemeId]/maintenance/page.tsx`

This page needs `useState` for local job list, plus two modals (one for agent "+ New job", one for resident "+ Submit request").

- [ ] **Read `app/app/[schemeId]/maintenance/page.tsx`** first.

- [ ] **Replace the entire file:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMaintenanceRequests, type MaintenanceRequest } from '@/lib/mock/maintenance'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

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
  const now = new Date('2025-10-16T12:00:00Z').getTime()
  const hoursElapsed = (now - created) / (1000 * 60 * 60)
  return hoursElapsed > req.sla_hours
}

const CATEGORIES = ['plumbing', 'electrical', 'structural', 'garden', 'pool', 'other'] as const

export default function MaintenancePage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [jobs, setJobs] = useState<MaintenanceRequest[]>([...mockMaintenanceRequests])
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ title: '', category: 'other', description: '' })

  function handleApprove(id: string) {
    setJobs(prev => prev.map(j => j.id === id ? { ...j, status: 'in_progress' as const } : j))
    addToast('Job approved and moved to In Progress', 'success')
  }

  function handleSubmit() {
    if (!form.title.trim()) return
    const newJob: MaintenanceRequest = {
      id: `mr-${Date.now()}`,
      scheme_id: 'scheme-001',
      unit_id: user?.role === 'resident' ? 'unit-4b' : null,
      title: form.title.trim(),
      description: form.description.trim(),
      category: form.category as MaintenanceRequest['category'],
      status: user?.role === 'resident' ? 'pending_approval' : 'open',
      contractor_name: null,
      contractor_phone: null,
      sla_hours: 72,
      created_at: new Date().toISOString(),
      resolved_at: null,
      submitted_by_unit: user?.role === 'resident' ? (user.unitIdentifier ?? null) : null,
    }
    setJobs(prev => [newJob, ...prev])
    setShowModal(false)
    setForm({ title: '', category: 'other', description: '' })
    addToast(user?.role === 'resident' ? 'Request submitted for approval' : 'Job created', 'success')
  }

  const modalTitle = user?.role === 'resident' ? 'Submit maintenance request' : 'New maintenance job'

  // Resident: show only their unit's requests
  if (user?.role === 'resident') {
    const myRequests = jobs.filter(r => r.submitted_by_unit === user.unitIdentifier)
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
        <p className="text-[14px] text-muted mb-8">Your maintenance requests for Unit {user.unitIdentifier}.</p>

        <div className="flex items-center justify-between mb-6">
          <span className="text-[13px] text-muted">{myRequests.length} request{myRequests.length !== 1 ? 's' : ''}</span>
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors"
          >
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

        <Modal open={showModal} onClose={() => setShowModal(false)} title={modalTitle}>
          <JobForm form={form} setForm={setForm} onSubmit={handleSubmit} onCancel={() => setShowModal(false)} />
        </Modal>
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const open = jobs.filter(r => r.status !== 'resolved')
  const breached = open.filter(isSlaBreached)
  const pendingApproval = open.filter(r => r.status === 'pending_approval')
  const resolvedCount = jobs.filter(r => r.status === 'resolved').length

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
      <p className="text-[14px] text-muted mb-8">Log and track maintenance requests.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Open jobs',           value: String(open.length) },
          { label: 'SLA breaches',        value: String(breached.length) },
          { label: 'Pending approval',    value: String(pendingApproval.length) },
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
            <button
              onClick={() => setShowModal(true)}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors"
            >
              + New job
            </button>
          )}
        </div>
        <div className="px-5 py-3 flex flex-col gap-0">
          {jobs.map((req, i) => {
            const breachedSla = isSlaBreached(req)
            return (
              <div key={req.id} className={`flex gap-3 items-start py-4 ${i < jobs.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[15px] mt-0.5">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold text-ink mb-[2px]">{req.title}</div>
                  <div className="text-[11px] text-muted">
                    {req.contractor_name ?? 'No contractor assigned'}
                    {req.submitted_by_unit && ` · Unit ${req.submitted_by_unit}`}
                  </div>
                  {breachedSla && (
                    <div className="text-[11px] text-red font-medium mt-[2px]">⚠ SLA breached</div>
                  )}
                  {!breachedSla && req.status !== 'resolved' && (
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
                    <button
                      onClick={() => handleApprove(req.id)}
                      className="text-[11px] text-accent font-medium hover:underline"
                    >
                      Approve
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <Modal open={showModal} onClose={() => setShowModal(false)} title={modalTitle}>
        <JobForm form={form} setForm={setForm} onSubmit={handleSubmit} onCancel={() => setShowModal(false)} />
      </Modal>
    </div>
  )
}

function JobForm({
  form,
  setForm,
  onSubmit,
  onCancel,
}: {
  form: { title: string; category: string; description: string }
  setForm: React.Dispatch<React.SetStateAction<{ title: string; category: string; description: string }>>
  onSubmit: () => void
  onCancel: () => void
}) {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Title *</label>
        <input
          type="text"
          value={form.title}
          onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
          placeholder="e.g. Leaking tap in Unit 3A"
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
        />
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
        <select
          value={form.category}
          onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
        >
          {['plumbing', 'electrical', 'structural', 'garden', 'pool', 'other'].map(c => (
            <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>
          ))}
        </select>
      </div>
      <div>
        <label className="text-[12px] font-semibold text-ink block mb-1">Description</label>
        <textarea
          value={form.description}
          onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
          placeholder="Brief description of the issue"
          rows={3}
          className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent resize-none"
        />
      </div>
      <div className="flex gap-2 pt-1">
        <button
          onClick={onSubmit}
          disabled={!form.title.trim()}
          className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Submit
        </button>
        <button
          onClick={onCancel}
          className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/maintenance/page.tsx
git commit -m "feat(maintenance): add New job modal, Approve action, and Submit request modal"
```

---

## Task 5: AGM page — Voting

**Files:**
- Modify: `app/app/[schemeId]/agm/page.tsx`

Voting: resident can vote once per resolution. Track voted IDs in a Set. Update vote counts in local state.

- [ ] **Read `app/app/[schemeId]/agm/page.tsx`** first.

- [ ] **Replace the entire file:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockAgmMeeting, mockAgmResolutions, mockUpcomingAgm, type AgmResolution } from '@/lib/mock/agm'
import { useToast } from '@/lib/toast'

export default function AgmVotingPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [resolutions, setResolutions] = useState<AgmResolution[]>([...mockAgmResolutions])
  const [voted, setVoted] = useState<Set<string>>(new Set())

  const meeting = mockAgmMeeting
  const upcoming = mockUpcomingAgm
  const quorumPct = Math.round((meeting.quorum_present / (meeting.quorum_required * 2)) * 100)

  function castVote(resId: string, inFavour: boolean) {
    if (voted.has(resId)) return
    setResolutions(prev => prev.map(r => {
      if (r.id !== resId) return r
      const newFor = inFavour ? r.votes_for + 1 : r.votes_for
      const newAgainst = inFavour ? r.votes_against : r.votes_against + 1
      return { ...r, votes_for: newFor, votes_against: newAgainst }
    }))
    setVoted(prev => new Set([...prev, resId]))
    addToast(inFavour ? 'Vote in favour recorded' : 'Vote against recorded', 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › AGM & Voting</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">AGM & Voting</h1>
      <p className="text-[14px] text-muted mb-8">Annual general meetings and trustee resolutions.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Last AGM',  value: new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Next AGM',  value: new Date(upcoming.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Quorum',    value: `${meeting.quorum_present}/${meeting.quorum_required * 2}` },
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
          {resolutions.map((res, i) => {
            const forPct = Math.round((res.votes_for / res.total_eligible) * 100)
            const hasVoted = voted.has(res.id)
            const canVote = user?.role === 'resident' && res.status === 'open'
            return (
              <div key={res.id} className={`${i < resolutions.length - 1 ? 'pb-4 border-b border-border' : ''}`}>
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div>
                    <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {i + 1} OF {resolutions.length}</div>
                    <div className="text-[13px] font-semibold text-ink">{res.title}</div>
                    <div className="text-[12px] text-muted mt-1">{res.description}</div>
                  </div>
                  <span className={`flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full ${res.status === 'passed' ? 'bg-green-bg text-green' : res.status === 'failed' ? 'bg-red-bg text-red' : 'bg-yellowbg text-[#92400e]'}`}>
                    {res.status === 'open' ? 'Voting open' : res.status.charAt(0).toUpperCase() + res.status.slice(1)}
                  </span>
                </div>
                <div className="flex justify-between text-[11px] text-muted mb-1">
                  <span>In favour · {res.votes_for} votes ({forPct}%)</span>
                  <span>Against · {res.votes_against}</span>
                </div>
                <div className="h-[6px] bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-accent rounded-full transition-all duration-300" style={{ width: `${forPct}%` }} />
                </div>
                {canVote && (
                  <div className="flex gap-2 mt-3">
                    {hasVoted ? (
                      <span className="text-[12px] text-green font-medium">✓ Vote recorded</span>
                    ) : (
                      <>
                        <button
                          onClick={() => castVote(res.id, true)}
                          className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:bg-[#245a96] transition-colors"
                        >
                          Vote in favour
                        </button>
                        <button
                          onClick={() => castVote(res.id, false)}
                          className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors"
                        >
                          Vote against
                        </button>
                      </>
                    )}
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

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/agm/page.tsx
git commit -m "feat(agm): wire voting buttons with local state and toast confirmation"
```

---

## Task 6: Members page — Invite member modal

**Files:**
- Modify: `app/app/[schemeId]/members/page.tsx`

- [ ] **Read `app/app/[schemeId]/members/page.tsx`** first.

- [ ] **Replace the entire file:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMembers, type Member } from '@/lib/mock/members'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  owner:    'bg-[#f0efe9] text-muted',
  resident: 'bg-green-bg text-green',
}

export default function MembersPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [members, setMembers] = useState<Member[]>([...mockMembers])
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ name: '', email: '', unit: '', role: 'owner' as 'owner' | 'resident' })

  function handleInvite() {
    if (!form.name.trim() || !form.email.trim() || !form.unit.trim()) return
    const newMember: Member = {
      id: `member-${Date.now()}`,
      scheme_id: 'scheme-001',
      unit_id: `unit-${form.unit.toLowerCase()}`,
      unit_identifier: form.unit.toUpperCase(),
      name: form.name.trim(),
      role: form.role,
      email: form.email.trim(),
      phone: null,
      is_trustee_committee: false,
    }
    setMembers(prev => [...prev, newMember])
    setShowModal(false)
    setForm({ name: '', email: '', unit: '', role: 'owner' })
    addToast(`Invite sent to ${form.email.trim()}`, 'success')
  }

  // Resident: trustees only
  if (user?.role === 'resident') {
    const trustees = members.filter(m => m.is_trustee_committee)
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
  const trustees = members.filter(m => m.is_trustee_committee)
  const owners = members.filter(m => !m.is_trustee_committee)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Members</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Members</h1>
      <p className="text-[14px] text-muted mb-8">Owners, trustees, and contact information.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Total members', value: String(members.length) },
          { label: 'Trustees',      value: String(trustees.length) },
          { label: 'Owners',        value: String(owners.length) },
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
            <button
              onClick={() => setShowModal(true)}
              className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors"
            >
              + Invite member
            </button>
          )}
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[auto_1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Unit</span><span>Name</span><span>Contact</span><span>Role</span>
          </div>
          {members.map((m, i) => (
            <div key={m.id} className={`grid grid-cols-[auto_1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < members.length - 1 ? 'border-b border-border' : ''}`}>
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

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Invite member">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name *</label>
            <input
              type="text"
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. Nkosi, A."
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email *</label>
            <input
              type="email"
              value={form.email}
              onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
              placeholder="e.g. nkosi@email.co.za"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Unit *</label>
              <input
                type="text"
                value={form.unit}
                onChange={e => setForm(f => ({ ...f, unit: e.target.value }))}
                placeholder="e.g. 9A"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Role</label>
              <select
                value={form.role}
                onChange={e => setForm(f => ({ ...f, role: e.target.value as 'owner' | 'resident' }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
              >
                <option value="owner">Owner</option>
                <option value="resident">Resident</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleInvite}
              disabled={!form.name.trim() || !form.email.trim() || !form.unit.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Send invite
            </button>
            <button
              onClick={() => setShowModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
```

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/members/page.tsx
git commit -m "feat(members): add Invite member modal with local state"
```

---

## Task 7: Communications page — Compose notice modal

**Files:**
- Modify: `app/app/[schemeId]/communications/page.tsx`

The expand/collapse already uses `useState`. Add compose modal that prepends to local notices list.

- [ ] **Read `app/app/[schemeId]/communications/page.tsx`** first.

- [ ] **Replace the entire file:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockNotices, type Notice } from '@/lib/mock/communications'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

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
  const { addToast } = useToast()

  const [notices, setNotices] = useState<Notice[]>([...mockNotices])
  const [expanded, setExpanded] = useState<string | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ title: '', body: '', type: 'general' as Notice['type'] })

  const canCompose = user?.role === 'agent'

  function handleCompose() {
    if (!form.title.trim() || !form.body.trim()) return
    const newNotice: Notice = {
      id: `notice-${Date.now()}`,
      scheme_id: 'scheme-001',
      title: form.title.trim(),
      body: form.body.trim(),
      sent_at: new Date().toISOString(),
      sent_by_name: user?.orgName ?? 'Managing Agent',
      type: form.type,
    }
    setNotices(prev => [newNotice, ...prev])
    setExpanded(newNotice.id)
    setShowModal(false)
    setForm({ title: '', body: '', type: 'general' })
    addToast('Notice sent to all residents', 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Communications</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Communications</h1>
      <p className="text-[14px] text-muted mb-8">Notices, announcements, and correspondence.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{notices.length} notices</span>
        {canCompose && (
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors"
          >
            + Compose notice
          </button>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {notices.map(notice => (
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

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Compose notice">
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Type</label>
              <select
                value={form.type}
                onChange={e => setForm(f => ({ ...f, type: e.target.value as Notice['type'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
              >
                <option value="general">General</option>
                <option value="urgent">Urgent</option>
                <option value="agm">AGM</option>
                <option value="levy">Levy</option>
              </select>
            </div>
            <div />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Subject *</label>
            <input
              type="text"
              value={form.title}
              onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
              placeholder="Notice subject"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Body *</label>
            <textarea
              value={form.body}
              onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
              placeholder="Write your notice here…"
              rows={5}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent resize-none"
            />
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleCompose}
              disabled={!form.title.trim() || !form.body.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Send to all residents
            </button>
            <button
              onClick={() => setShowModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
```

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/communications/page.tsx
git commit -m "feat(communications): add Compose notice modal with local state"
```

---

## Task 8: Documents page — Upload modal + Download toast

**Files:**
- Modify: `app/app/[schemeId]/documents/page.tsx`

- [ ] **Read `app/app/[schemeId]/documents/page.tsx`** first.

- [ ] **Replace the entire file:**

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockDocuments, type SchemeDocument } from '@/lib/mock/documents'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

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

function groupByCategory(docs: SchemeDocument[]): Record<string, SchemeDocument[]> {
  const order = Object.values(CATEGORY_LABELS)
  const grouped = docs.reduce((acc, doc) => {
    const key = CATEGORY_LABELS[doc.category]
    if (!acc[key]) acc[key] = []
    acc[key].push(doc)
    return acc
  }, {} as Record<string, SchemeDocument[]>)
  return Object.fromEntries(order.filter(k => grouped[k]).map(k => [k, grouped[k]]))
}

export default function DocumentsPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [documents, setDocuments] = useState<SchemeDocument[]>([...mockDocuments])
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ name: '', category: 'other' as SchemeDocument['category'], file_type: 'pdf' as SchemeDocument['file_type'] })

  const canUpload = user?.role === 'agent'
  const grouped = groupByCategory(documents)

  function handleUpload() {
    if (!form.name.trim()) return
    const newDoc: SchemeDocument = {
      id: `doc-${Date.now()}`,
      scheme_id: 'scheme-001',
      name: form.name.trim(),
      file_type: form.file_type,
      category: form.category,
      uploaded_at: new Date().toISOString(),
      uploaded_by_name: user?.orgName ?? 'Managing Agent',
      size_bytes: Math.floor(Math.random() * 500000) + 50000,
    }
    setDocuments(prev => [newDoc, ...prev])
    setShowModal(false)
    setForm({ name: '', category: 'other', file_type: 'pdf' })
    addToast(`"${newDoc.name}" uploaded successfully`, 'success')
  }

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Documents</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Documents</h1>
      <p className="text-[14px] text-muted mb-8">Scheme rules, minutes, and shared files.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{documents.length} documents</span>
        {canUpload && (
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors"
          >
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
                  <button
                    onClick={() => addToast(`Downloading "${doc.name}"…`, 'info')}
                    className="text-[12px] text-accent font-medium hover:underline flex-shrink-0"
                  >
                    Download
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Upload document">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Document name *</label>
            <input
              type="text"
              value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              placeholder="e.g. AGM Minutes November 2025"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Category</label>
              <select
                value={form.category}
                onChange={e => setForm(f => ({ ...f, category: e.target.value as SchemeDocument['category'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
              >
                {(Object.entries(CATEGORY_LABELS) as [SchemeDocument['category'], string][]).map(([val, label]) => (
                  <option key={val} value={val}>{label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">File type</label>
              <select
                value={form.file_type}
                onChange={e => setForm(f => ({ ...f, file_type: e.target.value as SchemeDocument['file_type'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
              >
                {['pdf', 'docx', 'xlsx', 'jpg', 'png'].map(t => (
                  <option key={t} value={t}>{t.toUpperCase()}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="border-2 border-dashed border-border rounded-lg px-6 py-8 text-center text-[13px] text-muted">
            File picker coming when backend is connected
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleUpload}
              disabled={!form.name.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:bg-[#245a96] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Upload
            </button>
            <button
              onClick={() => setShowModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
```

- [ ] **Commit:**

```bash
git add app/app/\[schemeId\]/documents/page.tsx
git commit -m "feat(documents): add Upload document modal and Download toast"
```

---

## Self-Review

**Spec coverage:**
- ✓ Modal component: Escape, backdrop click, accessible close button
- ✓ Toast system: 3-second auto-dismiss, success/info/error variants, ToastProvider in scheme layout
- ✓ Levy: Remind toast, Download statement toast
- ✓ Maintenance: New job modal, Approve state update, Submit request modal (resident)
- ✓ AGM: Vote buttons, one-vote-per-resolution guard, live vote count update
- ✓ Members: Invite member modal, appends to roster, stat cards update
- ✓ Communications: Compose notice modal, prepends to list, expands new notice
- ✓ Documents: Upload document modal, Download toast
- ✓ Overview: "Read →" wired to Link → /communications

**Type consistency:**
- `MaintenanceRequest.status` literal: `'pending_approval' as const` used correctly
- `Notice.type`, `SchemeDocument['category']`, `SchemeDocument['file_type']` all cast correctly on new items
- `Member.role` cast via `as 'owner' | 'resident'` on form submit
- `AgmResolution` spread pattern preserves all fields

**toast.tsx note:** The `animate-in slide-in-from-right-4` classes are from `tailwindcss-animate`. If not installed, they are silently ignored by Tailwind and the toast still renders — just without the entry animation.
