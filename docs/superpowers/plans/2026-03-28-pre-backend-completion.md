# Pre-Backend Completion Plan

**Goal:** Close all 24 gaps identified in the pre-backend audit so the app is fully functional before backend integration.

**Approach:** Group related items into 8 focused tasks. Each task is independently committable.

---

## Task 1: Fix sidebar navigation — `<a>` → `<Link>`

**Files:**
- Modify: `components/Sidebar.tsx`
- Modify: `app/agent/schemes/page.tsx`

The `NavLink` component uses `<a href>` causing full page reloads that destroy local React state. Replace with Next.js `<Link>`.

- [ ] In `components/Sidebar.tsx`, add `import Link from 'next/link'` at the top
- [ ] In the `NavLink` function, replace `<a href={item.href} className={...}>` with `<Link href={item.href} className={...}>` and close with `</Link>`. Keep all className logic identical.
- [ ] In `app/agent/schemes/page.tsx`, replace `<a href={...} className="text-[12px] text-accent font-medium">View →</a>` with `<Link href={...} className="text-[12px] text-accent font-medium">View →</Link>` and add `import Link from 'next/link'`
- [ ] Commit: `fix(nav): replace a href with Next.js Link to prevent full page reloads`

---

## Task 2: Add logout button to sidebar

**Files:**
- Modify: `components/Sidebar.tsx`

The sidebar has no logout button. Users cannot sign out without clearing localStorage.

- [ ] In `components/Sidebar.tsx`, add `import { useMockAuth } from '@/lib/mock-auth'`
- [ ] Inside `Sidebar` component, call `const { logout } = useMockAuth()`
- [ ] After the bottom nav item (the Settings/Profile link), add a logout button in the bottom section:

```tsx
<div className="border-t border-border py-2 flex-shrink-0">
  <NavLink item={bottomItem} pathname={pathname} />
  <button
    onClick={() => { logout(); router.push('/auth/login') }}
    className="flex items-center gap-2 px-3 py-[7px] text-[12px] text-muted hover:text-ink hover:bg-[#f0efe9] w-full transition-colors"
  >
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M5 2H3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h2M9 10l3-3-3-3M12 7H5" />
    </svg>
    Log out
  </button>
</div>
```

- [ ] Commit: `feat(sidebar): add logout button`

---

## Task 3: Add missing sidebar links (trustee Members, resident AGM + Financials)

**Files:**
- Modify: `components/Sidebar.tsx`

- [ ] In the `trustee` nav items array, add Members after Financials:
  ```tsx
  { icon: <UsersIcon />, label: 'Members', href: `${base}/members` },
  ```
- [ ] In the `resident` nav items array, add AGM & Voting and Financials:
  ```tsx
  { icon: <VoteIcon />, label: 'AGM & Voting', href: `${base}/agm` },
  { icon: <ChartIcon />, label: 'Financials', href: `${base}/financials` },
  ```
  Insert AGM after Maintenance, Financials after Documents.
- [ ] Commit: `fix(sidebar): add Members link for trustee, AGM and Financials links for resident`

---

## Task 4: Fix hardcoded timestamps

**Files:**
- Modify: `app/app/[schemeId]/page.tsx`
- Modify: `app/app/[schemeId]/maintenance/page.tsx`

- [ ] In `app/app/[schemeId]/page.tsx`:
  - Change `daysUntil` to use real current date: `const now = new Date()` instead of `new Date('2025-10-16')`
  - Change the SLA breach calculation in the agent/trustee view: `const now = new Date().getTime()` instead of `new Date('2025-10-16T12:00:00Z').getTime()`

- [ ] In `app/app/[schemeId]/maintenance/page.tsx`:
  - In `isSlaBreached`, change `const now = new Date('2025-10-16T12:00:00Z').getTime()` to `const now = new Date().getTime()`

- [ ] Commit: `fix(dates): replace hardcoded mock timestamps with real Date.now()`

---

## Task 5: Create the three missing pages (Settings x2 + Profile)

**Files:**
- Create: `app/agent/settings/page.tsx`
- Create: `app/app/[schemeId]/settings/page.tsx`
- Create: `app/app/[schemeId]/profile/page.tsx`

### `app/agent/settings/page.tsx`

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'

export default function AgentSettingsPage() {
  const { user } = useMockAuth()

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Settings</h1>
      <p className="text-[14px] text-muted mb-8">Organisation and account settings.</p>

      {/* Organisation */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Organisation</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Organisation name</label>
            <input
              type="text"
              defaultValue={user?.orgName}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Contact email</label>
            <input
              type="email"
              defaultValue="admin@acme.co.za"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              defaultValue="+27 21 555 0100"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
              Save changes
            </button>
          </div>
        </div>
      </div>

      {/* Account */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Account</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Password</label>
            <button className="text-[12px] text-accent font-medium hover:underline">
              Change password →
            </button>
          </div>
          <div className="pt-2 border-t border-border">
            <p className="text-[12px] text-muted mb-2">Danger zone</p>
            <button className="text-[12px] font-medium text-red hover:underline">
              Delete account
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
```

### `app/app/[schemeId]/settings/page.tsx`

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockScheme } from '@/lib/mock/scheme'

export default function SchemeSettingsPage() {
  const { user } = useMockAuth()

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Settings</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Scheme Settings</h1>
      <p className="text-[14px] text-muted mb-8">Manage scheme details and configuration.</p>

      {/* Scheme details */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Scheme details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Scheme name</label>
            <input
              type="text"
              defaultValue={mockScheme.name}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Physical address</label>
            <input
              type="text"
              defaultValue={mockScheme.address}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Total units</label>
            <input
              type="number"
              defaultValue={mockScheme.unit_count}
              disabled={user?.role !== 'agent'}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
            />
          </div>
          {user?.role === 'agent' && (
            <div>
              <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
                Save changes
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Levy configuration */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Levy configuration</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Base levy (ZAR)</label>
              <input
                type="text"
                defaultValue="2 450.00"
                disabled={user?.role !== 'agent'}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Levy period</label>
              <select
                disabled={user?.role !== 'agent'}
                defaultValue="Monthly"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent disabled:bg-page disabled:text-muted"
              >
                <option>Monthly</option>
                <option>Quarterly</option>
              </select>
            </div>
          </div>
          {user?.role === 'agent' && (
            <div>
              <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
                Update levy settings
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Notifications */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Notifications</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-3">
          {[
            { label: 'Overdue levy reminders', defaultChecked: true },
            { label: 'Maintenance SLA breach alerts', defaultChecked: true },
            { label: 'New maintenance requests', defaultChecked: false },
            { label: 'AGM reminders (30 days before)', defaultChecked: true },
          ].map(item => (
            <label key={item.label} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                defaultChecked={item.defaultChecked}
                disabled={user?.role !== 'agent'}
                className="w-4 h-4 accent-accent"
              />
              <span className="text-[13px] text-ink">{item.label}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  )
}
```

### `app/app/[schemeId]/profile/page.tsx`

```tsx
'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMembers } from '@/lib/mock/members'
import { useToast } from '@/lib/toast'

export default function ResidentProfilePage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const myMember = mockMembers.find(m => m.unit_identifier === user?.unitIdentifier)

  return (
    <div className="px-8 py-8 max-w-[700px]">
      <p className="text-[12px] text-muted mb-4">My Profile</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">My Profile</h1>
      <p className="text-[14px] text-muted mb-8">Your contact details and unit information.</p>

      {/* Unit info */}
      <div className="bg-white border border-border rounded-lg px-5 py-4 mb-6 flex items-center gap-4">
        <div className="w-12 h-12 rounded-lg bg-accent-bg flex items-center justify-center flex-shrink-0">
          <span className="text-[16px] font-semibold text-accent">{user?.unitIdentifier}</span>
        </div>
        <div>
          <div className="text-[14px] font-semibold text-ink">Unit {user?.unitIdentifier}</div>
          <div className="text-[12px] text-muted">{user?.schemeName}</div>
        </div>
        <span className="ml-auto text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">
          Owner
        </span>
      </div>

      {/* Contact details */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Contact details</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Full name</label>
            <input
              type="text"
              defaultValue={myMember?.name ?? ''}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Email</label>
            <input
              type="email"
              defaultValue={myMember?.email ?? ''}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Phone</label>
            <input
              type="tel"
              defaultValue={myMember?.phone ?? ''}
              placeholder="+27 82 000 0000"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-white focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <button
              onClick={() => addToast('Profile updated', 'success')}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors"
            >
              Save changes
            </button>
          </div>
        </div>
      </div>

      {/* Account */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">Account</span>
        </div>
        <div className="px-5 py-4 flex flex-col gap-3">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Password</label>
            <button className="text-[12px] text-accent font-medium hover:underline">
              Change password →
            </button>
          </div>
          <div className="text-[12px] text-muted pt-2 border-t border-border">
            Account managed by your body corporate managing agent.
          </div>
        </div>
      </div>
    </div>
  )
}
```

- [ ] Commit: `feat(pages): add Agent Settings, Scheme Settings, and Resident Profile pages`

---

## Task 6: Wire up `/agent/invitations` with mock data and actions

**Files:**
- Modify: `app/agent/invitations/page.tsx`

The page shows a static empty state. It should show mock pending invitations with Accept/Decline actions.

- [ ] Replace with a fully functional invitations page:

```tsx
'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { useToast } from '@/lib/toast'

interface Invitation {
  id: string
  name: string
  email: string
  role: 'trustee' | 'resident'
  scheme_name: string
  unit_identifier: string
  invited_at: string
}

const mockInvitations: Invitation[] = [
  { id: 'inv-001', name: 'Nkosi, A.',     email: 'a.nkosi@gmail.com',       role: 'resident', scheme_name: 'Sunridge Heights', unit_identifier: '9A',  invited_at: '2025-10-14T09:00:00Z' },
  { id: 'inv-002', name: 'Botha, C.',     email: 'c.botha@outlook.com',      role: 'trustee',  scheme_name: 'Sunridge Heights', unit_identifier: '10B', invited_at: '2025-10-13T11:30:00Z' },
  { id: 'inv-003', name: 'Fredericks, P.',email: 'p.fredericks@email.co.za', role: 'resident', scheme_name: 'Sunridge Heights', unit_identifier: '11C', invited_at: '2025-10-12T14:00:00Z' },
]

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  resident: 'bg-green-bg text-green',
}

export default function InvitationsPage() {
  useMockAuth()
  const { addToast } = useToast()
  const [invitations, setInvitations] = useState<Invitation[]>(mockInvitations)

  function handleAction(id: string, action: 'resend' | 'revoke') {
    if (action === 'revoke') {
      setInvitations(prev => prev.filter(i => i.id !== id))
      addToast('Invitation revoked', 'info')
    } else {
      addToast('Invitation resent', 'success')
    }
  }

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Portfolio › Invitations</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Invitations</h1>
      <p className="text-[14px] text-muted mb-8">Pending trustee and resident invitations.</p>

      {invitations.length === 0 ? (
        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No pending invitations
        </div>
      ) : (
        <div className="bg-white border border-border rounded-lg overflow-hidden">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">Pending</span>
            <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-[#92400e]">{invitations.length} pending</span>
          </div>
          <div className="px-5">
            <div className="grid grid-cols-[1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
              <span>Invitee</span><span>Unit</span><span>Role</span><span>Actions</span>
            </div>
            {invitations.map((inv, i) => (
              <div key={inv.id} className={`grid grid-cols-[1fr_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < invitations.length - 1 ? 'border-b border-border' : ''}`}>
                <div>
                  <div className="font-medium text-ink">{inv.name}</div>
                  <div className="text-[12px] text-muted">{inv.email}</div>
                </div>
                <span className="text-muted text-[12px]">{inv.unit_identifier}</span>
                <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[inv.role]}`}>
                  {inv.role.charAt(0).toUpperCase() + inv.role.slice(1)}
                </span>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => handleAction(inv.id, 'resend')}
                    className="text-[11px] text-accent font-medium hover:underline"
                  >
                    Resend
                  </button>
                  <button
                    onClick={() => handleAction(inv.id, 'revoke')}
                    className="text-[11px] text-red font-medium hover:underline"
                  >
                    Revoke
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
```

- [ ] Commit: `feat(invitations): wire pending invitations list with resend and revoke actions`

---

## Task 7: Add "Schedule AGM" button + wire Financials export action

**Files:**
- Modify: `app/app/[schemeId]/agm/page.tsx`
- Modify: `app/app/[schemeId]/financials/page.tsx`

### AGM: Schedule AGM button (agent only)

In the agent/trustee view, add a "+ Schedule AGM" button that opens a Modal with date/venue fields. On submit, update `upcoming` in local state and show a toast.

Add to the AGM page:
- `useState` for `upcomingMeeting` (initialized from `mockUpcomingAgm`)
- `useState` for `showScheduleModal` and form state `{ date: '', venue: '' }`
- A "+ Schedule AGM" button (agent only) in the header area
- A Modal with date input and venue text input
- On submit: update `upcomingMeeting`, close modal, toast "AGM scheduled"

### Financials: Export button (agent/trustee)

In the agent/trustee view header, add an "Export CSV" button that shows a toast "Budget export downloaded".

- [ ] Commit: `feat(agm): add Schedule AGM modal for agent; feat(financials): add Export CSV action`

---

## Task 8: Add not-found, loading, and fix "Forgot password" link

**Files:**
- Create: `app/not-found.tsx`
- Create: `app/app/[schemeId]/loading.tsx`
- Create: `app/agent/loading.tsx`
- Modify: `app/auth/login/page.tsx` (Forgot password → `/auth/forgot-password`)
- Create: `app/auth/forgot-password/page.tsx`

### `app/not-found.tsx`

```tsx
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'

export default function NotFound() {
  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="text-center">
        <div className="flex items-center justify-center gap-2 mb-10">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">StrataHQ</span>
        </div>
        <p className="text-[11px] font-semibold text-muted uppercase tracking-widest mb-3">404</p>
        <h1 className="font-serif text-[32px] font-semibold text-ink mb-3">Page not found</h1>
        <p className="text-[14px] text-muted mb-8">This page doesn't exist or you don't have access.</p>
        <Link
          href="/auth/login"
          className="text-[13px] font-semibold text-accent hover:underline"
        >
          Back to login →
        </Link>
      </div>
    </main>
  )
}
```

### `app/app/[schemeId]/loading.tsx`

```tsx
export default function Loading() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <div className="h-4 w-32 bg-border rounded animate-pulse mb-6" />
      <div className="h-8 w-64 bg-border rounded animate-pulse mb-2" />
      <div className="h-4 w-80 bg-border rounded animate-pulse mb-8" />
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="h-7 w-16 bg-border rounded animate-pulse mb-2" />
            <div className="h-3 w-20 bg-border rounded animate-pulse" />
          </div>
        ))}
      </div>
      <div className="bg-white border border-border rounded-lg h-64 animate-pulse" />
    </div>
  )
}
```

### `app/agent/loading.tsx`

Same structure as above.

### `app/auth/forgot-password/page.tsx`

```tsx
'use client'
import { useState } from 'react'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [submitted, setSubmitted] = useState(false)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitted(true)
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">StrataHQ</span>
        </div>

        {submitted ? (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-3">Check your email</h1>
            <p className="text-muted text-sm mb-6">
              If an account exists for <strong>{email}</strong>, a password reset link has been sent.
            </p>
            <Link href="/auth/login" className="text-[13px] text-accent font-medium hover:underline">
              Back to login →
            </Link>
          </div>
        ) : (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">Reset password</h1>
            <p className="text-muted text-sm mb-8">Enter your email and we'll send a reset link.</p>
            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-ink mb-1">Email</label>
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  className="w-full rounded border border-border bg-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                  placeholder="you@example.com"
                />
              </div>
              <button
                type="submit"
                className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:bg-ink-2 transition-colors"
              >
                Send reset link
              </button>
            </form>
            <p className="mt-6 text-center text-sm text-muted">
              <Link href="/auth/login" className="text-accent hover:underline font-medium">Back to login</Link>
            </p>
          </div>
        )}
      </div>
    </main>
  )
}
```

### Fix login "Forgot password?" link

Change `href="#"` to `href="/auth/forgot-password"` in `app/auth/login/page.tsx`.

- [ ] Commit: `feat(app): add 404 page, loading skeletons, and forgot password flow`

---

## Self-Review Checklist

- [ ] No more sidebar 404s (Settings, Profile pages exist)
- [ ] Logout button present and works
- [ ] Trustee can reach Members, Resident can reach AGM and Financials
- [ ] "Days to AGM" and SLA breach badges reflect real current date
- [ ] Invitations page has data and Resend/Revoke actions
- [ ] AGM has Schedule AGM button for agent
- [ ] Financials has Export CSV button
- [ ] Not-found page renders instead of raw Next.js error
- [ ] Loading skeletons appear during route transitions
- [ ] Forgot password flow is complete
- [ ] All sidebar nav uses Link (no full page reloads)
- [ ] Logout button clears session and redirects to login
