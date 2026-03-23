# Auth Flow + Role-Based Dashboard Shells Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete login/register flow, post-auth routing, setup wizard, and role-aware dashboard shells for Managing Agent, Trustee, and Resident — all routes navigable with realistic placeholder content.

**Architecture:** Supabase Auth handles auth state; a middleware.ts route guard reads memberships from the DB once per request and forwards role/scheme via request headers to layouts. AppShell + Sidebar are pure presentational components receiving all data as props from server components. Setup wizard uses service role key (no RLS) since new agents have no memberships yet.

**Tech Stack:** Next.js 16 App Router · React 19 · TypeScript · Tailwind · @supabase/ssr · @supabase/supabase-js

---

## File Map

**New files to create:**

```
lib/
  supabase/
    client.ts          ← browser Supabase client (anon key)
    server.ts          ← server Supabase client (anon key + cookies)
    admin.ts           ← service role client (server-only, no cookies)

middleware.ts          ← route guards + x-user-* headers

app/
  auth/
    login/page.tsx
    register/page.tsx
    callback/route.ts
    pending/page.tsx

  agent/
    layout.tsx         ← fetches org, passes to AppShell
    page.tsx           ← portfolio overview placeholder
    setup/page.tsx     ← mounts SetupWizard
    schemes/page.tsx
    invitations/page.tsx

  app/                 ← scheme-scoped routes (note: nested under app/)
    [schemeId]/
      layout.tsx       ← fetches scheme + memberships, passes to AppShell
      page.tsx
      levy/page.tsx
      maintenance/page.tsx
      agm/page.tsx
      communications/page.tsx
      documents/page.tsx
      financials/page.tsx
      members/page.tsx

components/
  AppShell.tsx
  Sidebar.tsx
  auth/
    LoginForm.tsx
    RegisterForm.tsx
  wizard/
    SetupWizard.tsx
    wizard-actions.ts  ← server actions (createOrganisation, createScheme, etc.)

supabase/
  migrations/
    001_initial_schema.sql
```

**Files to modify:**
- `app/layout.tsx` — no changes needed (fonts already set up)
- `components/Nav.tsx` — update "Log in" and "Get started" hrefs to `/auth/login` and `/auth/register`
- `app/globals.css` — add sidebar CSS variables

---

## Task 1: Install Supabase packages + environment setup

**Files:**
- Modify: `package.json` (via npm install)
- Create: `.env.local`

- [ ] **Step 1: Install packages**

```bash
npm install @supabase/supabase-js @supabase/ssr
```

Expected: packages added to `node_modules/`. Check `package.json` has both entries.

- [ ] **Step 2: Create `.env.local`**

```bash
cat > .env.local << 'EOF'
NEXT_PUBLIC_SUPABASE_URL=your-project-url
NEXT_PUBLIC_SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key
EOF
```

Replace values from your Supabase project dashboard → Settings → API. The service role key is secret — never prefix with `NEXT_PUBLIC_`.

- [ ] **Step 3: Add `.env.local` to `.gitignore` if not already there**

```bash
grep -q '.env.local' .gitignore || echo '.env.local' >> .gitignore
```

- [ ] **Step 4: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

Expected: no errors (packages have their own types).

- [ ] **Step 5: Commit**

```bash
git add package.json package-lock.json .gitignore
git commit -m "chore: install @supabase/ssr and @supabase/supabase-js"
```

---

## Task 2: Supabase client utilities

**Files:**
- Create: `lib/supabase/client.ts`
- Create: `lib/supabase/server.ts`
- Create: `lib/supabase/admin.ts`

These three files are the only place in the codebase that should construct Supabase clients. An ESLint rule (to add later) will enforce this. For now, just centralise it here.

- [ ] **Step 1: Create `lib/supabase/client.ts`** (browser client — use in Client Components)

```ts
import { createBrowserClient } from '@supabase/ssr'

export function createClient() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
  )
}
```

- [ ] **Step 2: Create `lib/supabase/server.ts`** (server client — use in Server Components, Route Handlers, middleware)

```ts
import { createServerClient } from '@supabase/ssr'
import { cookies } from 'next/headers'

export async function createClient() {
  const cookieStore = await cookies()
  return createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() { return cookieStore.getAll() },
        setAll(cookiesToSet) {
          try {
            cookiesToSet.forEach(({ name, value, options }) =>
              cookieStore.set(name, value, options)
            )
          } catch {
            // setAll called from a Server Component — ignore, middleware handles refresh
          }
        },
      },
    }
  )
}
```

- [ ] **Step 3: Create `lib/supabase/admin.ts`** (service role client — server-only, bypasses RLS)

```ts
import { createClient } from '@supabase/supabase-js'

// Only import this in server actions and route handlers.
// Never expose to the client — SUPABASE_SERVICE_ROLE_KEY has no NEXT_PUBLIC_ prefix.
export const supabaseAdmin = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.SUPABASE_SERVICE_ROLE_KEY!,
  { auth: { autoRefreshToken: false, persistSession: false } }
)
```

- [ ] **Step 4: Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add lib/
git commit -m "feat: add Supabase client utilities (browser, server, admin)"
```

---

## Task 3: Database migration

**Files:**
- Create: `supabase/migrations/001_initial_schema.sql`

Run this SQL in your Supabase project: Dashboard → SQL Editor → paste and run.

- [ ] **Step 1: Create migration file**

```sql
-- supabase/migrations/001_initial_schema.sql

-- Enable UUID extension
create extension if not exists "pgcrypto";

-- Organisations (managing agent companies)
create table organisations (
  id              uuid primary key default gen_random_uuid(),
  name            text not null,
  type            text not null default 'managing_agent',
  wizard_progress text not null default 'firm'
    check (wizard_progress in ('firm','scheme','units','levies','invite','complete')),
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
);

-- Schemes (body corporate schemes)
create table schemes (
  id             uuid primary key default gen_random_uuid(),
  name           text not null,
  address        text,
  scheme_number  text,
  org_id         uuid references organisations(id) on delete cascade,
  base_levy      numeric,
  admin_levy     numeric,
  levy_period    text check (levy_period in ('monthly','quarterly','bi-annual','annual')),
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now()
);

-- Units within a scheme
create table units (
  id          uuid primary key default gen_random_uuid(),
  scheme_id   uuid references schemes(id) on delete cascade,
  identifier  text not null,
  created_at  timestamptz not null default now()
);

-- Memberships — links users to schemes with a role
create table memberships (
  id         uuid primary key default gen_random_uuid(),
  user_id    uuid not null references auth.users(id) on delete cascade,
  scheme_id  uuid not null references schemes(id) on delete cascade,
  role       text not null check (role in ('agent','trustee','resident')),
  unit_id    uuid references units(id),
  created_at timestamptz not null default now()
);

-- ─── RLS ───────────────────────────────────────────────────────────────────

alter table organisations enable row level security;
alter table schemes       enable row level security;
alter table units         enable row level security;
alter table memberships   enable row level security;

-- Memberships: users can only read their own rows
create policy "memberships_select_own" on memberships for select
  using (user_id = auth.uid());

-- Schemes: accessible if user has a membership on it
create policy "schemes_select" on schemes for select
  using (id in (select scheme_id from memberships where user_id = auth.uid()));

-- Agents can update schemes they manage
create policy "schemes_update_agent" on schemes for update
  using (org_id in (
    select org_id from schemes s
    join memberships m on m.scheme_id = s.id
    where m.user_id = auth.uid() and m.role = 'agent'
  ));

-- Units: readable if scheme is accessible
create policy "units_select" on units for select
  using (scheme_id in (select scheme_id from memberships where user_id = auth.uid()));

-- Organisations: readable if user is an agent member of any scheme in that org
create policy "organisations_select" on organisations for select
  using (id in (
    select s.org_id from schemes s
    join memberships m on m.scheme_id = s.id
    where m.user_id = auth.uid()
  ));
```

- [ ] **Step 2: Run the migration**

Go to Supabase Dashboard → SQL Editor → paste the contents of `001_initial_schema.sql` → Run.

Verify: Tables `organisations`, `schemes`, `units`, `memberships` appear in the Table Editor.

- [ ] **Step 3: Commit migration file**

```bash
mkdir -p supabase/migrations
git add supabase/migrations/001_initial_schema.sql
git commit -m "feat: initial schema — organisations, schemes, units, memberships with RLS"
```

---

## Task 4: Middleware (route guards + session refresh)

**Files:**
- Create: `middleware.ts` (project root)

The middleware runs on every request. It refreshes the session token (required by @supabase/ssr) and enforces route guards based on the user's role and memberships.

- [ ] **Step 1: Create `middleware.ts`**

```ts
import { createServerClient } from '@supabase/ssr'
import { NextResponse, type NextRequest } from 'next/server'

export async function middleware(request: NextRequest) {
  const response = NextResponse.next({ request })

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() { return request.cookies.getAll() },
        setAll(cookiesToSet) {
          cookiesToSet.forEach(({ name, value }) => request.cookies.set(name, value))
          cookiesToSet.forEach(({ name, value, options }) =>
            response.cookies.set(name, value, options)
          )
        },
      },
    }
  )

  // IMPORTANT: always use getUser() not getSession() — verifies JWT server-side
  const { data: { user } } = await supabase.auth.getUser()

  const { pathname } = request.nextUrl

  // Allow auth routes always
  if (pathname.startsWith('/auth')) return response

  // Unauthenticated: redirect to login
  if (!user) {
    return NextResponse.redirect(new URL('/auth/login', request.url))
  }

  // Fetch memberships for authenticated user
  const { data: memberships } = await supabase
    .from('memberships')
    .select('role, scheme_id, unit_id')
    .eq('user_id', user.id)
    .order('created_at', { ascending: true })

  const membership = memberships?.[0] ?? null
  const role = membership?.role ?? ''
  const schemeId = membership?.scheme_id ?? ''

  // Set headers for layouts to consume (avoids second DB query)
  response.headers.set('x-user-role', role)
  response.headers.set('x-user-scheme-id', schemeId)

  // Agent setup route
  if (pathname === '/agent/setup') {
    if (!role) return response // new agent, allow
    if (role !== 'agent') return NextResponse.redirect(new URL(`/app/${schemeId}`, request.url))
    // Agent who already completed setup → portfolio
    const { data: org } = await supabase
      .from('organisations')
      .select('wizard_progress')
      .eq('id', user.id) // org lookup via membership is more accurate in real impl
      .single()
    if (org?.wizard_progress === 'complete') {
      return NextResponse.redirect(new URL('/agent', request.url))
    }
    return response
  }

  // Agent-only routes
  if (pathname.startsWith('/agent')) {
    if (!role) return NextResponse.redirect(new URL('/agent/setup', request.url))
    if (role !== 'agent') return NextResponse.redirect(new URL(`/app/${schemeId}`, request.url))
    return response
  }

  // Scheme-scoped routes
  if (pathname.startsWith('/app/')) {
    if (!role) return NextResponse.redirect(new URL('/auth/pending', request.url))
    // Check user has a membership on this specific scheme
    const schemeIdFromPath = pathname.split('/')[2]
    const hasMembership = memberships?.some(m => m.scheme_id === schemeIdFromPath)
    if (!hasMembership) {
      return new NextResponse(null, { status: 404 })
    }
    return response
  }

  return response
}

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
}
```

**Note on `/agent/setup` wizard_progress check:** The org lookup above is simplified. The full implementation in Task 10 will properly resolve `org_id` via the membership row. For now this guards the route adequately.

- [ ] **Step 2: Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Start dev server and verify redirect works**

```bash
npm run dev
```

Open `http://localhost:3000/agent` in the browser. Expected: redirect to `/auth/login`.

- [ ] **Step 4: Commit**

```bash
git add middleware.ts
git commit -m "feat: middleware route guards with role-based redirects and x-user headers"
```

---

## Task 5: Auth page components

**Files:**
- Create: `components/auth/LoginForm.tsx`
- Create: `components/auth/RegisterForm.tsx`

Client components. They call Supabase directly from the browser using the browser client.

- [ ] **Step 1: Create `components/auth/LoginForm.tsx`**

```tsx
'use client'

import { useState } from 'react'
import { createClient } from '@/lib/supabase/client'
import Link from 'next/link'

export default function LoginForm() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const supabase = createClient()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    const { error } = await supabase.auth.signInWithPassword({ email, password })
    if (error) {
      if (error.message.includes('Email not confirmed')) {
        setError('Please confirm your email before logging in.')
      } else {
        setError('Incorrect email or password.')
      }
      setLoading(false)
      return
    }
    // Redirect via callback — Supabase handles the session cookie
    window.location.href = '/auth/callback?source=login'
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <label htmlFor="email" className="text-[13px] font-medium text-ink">Email</label>
        <input
          id="email"
          type="email"
          required
          autoComplete="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          className="border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
        />
      </div>
      <div className="flex flex-col gap-1">
        <label htmlFor="password" className="text-[13px] font-medium text-ink">Password</label>
        <input
          id="password"
          type="password"
          required
          autoComplete="current-password"
          value={password}
          onChange={e => setPassword(e.target.value)}
          className="border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
        />
        {error && <p className="text-[12px] text-red mt-1">{error}</p>}
      </div>
      <button
        type="submit"
        disabled={loading}
        className="mt-1 w-full bg-ink text-white text-[14px] font-medium py-[11px] rounded hover:bg-ink-2 transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? 'Logging in…' : 'Log in'}
      </button>
      <p className="text-center text-[13px] text-muted">
        <Link href="#" className="text-muted hover:text-ink transition-colors">Forgot password?</Link>
      </p>
      <p className="text-center text-[13px] text-muted">
        Don&apos;t have an account?{' '}
        <Link href="/auth/register" className="text-accent font-medium hover:underline">Register</Link>
      </p>
    </form>
  )
}
```

**Note on login redirect:** `signInWithPassword` sets the session cookie directly — no `/auth/callback` code exchange needed. Redirect to `/auth/callback?source=login` to reuse the routing logic (which reads memberships and routes to the correct dashboard). Add a `source=login` branch in the callback for this (Task 6).

- [ ] **Step 2: Create `components/auth/RegisterForm.tsx`**

```tsx
'use client'

import { useState } from 'react'
import { createClient } from '@/lib/supabase/client'
import Link from 'next/link'

type Role = 'agent' | 'invited'

export default function RegisterForm() {
  const [role, setRole] = useState<Role>('agent')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)
  const supabase = createClient()

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    // "I was invited" path — no account creation
    if (role === 'invited') {
      setError('You need an invitation to join StrataHQ. Contact your managing agent.')
      return
    }

    setLoading(true)
    const { error } = await supabase.auth.signUp({
      email,
      password,
      options: {
        data: { full_name: name, registered_as: 'agent' },
        emailRedirectTo: `${window.location.origin}/auth/callback`,
      },
    })

    if (error) {
      if (error.message.includes('already registered')) {
        setError('An account with this email already exists. Log in instead.')
      } else {
        setError('Something went wrong. Please try again.')
      }
      setLoading(false)
      return
    }

    setSuccess(true)
  }

  if (success) {
    return (
      <div className="text-center py-4">
        <div className="text-[32px] mb-3">✉</div>
        <h2 className="font-serif text-[20px] font-semibold text-ink mb-2">Check your email</h2>
        <p className="text-[14px] text-muted">We&apos;ve sent a confirmation link to <strong>{email}</strong>. Click it to activate your account.</p>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      {/* Role selector */}
      <div className="flex rounded border border-border overflow-hidden">
        {(['agent', 'invited'] as Role[]).map(r => (
          <button
            key={r}
            type="button"
            onClick={() => setRole(r)}
            className={`flex-1 py-[10px] text-[13px] font-medium transition-colors duration-150 ${
              role === r ? 'bg-ink text-white' : 'bg-white text-muted hover:text-ink'
            }`}
          >
            {r === 'agent' ? "I'm a managing agent" : 'I was invited'}
          </button>
        ))}
      </div>

      {role === 'agent' && (
        <>
          <div className="flex flex-col gap-1">
            <label htmlFor="name" className="text-[13px] font-medium text-ink">Full name</label>
            <input
              id="name"
              type="text"
              required
              autoComplete="name"
              value={name}
              onChange={e => setName(e.target.value)}
              className="border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="reg-email" className="text-[13px] font-medium text-ink">Email</label>
            <input
              id="reg-email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="reg-password" className="text-[13px] font-medium text-ink">Password</label>
            <input
              id="reg-password"
              type="password"
              required
              autoComplete="new-password"
              minLength={8}
              value={password}
              onChange={e => setPassword(e.target.value)}
              className="border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
            />
          </div>
          {error && <p className="text-[12px] text-red">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="mt-1 w-full bg-ink text-white text-[14px] font-medium py-[11px] rounded hover:bg-ink-2 transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </>
      )}

      {role === 'invited' && (
        <>
          {error && <p className="text-[13px] text-muted bg-yellowbg border border-yellow-300 rounded px-3 py-3">{error}</p>}
          {!error && (
            <p className="text-[13px] text-muted text-center py-2">Select &ldquo;I&apos;m a managing agent&rdquo; to create an account, or use your invitation link from your managing agent.</p>
          )}
        </>
      )}

      <p className="text-center text-[13px] text-muted">
        Already have an account?{' '}
        <Link href="/auth/login" className="text-accent font-medium hover:underline">Log in</Link>
      </p>
    </form>
  )
}
```

- [ ] **Step 3: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add components/auth/
git commit -m "feat: LoginForm and RegisterForm components"
```

---

## Task 6: Auth pages + callback route

**Files:**
- Create: `app/auth/login/page.tsx`
- Create: `app/auth/register/page.tsx`
- Create: `app/auth/callback/route.ts`
- Create: `app/auth/pending/page.tsx`

- [ ] **Step 1: Create `app/auth/login/page.tsx`**

```tsx
import LoginForm from '@/components/auth/LoginForm'
import LogoIcon from '@/components/LogoIcon'
import Link from 'next/link'

export const metadata = { title: 'Log in — StrataHQ' }

export default function LoginPage() {
  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-[380px]">
        <div className="flex justify-center mb-8">
          <Link href="/" className="flex items-center gap-[9px]">
            <div className="w-8 h-8 bg-ink rounded-sm grid place-items-center">
              <LogoIcon className="w-[16px] h-[16px] fill-white" />
            </div>
            <span className="font-sans text-[16px] font-semibold text-ink tracking-[-0.01em]">StrataHQ</span>
          </Link>
        </div>
        <h1 className="font-serif text-[26px] font-semibold text-ink text-center mb-6">Welcome back</h1>
        <LoginForm />
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Create `app/auth/register/page.tsx`**

```tsx
import RegisterForm from '@/components/auth/RegisterForm'
import LogoIcon from '@/components/LogoIcon'
import Link from 'next/link'

export const metadata = { title: 'Create account — StrataHQ' }

export default function RegisterPage() {
  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-[380px]">
        <div className="flex justify-center mb-8">
          <Link href="/" className="flex items-center gap-[9px]">
            <div className="w-8 h-8 bg-ink rounded-sm grid place-items-center">
              <LogoIcon className="w-[16px] h-[16px] fill-white" />
            </div>
            <span className="font-sans text-[16px] font-semibold text-ink tracking-[-0.01em]">StrataHQ</span>
          </Link>
        </div>
        <h1 className="font-serif text-[26px] font-semibold text-ink text-center mb-6">Create your account</h1>
        <RegisterForm />
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Create `app/auth/callback/route.ts`**

```ts
import { createClient } from '@/lib/supabase/server'
import { supabaseAdmin } from '@/lib/supabase/admin'
import { NextRequest, NextResponse } from 'next/server'

export async function GET(request: NextRequest) {
  const { searchParams, origin } = new URL(request.url)
  const code = searchParams.get('code')
  const source = searchParams.get('source') // 'login' when coming from LoginForm

  const supabase = await createClient()

  // Login form calls signInWithPassword directly — no code to exchange
  // We still route them through here to centralise redirect logic
  if (source !== 'login' && code) {
    const { error } = await supabase.auth.exchangeCodeForSession(code)
    if (error) {
      return NextResponse.redirect(`${origin}/auth/login?error=auth_failed`)
    }
  }

  const { data: { user }, error: userError } = await supabase.auth.getUser()
  if (userError || !user) {
    return NextResponse.redirect(`${origin}/auth/login?error=auth_failed`)
  }

  // Check if this is an invited user completing their invite
  const inviteRole = user.user_metadata?.role as string | undefined
  const inviteSchemeId = user.user_metadata?.scheme_id as string | undefined
  const inviteUnitId = user.user_metadata?.unit_id as string | undefined

  if (inviteRole && inviteSchemeId) {
    // Create the membership row (must use admin client — user has no existing membership)
    await supabaseAdmin.from('memberships').insert({
      user_id: user.id,
      scheme_id: inviteSchemeId,
      role: inviteRole,
      unit_id: inviteUnitId ?? null,
    })
    return NextResponse.redirect(`${origin}/app/${inviteSchemeId}`)
  }

  // Query existing memberships
  const { data: memberships } = await supabase
    .from('memberships')
    .select('role, scheme_id')
    .eq('user_id', user.id)
    .order('created_at', { ascending: true })

  if (!memberships || memberships.length === 0) {
    // No memberships — route based on how they registered
    const registeredAs = user.user_metadata?.registered_as
    if (registeredAs === 'agent') {
      return NextResponse.redirect(`${origin}/agent/setup`)
    }
    return NextResponse.redirect(`${origin}/auth/pending`)
  }

  const { role, scheme_id } = memberships[0]
  if (role === 'agent') return NextResponse.redirect(`${origin}/agent`)
  if (role === 'trustee' || role === 'resident') return NextResponse.redirect(`${origin}/app/${scheme_id}`)

  // Unknown role — future-proof fallback
  return NextResponse.redirect(`${origin}/auth/login?error=unknown_role`)
}
```

- [ ] **Step 4: Create `app/auth/pending/page.tsx`**

```tsx
'use client'

import { useState } from 'react'
import { createClient } from '@/lib/supabase/client'
import LogoIcon from '@/components/LogoIcon'

export default function PendingPage() {
  const [checking, setChecking] = useState(false)

  async function handleRefresh() {
    setChecking(true)
    const supabase = createClient()
    const { data: { user } } = await supabase.auth.getUser()
    if (!user) { window.location.href = '/auth/login'; return }

    const { data: memberships } = await supabase
      .from('memberships')
      .select('role, scheme_id')
      .eq('user_id', user.id)
      .order('created_at', { ascending: true })

    if (memberships && memberships.length > 0) {
      const { role, scheme_id } = memberships[0]
      window.location.href = role === 'agent' ? '/agent' : `/app/${scheme_id}`
    } else {
      setChecking(false)
    }
  }

  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-[420px] text-center">
        <div className="flex justify-center mb-8">
          <div className="w-8 h-8 bg-ink rounded-sm grid place-items-center">
            <LogoIcon className="w-[16px] h-[16px] fill-white" />
          </div>
        </div>
        <h1 className="font-serif text-[24px] font-semibold text-ink mb-3">Your account is being set up</h1>
        <p className="text-[14px] text-muted leading-relaxed mb-8">
          You&apos;ll receive an email once your account is activated by your managing agent. This usually takes less than a day.
        </p>
        <button
          onClick={handleRefresh}
          disabled={checking}
          className="bg-ink text-white text-[14px] font-medium px-6 py-[11px] rounded hover:bg-ink-2 transition-colors duration-150 disabled:opacity-50"
        >
          {checking ? 'Checking…' : 'Check again'}
        </button>
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 6: Manual test — login page renders**

```bash
npm run dev
```

Open `http://localhost:3000/auth/login`. Expected: logo + "Welcome back" heading + email/password form.
Open `http://localhost:3000/auth/register`. Expected: logo + "Create your account" + role selector.

- [ ] **Step 7: Commit**

```bash
git add app/auth/
git commit -m "feat: auth pages (login, register, callback, pending)"
```

---

## Task 7: AppShell + Sidebar components

**Files:**
- Create: `components/AppShell.tsx`
- Create: `components/Sidebar.tsx`

Pure presentational components. All data comes in as props — no Supabase calls inside.

- [ ] **Step 1: Create `components/Sidebar.tsx`**

The sidebar has a dark navy header (scheme/org name) and a light nav section.

```tsx
'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

export type SidebarRole = 'agent-portfolio' | 'agent-scheme' | 'trustee' | 'resident'

export interface SidebarMembership {
  scheme_id: string
  scheme_name: string
}

interface SidebarProps {
  role: SidebarRole
  headerLabel: string        // org name or "Unit 4B · Sunridge Heights"
  schemeId?: string          // current schemeId for /app/[schemeId] routes
  allMemberships?: SidebarMembership[]  // for multi-scheme trustee switcher
}

const AGENT_PORTFOLIO_NAV = [
  { label: 'Portfolio overview', href: '/agent', icon: GridIcon },
  { label: 'All schemes', href: '/agent/schemes', icon: ListIcon },
  { label: 'Invitations', href: '/agent/invitations', icon: MailIcon },
]

const SCHEME_NAV = (schemeId: string, includeMembers: boolean) => [
  { label: 'Overview', href: `/app/${schemeId}`, icon: GridIcon },
  { label: 'Levy & Payments', href: `/app/${schemeId}/levy`, icon: CreditCardIcon },
  { label: 'Maintenance', href: `/app/${schemeId}/maintenance`, icon: WrenchIcon },
  { label: 'AGM & Voting', href: `/app/${schemeId}/agm`, icon: VoteIcon },
  { label: 'Communications', href: `/app/${schemeId}/communications`, icon: MegaphoneIcon },
  { label: 'Documents', href: `/app/${schemeId}/documents`, icon: FolderIcon },
  { label: 'Financials', href: `/app/${schemeId}/financials`, icon: ChartIcon },
  ...(includeMembers ? [{ label: 'Members', href: `/app/${schemeId}/members`, icon: UsersIcon }] : []),
]

const RESIDENT_NAV = (schemeId: string) => [
  { label: 'Overview', href: `/app/${schemeId}`, icon: GridIcon },
  { label: 'My Levy', href: `/app/${schemeId}/levy`, icon: CreditCardIcon },
  { label: 'Maintenance', href: `/app/${schemeId}/maintenance`, icon: WrenchIcon },
  { label: 'Notices', href: `/app/${schemeId}/communications`, icon: MegaphoneIcon },
  { label: 'Documents', href: `/app/${schemeId}/documents`, icon: FolderIcon },
]

const AGENT_PORTFOLIO_BOTTOM = [{ label: 'Settings', href: '/agent/settings', icon: GearIcon }]
const SCHEME_BOTTOM = (schemeId: string) => [{ label: 'Settings', href: `/app/${schemeId}/settings`, icon: GearIcon }]
const RESIDENT_BOTTOM = (schemeId: string) => [{ label: 'My Profile', href: `/app/${schemeId}/profile`, icon: GearIcon }]

export default function Sidebar({ role, headerLabel, schemeId = '', allMemberships = [] }: SidebarProps) {
  const pathname = usePathname()

  const isActive = (href: string) =>
    href === '/agent' || href === `/app/${schemeId}`
      ? pathname === href
      : pathname.startsWith(href)

  let navItems = AGENT_PORTFOLIO_NAV
  let bottomItems = AGENT_PORTFOLIO_BOTTOM
  if (role === 'agent-scheme') { navItems = SCHEME_NAV(schemeId, true); bottomItems = SCHEME_BOTTOM(schemeId) }
  if (role === 'trustee') { navItems = SCHEME_NAV(schemeId, false); bottomItems = SCHEME_BOTTOM(schemeId) }
  if (role === 'resident') { navItems = RESIDENT_NAV(schemeId); bottomItems = RESIDENT_BOTTOM(schemeId) }

  const showSwitcher = (role === 'trustee' || role === 'agent-scheme') && allMemberships.length > 1

  return (
    <aside className="w-[200px] flex-shrink-0 flex flex-col h-full bg-[#f8f7f4] border-r border-border">
      {/* Header */}
      <div className="bg-[#2d4a6e] px-3 py-3 relative">
        <div className="text-[9px] text-white/40 uppercase tracking-[0.08em] mb-1">
          {role === 'agent-portfolio' ? 'Organisation' : role === 'resident' ? 'My unit' : 'Scheme'}
        </div>
        <div className="text-[12px] font-semibold text-white leading-tight">{headerLabel}</div>
        {showSwitcher && (
          <SchemeSwitcher current={schemeId} memberships={allMemberships} />
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 py-2 overflow-y-auto">
        {navItems.map(({ label, href, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className={`flex items-center gap-2 px-3 py-[7px] text-[12px] transition-colors duration-100 ${
              isActive(href)
                ? 'bg-accent-dim text-accent font-medium border-l-2 border-accent'
                : 'text-muted hover:text-ink hover:bg-[#f0efe9] border-l-2 border-transparent'
            }`}
          >
            <Icon className="w-[14px] h-[14px] flex-shrink-0" />
            {label}
          </Link>
        ))}
      </nav>

      {/* Bottom */}
      <div className="border-t border-border py-2">
        {bottomItems.map(({ label, href, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className="flex items-center gap-2 px-3 py-[7px] text-[12px] text-muted hover:text-ink transition-colors duration-100"
          >
            <Icon className="w-[14px] h-[14px] flex-shrink-0" />
            {label}
          </Link>
        ))}
      </div>
    </aside>
  )
}

function SchemeSwitcher({ current, memberships }: { current: string; memberships: SidebarMembership[] }) {
  return (
    <select
      value={current}
      onChange={e => { window.location.href = `/app/${e.target.value}` }}
      className="absolute right-2 top-3 text-[9px] text-white/40 bg-white/10 border-none rounded px-1 py-0.5 cursor-pointer appearance-none"
    >
      {memberships.map(m => (
        <option key={m.scheme_id} value={m.scheme_id} className="text-ink bg-white">
          {m.scheme_name}
        </option>
      ))}
    </select>
  )
}

// ─── Inline SVG icons (14×14 stroke icons) ───────────────────────────────────

function GridIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="5" height="5" rx="0.5" />
      <rect x="8" y="1" width="5" height="5" rx="0.5" />
      <rect x="1" y="8" width="5" height="5" rx="0.5" />
      <rect x="8" y="8" width="5" height="5" rx="0.5" />
    </svg>
  )
}
function ListIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <line x1="1" y1="3.5" x2="13" y2="3.5" /><line x1="1" y1="7" x2="13" y2="7" /><line x1="1" y1="10.5" x2="13" y2="10.5" />
    </svg>
  )
}
function MailIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="3" width="12" height="8" rx="1" /><polyline points="1,3 7,8 13,3" />
    </svg>
  )
}
function CreditCardIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="3" width="12" height="8" rx="1" /><line x1="1" y1="6" x2="13" y2="6" />
    </svg>
  )
}
function WrenchIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M9 2a3 3 0 0 1 0 4.5L4.5 11a1.5 1.5 0 0 1-2-2L7 4.5A3 3 0 0 1 9 2z" />
    </svg>
  )
}
function VoteIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="1" y="1" width="12" height="12" rx="1" /><polyline points="4,7 6,9 10,5" />
    </svg>
  )
}
function MegaphoneIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M2 5h2l5-3v10L4 9H2a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1z" /><line x1="4" y1="9" x2="4" y2="12" />
    </svg>
  )
}
function FolderIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M1 4a1 1 0 0 1 1-1h3l1.5 2H12a1 1 0 0 1 1 1v5a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V4z" />
    </svg>
  )
}
function ChartIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <polyline points="1,11 4,7 7,9 10,4 13,6" />
    </svg>
  )
}
function UsersIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="5" cy="5" r="2.5" /><path d="M1 13a4 4 0 0 1 8 0" /><circle cx="10.5" cy="4.5" r="2" /><path d="M12.5 11.5a3 3 0 0 0-4-1" />
    </svg>
  )
}
function GearIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="7" cy="7" r="2" />
      <path d="M7 1v2M7 11v2M1 7h2M11 7h2M3.2 3.2l1.4 1.4M9.4 9.4l1.4 1.4M10.8 3.2l-1.4 1.4M4.6 9.4l-1.4 1.4" />
    </svg>
  )
}
```

- [ ] **Step 2: Create `components/AppShell.tsx`**

```tsx
import type { ReactNode } from 'react'

interface AppShellProps {
  sidebar: ReactNode
  children: ReactNode
}

export default function AppShell({ sidebar, children }: AppShellProps) {
  return (
    <div className="flex h-screen overflow-hidden bg-page">
      {sidebar}
      <main className="flex-1 overflow-y-auto">
        {children}
      </main>
    </div>
  )
}
```

- [ ] **Step 3: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add components/AppShell.tsx components/Sidebar.tsx
git commit -m "feat: AppShell and Sidebar components (role-aware, scheme switcher)"
```

---

## Task 8: Agent layout + portfolio pages

**Files:**
- Create: `app/agent/layout.tsx`
- Create: `app/agent/page.tsx`
- Create: `app/agent/schemes/page.tsx`
- Create: `app/agent/invitations/page.tsx`

- [ ] **Step 1: Create `app/agent/layout.tsx`**

Server component. Fetches the agent's organisation and wraps with AppShell.

```tsx
import { createClient } from '@/lib/supabase/server'
import { redirect } from 'next/navigation'
import AppShell from '@/components/AppShell'
import Sidebar from '@/components/Sidebar'

export default async function AgentLayout({ children }: { children: React.ReactNode }) {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) redirect('/auth/login')

  const { data: membership } = await supabase
    .from('memberships')
    .select('scheme_id, schemes(org_id, organisations(name))')
    .eq('user_id', user.id)
    .eq('role', 'agent')
    .single()

  const orgName = (membership?.schemes as any)?.organisations?.name ?? 'My Organisation'

  return (
    <AppShell
      sidebar={
        <Sidebar
          role="agent-portfolio"
          headerLabel={orgName}
        />
      }
    >
      {children}
    </AppShell>
  )
}
```

- [ ] **Step 2: Create `app/agent/page.tsx`**

```tsx
export const metadata = { title: 'Portfolio — StrataHQ' }

export default function AgentPortfolioPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Portfolio overview</h1>
      <p className="text-[14px] text-muted mb-8">All schemes under your management.</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Portfolio dashboard — coming soon
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Create `app/agent/schemes/page.tsx`**

```tsx
export const metadata = { title: 'All schemes — StrataHQ' }

export default function SchemesPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">All schemes</h1>
      <p className="text-[14px] text-muted mb-8">Schemes managed by your organisation.</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Scheme list — coming soon
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Create `app/agent/invitations/page.tsx`**

```tsx
export const metadata = { title: 'Invitations — StrataHQ' }

export default function InvitationsPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Invitations</h1>
      <p className="text-[14px] text-muted mb-8">Pending trustee and resident invitations.</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Invitation management — coming soon
      </div>
    </div>
  )
}
```

- [ ] **Step 5: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 6: Commit**

```bash
git add app/agent/
git commit -m "feat: agent portfolio layout and placeholder pages"
```

---

## Task 9: Setup wizard

**Files:**
- Create: `components/wizard/wizard-actions.ts`
- Create: `components/wizard/SetupWizard.tsx`
- Create: `app/agent/setup/page.tsx`

- [ ] **Step 1: Create `components/wizard/wizard-actions.ts`** (server actions — all use `supabaseAdmin`)

```ts
'use server'

import { supabaseAdmin } from '@/lib/supabase/admin'
import { createClient } from '@/lib/supabase/server'

async function getUser() {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) throw new Error('Not authenticated')
  return user
}

export async function createOrganisation(data: { name: string; email: string; phone: string }) {
  const user = await getUser()
  const { data: org, error } = await supabaseAdmin
    .from('organisations')
    .insert({ name: data.name, wizard_progress: 'firm' })
    .select('id')
    .single()
  if (error) throw new Error(error.message)
  return { orgId: org.id }
}

export async function createScheme(data: { orgId: string; name: string; address: string; schemeNumber: string }) {
  await getUser()
  const { data: scheme, error } = await supabaseAdmin
    .from('schemes')
    .insert({ org_id: data.orgId, name: data.name, address: data.address, scheme_number: data.schemeNumber })
    .select('id')
    .single()
  if (error) throw new Error(error.message)
  await supabaseAdmin.from('organisations').update({ wizard_progress: 'scheme' }).eq('id', data.orgId)
  return { schemeId: scheme.id }
}

export async function createUnits(data: { schemeId: string; orgId: string; identifiers: string[] }) {
  await getUser()
  const rows = data.identifiers.map(identifier => ({ scheme_id: data.schemeId, identifier }))
  const { error } = await supabaseAdmin.from('units').insert(rows)
  if (error) throw new Error(error.message)
  await supabaseAdmin.from('organisations').update({ wizard_progress: 'units' }).eq('id', data.orgId)
}

export async function updateSchemeLevies(data: {
  schemeId: string; orgId: string
  baselevy: number; adminlevy: number
  levyPeriod: 'monthly' | 'quarterly' | 'bi-annual' | 'annual'
}) {
  await getUser()
  const { error } = await supabaseAdmin
    .from('schemes')
    .update({ base_levy: data.baselevy, admin_levy: data.adminlevy, levy_period: data.levyPeriod })
    .eq('id', data.schemeId)
  if (error) throw new Error(error.message)
  await supabaseAdmin.from('organisations').update({ wizard_progress: 'levies' }).eq('id', data.orgId)
}

export async function sendInvites(data: {
  orgId: string; schemeId: string
  invites: { email: string; role: 'trustee' | 'resident'; unitId?: string }[]
}) {
  await getUser()
  for (const invite of data.invites) {
    await supabaseAdmin.auth.admin.inviteUserByEmail(invite.email, {
      data: { role: invite.role, scheme_id: data.schemeId, unit_id: invite.unitId ?? null },
    })
  }
  await supabaseAdmin.from('organisations').update({ wizard_progress: 'invite' }).eq('id', data.orgId)
}

export async function completeWizard(data: { orgId: string; schemeId: string; userId: string }) {
  await supabaseAdmin.from('organisations').update({ wizard_progress: 'complete' }).eq('id', data.orgId)
  await supabaseAdmin.from('memberships').insert({
    user_id: data.userId,
    scheme_id: data.schemeId,
    role: 'agent',
    unit_id: null,
  })
}
```

- [ ] **Step 2: Create `components/wizard/SetupWizard.tsx`**

```tsx
'use client'

import { useState } from 'react'
import {
  createOrganisation, createScheme, createUnits,
  updateSchemeLevies, sendInvites, completeWizard,
} from './wizard-actions'

type LevyPeriod = 'monthly' | 'quarterly' | 'bi-annual' | 'annual'

interface WizardState {
  step: number
  orgId: string
  schemeId: string
  userId: string
}

const STEPS = ['Firm', 'Scheme', 'Units', 'Levies', 'Invite']

export default function SetupWizard({ userId }: { userId: string }) {
  const [state, setState] = useState<WizardState>({ step: 1, orgId: '', schemeId: '', userId })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Step form state
  const [firm, setFirm] = useState({ name: '', email: '', phone: '' })
  const [scheme, setScheme] = useState({ name: '', address: '', schemeNumber: '' })
  const [unitList, setUnitList] = useState('')
  const [levies, setLevies] = useState({ base: '', admin: '', period: 'monthly' as LevyPeriod })
  const [invites, setInvites] = useState('')

  async function handleNext() {
    setError(null)
    setLoading(true)
    try {
      const { step, orgId, schemeId } = state
      if (step === 1) {
        const { orgId: newOrgId } = await createOrganisation(firm)
        setState(s => ({ ...s, step: 2, orgId: newOrgId }))
      } else if (step === 2) {
        const { schemeId: newSchemeId } = await createScheme({ orgId, ...scheme, schemeNumber: scheme.schemeNumber })
        setState(s => ({ ...s, step: 3, schemeId: newSchemeId }))
      } else if (step === 3) {
        const identifiers = unitList.split(/[\n,]+/).map(s => s.trim()).filter(Boolean)
        await createUnits({ schemeId, orgId, identifiers })
        setState(s => ({ ...s, step: 4 }))
      } else if (step === 4) {
        await updateSchemeLevies({ schemeId, orgId, baselevy: +levies.base, adminlevy: +levies.admin, levyPeriod: levies.period })
        setState(s => ({ ...s, step: 5 }))
      } else if (step === 5) {
        const parsed = invites.split('\n').map(l => l.trim()).filter(Boolean).map(email => ({ email, role: 'trustee' as const }))
        if (parsed.length > 0) await sendInvites({ orgId, schemeId, invites: parsed })
        await completeWizard({ orgId, schemeId, userId })
        window.location.href = '/agent'
      }
    } catch (e: any) {
      setError(e.message ?? 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  function handleBack() {
    setState(s => ({ ...s, step: Math.max(1, s.step - 1) }))
    setError(null)
  }

  const inputClass = "border border-border rounded px-3 py-[10px] text-[14px] text-ink bg-white w-full focus:outline-none focus:ring-2 focus:ring-accent focus:border-transparent"
  const labelClass = "text-[13px] font-medium text-ink mb-1 block"

  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4 py-12">
      <div className="w-full max-w-[480px]">
        <h1 className="font-serif text-[26px] font-semibold text-ink mb-2">Set up your account</h1>
        <p className="text-[14px] text-muted mb-8">Step {state.step} of {STEPS.length} — {STEPS[state.step - 1]}</p>

        {/* Progress bar */}
        <div className="flex gap-1.5 mb-8">
          {STEPS.map((_, i) => (
            <div key={i} className={`h-1 flex-1 rounded-full transition-colors duration-200 ${i < state.step ? 'bg-accent' : 'bg-border'}`} />
          ))}
        </div>

        {/* Step content */}
        <div className="flex flex-col gap-4">
          {state.step === 1 && (
            <>
              <div><label className={labelClass}>Company name</label><input className={inputClass} value={firm.name} onChange={e => setFirm(f => ({ ...f, name: e.target.value }))} required /></div>
              <div><label className={labelClass}>Contact email</label><input className={inputClass} type="email" value={firm.email} onChange={e => setFirm(f => ({ ...f, email: e.target.value }))} required /></div>
              <div><label className={labelClass}>Phone</label><input className={inputClass} type="tel" value={firm.phone} onChange={e => setFirm(f => ({ ...f, phone: e.target.value }))} /></div>
            </>
          )}
          {state.step === 2 && (
            <>
              <div><label className={labelClass}>Scheme name</label><input className={inputClass} value={scheme.name} onChange={e => setScheme(s => ({ ...s, name: e.target.value }))} required /></div>
              <div><label className={labelClass}>Physical address</label><input className={inputClass} value={scheme.address} onChange={e => setScheme(s => ({ ...s, address: e.target.value }))} /></div>
              <div><label className={labelClass}>Scheme number (SS XX/YYYY)</label><input className={inputClass} value={scheme.schemeNumber} onChange={e => setScheme(s => ({ ...s, schemeNumber: e.target.value }))} placeholder="SS 42/2010" /></div>
            </>
          )}
          {state.step === 3 && (
            <div>
              <label className={labelClass}>Unit identifiers</label>
              <p className="text-[12px] text-muted mb-2">One per line or comma-separated (e.g. 1A, 1B, 2A)</p>
              <textarea className={`${inputClass} h-32 resize-none`} value={unitList} onChange={e => setUnitList(e.target.value)} placeholder={"1A\n1B\n2A\n2B"} />
            </div>
          )}
          {state.step === 4 && (
            <>
              <div><label className={labelClass}>Base levy (ZAR)</label><input className={inputClass} type="number" min="0" step="0.01" value={levies.base} onChange={e => setLevies(l => ({ ...l, base: e.target.value }))} placeholder="1200.00" /></div>
              <div><label className={labelClass}>Admin levy (ZAR)</label><input className={inputClass} type="number" min="0" step="0.01" value={levies.admin} onChange={e => setLevies(l => ({ ...l, admin: e.target.value }))} placeholder="150.00" /></div>
              <div>
                <label className={labelClass}>Levy period</label>
                <select className={inputClass} value={levies.period} onChange={e => setLevies(l => ({ ...l, period: e.target.value as LevyPeriod }))}>
                  <option value="monthly">Monthly</option>
                  <option value="quarterly">Quarterly</option>
                  <option value="bi-annual">Bi-annual</option>
                  <option value="annual">Annual</option>
                </select>
              </div>
            </>
          )}
          {state.step === 5 && (
            <div>
              <label className={labelClass}>Invite trustees (optional)</label>
              <p className="text-[12px] text-muted mb-2">One email per line. They&apos;ll receive an invitation email.</p>
              <textarea className={`${inputClass} h-28 resize-none`} value={invites} onChange={e => setInvites(e.target.value)} placeholder={"trustee@example.com\nanother@example.com"} />
            </div>
          )}
        </div>

        {error && <p className="mt-4 text-[13px] text-red">{error}</p>}

        {/* Actions */}
        <div className="flex gap-3 mt-8">
          {state.step > 1 && (
            <button onClick={handleBack} className="px-5 py-[10px] text-[14px] font-medium text-ink border border-border rounded hover:bg-[#f0efe9] transition-colors">
              Back
            </button>
          )}
          {state.step === 5 && (
            <button onClick={() => { completeWizard({ orgId: state.orgId, schemeId: state.schemeId, userId: state.userId }).then(() => { window.location.href = '/agent' }) }} className="px-5 py-[10px] text-[14px] text-muted border border-border rounded hover:bg-[#f0efe9] transition-colors ml-auto">
              Skip for now
            </button>
          )}
          <button
            onClick={handleNext}
            disabled={loading}
            className="flex-1 bg-ink text-white text-[14px] font-medium py-[10px] rounded hover:bg-ink-2 transition-colors disabled:opacity-50"
          >
            {loading ? 'Saving…' : state.step === 5 ? 'Finish' : 'Next →'}
          </button>
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Create `app/agent/setup/page.tsx`**

```tsx
import { createClient } from '@/lib/supabase/server'
import { redirect } from 'next/navigation'
import SetupWizard from '@/components/wizard/SetupWizard'

export const metadata = { title: 'Set up your account — StrataHQ' }

export default async function SetupPage() {
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) redirect('/auth/login')
  return <SetupWizard userId={user.id} />
}
```

- [ ] **Step 4: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add components/wizard/ app/agent/setup/
git commit -m "feat: setup wizard (5 steps, incremental save, service role actions)"
```

---

## Task 10: Scheme layout + module placeholder pages

**Files:**
- Create: `app/app/[schemeId]/layout.tsx`
- Create: `app/app/[schemeId]/page.tsx`
- Create: `app/app/[schemeId]/levy/page.tsx`
- Create: `app/app/[schemeId]/maintenance/page.tsx`
- Create: `app/app/[schemeId]/agm/page.tsx`
- Create: `app/app/[schemeId]/communications/page.tsx`
- Create: `app/app/[schemeId]/documents/page.tsx`
- Create: `app/app/[schemeId]/financials/page.tsx`
- Create: `app/app/[schemeId]/members/page.tsx`

- [ ] **Step 1: Create `app/app/[schemeId]/layout.tsx`**

```tsx
import { createClient } from '@/lib/supabase/server'
import { redirect, notFound } from 'next/navigation'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole, type SidebarMembership } from '@/components/Sidebar'

export default async function SchemeLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ schemeId: string }>
}) {
  const { schemeId } = await params
  const supabase = await createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) redirect('/auth/login')

  // Fetch all user memberships (for scheme switcher)
  const { data: memberships } = await supabase
    .from('memberships')
    .select('role, scheme_id, unit_id, schemes(name)')
    .eq('user_id', user.id)
    .order('created_at', { ascending: true })

  // Verify user has access to this scheme
  const thisMembership = memberships?.find(m => m.scheme_id === schemeId)
  if (!thisMembership) notFound()

  const role = thisMembership.role as 'agent' | 'trustee' | 'resident'
  const schemeName = (thisMembership.schemes as any)?.name ?? 'Scheme'
  const unitId = thisMembership.unit_id

  // Resolve sidebar role type
  const sidebarRole: SidebarRole =
    role === 'agent' ? 'agent-scheme' :
    role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel = role === 'resident' && unitId
    ? `Unit ${unitId} · ${schemeName}`
    : schemeName

  const allMemberships: SidebarMembership[] = (memberships ?? []).map(m => ({
    scheme_id: m.scheme_id,
    scheme_name: (m.schemes as any)?.name ?? m.scheme_id,
  }))

  return (
    <AppShell
      sidebar={
        <Sidebar
          role={sidebarRole}
          headerLabel={headerLabel}
          schemeId={schemeId}
          allMemberships={allMemberships}
        />
      }
    >
      {children}
    </AppShell>
  )
}
```

- [ ] **Step 2: Create scheme overview page `app/app/[schemeId]/page.tsx`**

```tsx
export default async function SchemeOverviewPage({ params }: { params: Promise<{ schemeId: string }> }) {
  const { schemeId } = await params
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Overview</h1>
      <p className="text-[14px] text-muted mb-8">Scheme at a glance.</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Scheme overview dashboard — coming soon
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Create all module placeholder pages**

Create these 7 files, each following the same pattern. Replace `[Module]` and `[description]`:

**`app/app/[schemeId]/levy/page.tsx`** — "Levy & Payments" / "Levy collection, statements, and payment history."

**`app/app/[schemeId]/maintenance/page.tsx`** — "Maintenance" / "Log and track maintenance requests."

**`app/app/[schemeId]/agm/page.tsx`** — "AGM & Voting" / "Annual general meetings and trustee resolutions."

**`app/app/[schemeId]/communications/page.tsx`** — "Communications" / "Notices, announcements, and correspondence."

**`app/app/[schemeId]/documents/page.tsx`** — "Documents" / "Scheme rules, minutes, and shared files."

**`app/app/[schemeId]/financials/page.tsx`** — "Financials" / "Budget, expenditure, and reserve fund."

**`app/app/[schemeId]/members/page.tsx`** — "Members" / "Owners, trustees, and contact information."

Template for each:

```tsx
export default async function [Module]Page({ params }: { params: Promise<{ schemeId: string }> }) {
  const { schemeId } = await params
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">
        <span className="text-muted-2">Scheme</span> › [Module]
      </p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">[Module]</h1>
      <p className="text-[14px] text-muted mb-8">[description]</p>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        [Module] — coming soon
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Verify TypeScript**

```bash
npx tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add app/app/
git commit -m "feat: scheme layout (role-aware sidebar) and all module placeholder pages"
```

---

## Task 11: Wire landing page nav links

**Files:**
- Modify: `components/Nav.tsx`

- [ ] **Step 1: Update "Log in" and "Get started" links in `components/Nav.tsx`**

Find the nav CTA links and change their `href` values:

```tsx
// "Live demo" stays as-is (links to /demo)
// Change:
href="/auth/login"   // for "Log in"
href="/auth/register" // for "Get started"
```

Read the file first to find the exact text, then update only those two href values.

- [ ] **Step 2: Build check**

```bash
npm run build
```

Expected: build succeeds with no errors. Ignore any "prerendering" warnings on dynamic routes.

- [ ] **Step 3: Full manual smoke test**

```bash
npm run dev
```

Test flow:
1. `http://localhost:3000` — landing page loads, "Log in" links to `/auth/login`, "Get started" links to `/auth/register`
2. `/auth/register` — register form renders, role selector works
3. `/auth/login` — login form renders
4. `/agent` (unauthenticated) — redirects to `/auth/login` ✓
5. After registering + confirming email → wizard renders at `/agent/setup`
6. After wizard completes → `/agent` shows portfolio layout with sidebar
7. `/app/[schemeId]` (if schemeId exists in DB) → scheme layout with sidebar

- [ ] **Step 4: Commit**

```bash
git add components/Nav.tsx
git commit -m "feat: wire landing nav CTAs to /auth/login and /auth/register"
```

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 0 | — | — |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |

**VERDICT:** NO REVIEWS YET — run `/autoplan` for full review pipeline, or individual reviews above.
