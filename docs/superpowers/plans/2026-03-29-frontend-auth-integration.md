# Frontend Auth Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `MockAuthProvider` with a real hybrid cookie-based auth system wired to the Go backend, covering login, register, onboarding wizard, invite acceptance, password reset, and all route guards.

**Architecture:** Server actions (`'use server'`) handle auth mutations and set three cookies: `sh_access` (readable JWT, 15 min), `sh_refresh` (httpOnly refresh token, 30 days), `sh_session` (readable JSON user state, 30 days). `AuthProvider` reads `sh_session` synchronously on mount — no network call needed for user state. `apiFetch()` attaches the access token as a Bearer header and auto-refreshes on 401 via a server action.

**Tech Stack:** Next.js 16 App Router, TypeScript, `next/headers` cookies API (async in Next.js 15+), server actions, Go backend at `http://localhost:8080`.

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Create | `.env.local` | `NEXT_PUBLIC_API_URL` + `BACKEND_URL` env vars |
| Create | `lib/auth.tsx` | `AuthProvider`, `useAuth`, `SessionUser` type — replaces `lib/mock-auth.tsx` |
| Create | `lib/auth-actions.ts` | All `'use server'` auth mutations |
| Create | `lib/api.ts` | `apiFetch()` client utility with auto-refresh |
| Modify | `app/layout.tsx` | Swap `MockAuthProvider` → `AuthProvider` |
| Modify | `app/auth/login/page.tsx` | Remove mock, wire `loginAction` |
| Modify | `app/auth/register/page.tsx` | Remove role selector + mock, wire `registerAction` |
| Modify | `app/auth/forgot-password/page.tsx` | Wire `forgotPasswordAction` |
| Create | `app/auth/reset-password/page.tsx` | New password-reset form |
| Create | `app/auth/invite/[token]/page.tsx` | New invite-acceptance page |
| Modify | `components/wizard/SetupWizard.tsx` | Wire `setupAction` |
| Modify | `app/agent/layout.tsx` | Swap `useMockAuth` → `useAuth` |
| Modify | `app/agent/setup/page.tsx` | Swap `useMockAuth` → `useAuth` |
| Modify | `app/agent/page.tsx` | Fix `orgName` (not in session — use fallback) |
| Modify | `app/agent/invitations/page.tsx` | Replace mock data with `apiFetch` calls |
| Modify | `app/app/[schemeId]/layout.tsx` | Swap `useMockAuth` → `useAuth` |
| Delete | `lib/mock-auth.tsx` | Removed after all consumers migrated |

---

## Task 1: Environment Variables

**Files:**
- Create: `.env.local`

- [ ] **Step 1: Create `.env.local`**

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
BACKEND_URL=http://localhost:8080
```

- [ ] **Step 2: Commit**

```bash
git add .env.local
git commit -m "chore: add backend API env vars"
```

---

## Task 2: `lib/auth.tsx` — AuthProvider

Replaces `lib/mock-auth.tsx`. Reads `sh_session` cookie in a `useEffect` (same loading pattern as existing mock auth) and exposes `{ user, loading, clearUser }`.

**Files:**
- Create: `lib/auth.tsx`

- [ ] **Step 1: Write `lib/auth.tsx`**

```tsx
'use client'

import React, { createContext, useContext, useEffect, useState } from 'react'
import { logoutAction } from './auth-actions'

export interface SessionUser {
  id: string
  email: string
  full_name: string
  role: 'admin' | 'trustee' | 'resident'
  wizard_complete: boolean
  scheme_memberships: {
    scheme_id: string
    scheme_name: string
    unit_id: string | null
    role: string
  }[]
}

interface AuthContextValue {
  user: SessionUser | null
  loading: boolean
  clearUser: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function readSessionCookie(): SessionUser | null {
  if (typeof document === 'undefined') return null
  const match = document.cookie.match(/(?:^|;\s*)sh_session=([^;]+)/)
  if (!match) return null
  try {
    return JSON.parse(decodeURIComponent(match[1]))
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setUser(readSessionCookie())
    setLoading(false)
  }, [])

  function clearUser() {
    logoutAction().finally(() => {
      window.location.replace('/auth/login')
    })
  }

  return (
    <AuthContext.Provider value={{ user, loading, clearUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within an AuthProvider')
  return ctx
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/auth.tsx
git commit -m "feat(auth): add AuthProvider with cookie-based session"
```

---

## Task 3: `lib/auth-actions.ts` — Server Actions

All mutations that set/clear cookies live here. The Go backend URL is read from `process.env.BACKEND_URL` (server-only env var).

**Files:**
- Create: `lib/auth-actions.ts`

- [ ] **Step 1: Write `lib/auth-actions.ts`**

```ts
'use server'

import { cookies } from 'next/headers'
import type { SessionUser } from './auth'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

const ACCESS_OPTS = {
  httpOnly: false,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'lax' as const,
  path: '/',
  maxAge: 15 * 60,
}

const REFRESH_OPTS = {
  httpOnly: true,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'lax' as const,
  path: '/',
  maxAge: 30 * 24 * 60 * 60,
}

const SESSION_OPTS = {
  httpOnly: false,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'lax' as const,
  path: '/',
  maxAge: 30 * 24 * 60 * 60,
}

async function setAuthCookies(
  cookieStore: Awaited<ReturnType<typeof cookies>>,
  accessToken: string,
  refreshToken: string,
  me: SessionUser,
) {
  const session: SessionUser = {
    id: me.id,
    email: me.email,
    full_name: me.full_name,
    role: me.role,
    wizard_complete: me.wizard_complete,
    scheme_memberships: me.scheme_memberships ?? [],
  }
  cookieStore.set('sh_access', accessToken, ACCESS_OPTS)
  cookieStore.set('sh_refresh', refreshToken, REFRESH_OPTS)
  cookieStore.set('sh_session', encodeURIComponent(JSON.stringify(session)), SESSION_OPTS)
  return session
}

async function fetchMe(accessToken: string): Promise<SessionUser | null> {
  const res = await fetch(`${BACKEND()}/auth/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
  if (!res.ok) return null
  return res.json()
}

// ─── Login ────────────────────────────────────────────────────────────────────

export async function loginAction(
  email: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })

  if (!res.ok) {
    return { error: res.status === 401 ? 'Invalid email or password' : 'Login failed — please try again' }
  }

  const { access_token, refresh_token } = await res.json()
  const me = await fetchMe(access_token)
  if (!me) return { error: 'Login failed — please try again' }

  const cookieStore = await cookies()
  const session = await setAuthCookies(cookieStore, access_token, refresh_token, me)
  return { user: session }
}

// ─── Register ─────────────────────────────────────────────────────────────────

export async function registerAction(
  email: string,
  password: string,
  full_name: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, full_name }),
  })

  if (!res.ok) {
    if (res.status === 409) return { error: 'An account with this email already exists' }
    return { error: 'Registration failed — please try again' }
  }

  const { access_token, refresh_token } = await res.json()
  const me = await fetchMe(access_token)
  if (!me) return { error: 'Registration failed — please try again' }

  const cookieStore = await cookies()
  const session = await setAuthCookies(cookieStore, access_token, refresh_token, me)
  return { user: session }
}

// ─── Logout ───────────────────────────────────────────────────────────────────

export async function logoutAction(): Promise<void> {
  const cookieStore = await cookies()
  const refreshToken = cookieStore.get('sh_refresh')?.value

  if (refreshToken) {
    await fetch(`${BACKEND()}/auth/logout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    }).catch(() => {})
  }

  cookieStore.delete('sh_access')
  cookieStore.delete('sh_refresh')
  cookieStore.delete('sh_session')
}

// ─── Token refresh ────────────────────────────────────────────────────────────

export async function refreshTokens(): Promise<string | null> {
  const cookieStore = await cookies()
  const refreshToken = cookieStore.get('sh_refresh')?.value
  if (!refreshToken) return null

  const res = await fetch(`${BACKEND()}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })

  if (!res.ok) return null

  const { access_token } = await res.json()
  cookieStore.set('sh_access', access_token, ACCESS_OPTS)
  return access_token
}

// ─── Clear auth ───────────────────────────────────────────────────────────────

export async function clearAuth(): Promise<void> {
  const cookieStore = await cookies()
  cookieStore.delete('sh_access')
  cookieStore.delete('sh_refresh')
  cookieStore.delete('sh_session')
}

// ─── Onboarding setup ─────────────────────────────────────────────────────────

export async function setupAction(data: {
  org_name: string
  contact_email: string
  scheme_name: string
  scheme_address: string
  unit_count: number
}): Promise<{ scheme: { id: string; name: string } } | { error: string }> {
  const cookieStore = await cookies()
  const accessToken = cookieStore.get('sh_access')?.value
  if (!accessToken) return { error: 'Not authenticated' }

  const res = await fetch(`${BACKEND()}/api/v1/onboarding/setup`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(data),
  })

  if (!res.ok) return { error: 'Setup failed — please try again' }

  const result = await res.json()

  // Update session cookie: wizard_complete + first scheme membership
  const raw = cookieStore.get('sh_session')?.value
  if (raw) {
    const session = JSON.parse(decodeURIComponent(raw)) as SessionUser
    session.wizard_complete = true
    session.scheme_memberships = [{
      scheme_id: result.scheme.id,
      scheme_name: result.scheme.name,
      unit_id: null,
      role: 'admin',
    }]
    cookieStore.set('sh_session', encodeURIComponent(JSON.stringify(session)), SESSION_OPTS)
  }

  return { scheme: result.scheme }
}

// ─── Forgot password ──────────────────────────────────────────────────────────

export async function forgotPasswordAction(email: string): Promise<void> {
  await fetch(`${BACKEND()}/auth/forgot-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  }).catch(() => {})
  // Always succeeds from the client's perspective (no email enumeration)
}

// ─── Reset password ───────────────────────────────────────────────────────────

export async function resetPasswordAction(
  token: string,
  password: string,
): Promise<{ ok: true } | { error: string }> {
  const res = await fetch(`${BACKEND()}/auth/reset-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, password }),
  })

  if (!res.ok) {
    if (res.status === 401) return { error: 'This reset link is invalid or has expired' }
    return { error: 'Reset failed — please try again' }
  }

  return { ok: true }
}

// ─── Accept invite ────────────────────────────────────────────────────────────

export async function acceptInviteAction(
  token: string,
  password: string,
): Promise<{ user: SessionUser } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/invitations/${token}/accept`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })

  if (!res.ok) {
    if (res.status === 401) return { error: 'This invite link is invalid or has expired' }
    if (res.status === 409) return { error: 'An account with this email already exists — log in instead' }
    return { error: 'Something went wrong — please try again' }
  }

  const { access_token, refresh_token } = await res.json()
  const me = await fetchMe(access_token)
  if (!me) return { error: 'Something went wrong — please try again' }

  const cookieStore = await cookies()
  const session = await setAuthCookies(cookieStore, access_token, refresh_token, me)
  return { user: session }
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/auth-actions.ts
git commit -m "feat(auth): add server actions for all auth mutations"
```

---

## Task 4: `lib/api.ts` — Client Fetch Utility

**Files:**
- Create: `lib/api.ts`

- [ ] **Step 1: Write `lib/api.ts`**

```ts
import { refreshTokens, clearAuth } from './auth-actions'

function getAccessToken(): string | null {
  if (typeof document === 'undefined') return null
  const match = document.cookie.match(/(?:^|;\s*)sh_access=([^;]+)/)
  return match ? match[1] : null
}

function buildHeaders(token: string | null, extra?: HeadersInit): HeadersInit {
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(extra ?? {}),
  }
}

export async function apiFetch(
  path: string,
  options: RequestInit = {},
): Promise<Response> {
  const base = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'
  const token = getAccessToken()

  const res = await fetch(`${base}${path}`, {
    ...options,
    headers: buildHeaders(token, options.headers),
  })

  if (res.status !== 401) return res

  // Token expired — attempt refresh via server action
  const newToken = await refreshTokens()
  if (!newToken) {
    await clearAuth()
    window.location.replace('/auth/login')
    return res
  }

  // Retry once with new token
  const retry = await fetch(`${base}${path}`, {
    ...options,
    headers: buildHeaders(newToken, options.headers),
  })

  if (retry.status === 401) {
    await clearAuth()
    window.location.replace('/auth/login')
  }

  return retry
}
```

- [ ] **Step 2: Commit**

```bash
git add lib/api.ts
git commit -m "feat(auth): add apiFetch with auto token refresh"
```

---

## Task 5: `app/layout.tsx` — Swap Provider

**Files:**
- Modify: `app/layout.tsx`

- [ ] **Step 1: Update `app/layout.tsx`**

Replace the `MockAuthProvider` import and usage:

```tsx
import type { Metadata } from 'next'
import { Lora, DM_Sans } from 'next/font/google'
import './globals.css'
import { AuthProvider } from '@/lib/auth'
import { ThemeProvider } from '@/components/ThemeProvider'

const lora = Lora({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  style: ['normal', 'italic'],
  variable: '--font-lora',
  display: 'swap',
})

const dmSans = DM_Sans({
  subsets: ['latin'],
  weight: ['300', '400', '500', '600'],
  style: ['normal', 'italic'],
  variable: '--font-dm-sans',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'StrataHQ — Body Corporate Management Platform',
  description:
    'One platform for managing agents, trustees and residents. Levy collections, maintenance, communications and AGMs — clear, connected and under control.',
}

const themeScript = `
(function(){
  try {
    var t = localStorage.getItem('stratahq-theme');
    if (!t) t = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    if (t === 'dark') document.documentElement.classList.add('dark');
  } catch(e) {}
})();
`

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${lora.variable} ${dmSans.variable}`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="bg-page text-ink font-sans antialiased leading-relaxed">
        <ThemeProvider>
          <AuthProvider>{children}</AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/layout.tsx
git commit -m "feat(auth): wire AuthProvider in root layout"
```

---

## Task 6: `app/auth/login/page.tsx`

Remove role selector. Call `loginAction`. Route based on return value.

**Files:**
- Modify: `app/auth/login/page.tsx`

- [ ] **Step 1: Rewrite `app/auth/login/page.tsx`**

```tsx
'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { loginAction } from '@/lib/auth-actions'

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    const result = await loginAction(email, password)
    setLoading(false)

    if ('error' in result) {
      setError(result.error)
      return
    }

    const { user } = result
    if (user.role === 'admin' && !user.wizard_complete) {
      router.replace('/agent/setup')
    } else if (user.role === 'admin') {
      router.replace('/agent')
    } else {
      router.replace(`/app/${user.scheme_memberships[0]?.scheme_id ?? ''}`)
    }
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
          Welcome back
        </h1>
        <p className="text-muted text-sm mb-8">Log in to your account</p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-ink mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label htmlFor="password" className="block text-sm font-medium text-ink">
                Password
              </label>
              <Link href="/auth/forgot-password" className="text-xs text-accent hover:underline">
                Forgot password?
              </Link>
            </div>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="••••••••"
            />
          </div>

          {error && (
            <p className="text-sm text-red">{error}</p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
          >
            {loading ? 'Logging in…' : 'Log in'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-muted">
          Don&apos;t have an account?{' '}
          <Link href="/auth/register" className="text-accent hover:underline font-medium">
            Register
          </Link>
        </p>
      </div>
    </main>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/auth/login/page.tsx
git commit -m "feat(auth): wire login page to real backend"
```

---

## Task 7: `app/auth/register/page.tsx`

Agent-only. Remove role selector. Call `registerAction`.

**Files:**
- Modify: `app/auth/register/page.tsx`

- [ ] **Step 1: Rewrite `app/auth/register/page.tsx`**

```tsx
'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { registerAction } from '@/lib/auth-actions'

export default function RegisterPage() {
  const router = useRouter()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    const result = await registerAction(email, password, name)
    setLoading(false)

    if ('error' in result) {
      setError(result.error)
      return
    }

    router.replace('/agent/setup')
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
          Create your account
        </h1>
        <p className="text-muted text-sm mb-8">Get started with StrataHQ</p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-ink mb-1">
              Full name
            </label>
            <input
              id="name"
              type="text"
              autoComplete="name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="Jane Smith"
            />
          </div>

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-ink mb-1">
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-ink mb-1">
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="••••••••"
            />
          </div>

          {error && (
            <p className="text-sm text-red">{error}</p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
          >
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-muted">
          Already have an account?{' '}
          <Link href="/auth/login" className="text-accent hover:underline font-medium">
            Log in
          </Link>
        </p>
      </div>
    </main>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/auth/register/page.tsx
git commit -m "feat(auth): wire register page — agent-only, remove role selector"
```

---

## Task 8: `app/auth/forgot-password/page.tsx`

Wire `forgotPasswordAction`. No UI changes — just replace the mock `setSubmitted(true)` with a real call.

**Files:**
- Modify: `app/auth/forgot-password/page.tsx`

- [ ] **Step 1: Update `app/auth/forgot-password/page.tsx`**

```tsx
'use client'
import { useState } from 'react'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'
import { forgotPasswordAction } from '@/lib/auth-actions'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    await forgotPasswordAction(email)
    setLoading(false)
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
            <p className="text-muted text-sm mb-8">Enter your email and we&apos;ll send a reset link.</p>
            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-ink mb-1">Email</label>
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                  placeholder="you@example.com"
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
              >
                {loading ? 'Sending…' : 'Send reset link'}
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

- [ ] **Step 2: Commit**

```bash
git add app/auth/forgot-password/page.tsx
git commit -m "feat(auth): wire forgot-password page to real backend"
```

---

## Task 9: `app/auth/reset-password/page.tsx` — New Page

Reads `?token=` from search params. Password + confirm fields. Calls `resetPasswordAction`.

**Files:**
- Create: `app/auth/reset-password/page.tsx`

- [ ] **Step 1: Create `app/auth/reset-password/page.tsx`**

```tsx
'use client'

import { useState, Suspense } from 'react'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { resetPasswordAction } from '@/lib/auth-actions'

function ResetPasswordForm() {
  const searchParams = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (password !== confirm) {
      setError('Passwords do not match')
      return
    }
    setError('')
    setLoading(true)
    const result = await resetPasswordAction(token, password)
    setLoading(false)

    if ('error' in result) {
      setError(result.error)
      return
    }

    setSuccess(true)
  }

  if (!token) {
    return (
      <div>
        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">Invalid link</h1>
        <p className="text-muted text-sm mb-6">This reset link is missing a token.</p>
        <Link href="/auth/forgot-password" className="text-[13px] text-accent font-medium hover:underline">
          Request a new reset link →
        </Link>
      </div>
    )
  }

  if (success) {
    return (
      <div>
        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">Password updated</h1>
        <p className="text-muted text-sm mb-6">Your password has been reset. You can now log in.</p>
        <Link href="/auth/login" className="text-[13px] text-accent font-medium hover:underline">
          Go to login →
        </Link>
      </div>
    )
  }

  return (
    <div>
      <h1 className="font-serif text-2xl font-semibold text-ink mb-1">Set new password</h1>
      <p className="text-muted text-sm mb-8">Enter and confirm your new password below.</p>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div>
          <label htmlFor="password" className="block text-sm font-medium text-ink mb-1">New password</label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={e => setPassword(e.target.value)}
            className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
            placeholder="••••••••"
          />
        </div>
        <div>
          <label htmlFor="confirm" className="block text-sm font-medium text-ink mb-1">Confirm password</label>
          <input
            id="confirm"
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={e => setConfirm(e.target.value)}
            className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
            placeholder="••••••••"
          />
        </div>
        {error && <p className="text-sm text-red">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
        >
          {loading ? 'Updating…' : 'Update password'}
        </button>
      </form>
    </div>
  )
}

export default function ResetPasswordPage() {
  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">StrataHQ</span>
        </div>
        <Suspense fallback={null}>
          <ResetPasswordForm />
        </Suspense>
      </div>
    </main>
  )
}
```

> Note: `useSearchParams()` must be wrapped in `<Suspense>` in Next.js App Router — that's why `ResetPasswordForm` is a separate component.

- [ ] **Step 2: Commit**

```bash
git add app/auth/reset-password/page.tsx
git commit -m "feat(auth): add reset-password page"
```

---

## Task 10: `app/auth/invite/[token]/page.tsx` — New Page

Public invite acceptance. Fetches invite info from backend on load, shows pre-filled email/name, password field.

**Files:**
- Create: `app/auth/invite/[token]/page.tsx`

- [ ] **Step 1: Create directory and page**

```bash
mkdir -p app/auth/invite/\[token\]
```

- [ ] **Step 2: Write `app/auth/invite/[token]/page.tsx`**

```tsx
'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'
import { acceptInviteAction } from '@/lib/auth-actions'

interface InviteInfo {
  email: string
  full_name: string
  role: 'trustee' | 'resident'
}

export default function AcceptInvitePage() {
  const params = useParams()
  const router = useRouter()
  const token = params.token as string

  const [invite, setInvite] = useState<InviteInfo | null>(null)
  const [fetchError, setFetchError] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'
    fetch(`${apiUrl}/api/v1/invitations/${token}`)
      .then(async res => {
        if (!res.ok) {
          setFetchError('This invite link is invalid or has expired.')
          return
        }
        const data = await res.json()
        setInvite(data)
      })
      .catch(() => setFetchError('Something went wrong — please try again.'))
  }, [token])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    const result = await acceptInviteAction(token, password)
    setLoading(false)

    if ('error' in result) {
      setError(result.error)
      return
    }

    const schemeId = result.user.scheme_memberships[0]?.scheme_id ?? ''
    router.replace(`/app/${schemeId}`)
  }

  const ROLE_LABELS: Record<string, string> = {
    trustee: 'Trustee',
    resident: 'Resident',
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <div className="flex items-center gap-2 mb-8">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">StrataHQ</span>
        </div>

        {fetchError ? (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-3">Invalid invite</h1>
            <p className="text-muted text-sm mb-6">{fetchError}</p>
            <Link href="/auth/login" className="text-[13px] text-accent font-medium hover:underline">
              Back to login →
            </Link>
          </div>
        ) : !invite ? (
          <p className="text-muted text-sm">Loading…</p>
        ) : (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
              Accept your invitation
            </h1>
            <p className="text-muted text-sm mb-8">
              You&apos;ve been invited as a <strong>{ROLE_LABELS[invite.role]}</strong>. Set a password to activate your account.
            </p>
            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label className="block text-sm font-medium text-ink mb-1">Full name</label>
                <input
                  type="text"
                  disabled
                  value={invite.full_name}
                  className="w-full rounded border border-border bg-hover-subtle px-3 py-2 text-sm text-muted cursor-not-allowed"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-ink mb-1">Email</label>
                <input
                  type="email"
                  disabled
                  value={invite.email}
                  className="w-full rounded border border-border bg-hover-subtle px-3 py-2 text-sm text-muted cursor-not-allowed"
                />
              </div>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-ink mb-1">
                  Password
                </label>
                <input
                  id="password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                  placeholder="••••••••"
                />
              </div>
              {error && <p className="text-sm text-red">{error}</p>}
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
              >
                {loading ? 'Activating…' : 'Activate account'}
              </button>
            </form>
          </div>
        )}
      </div>
    </main>
  )
}
```

- [ ] **Step 3: Commit**

```bash
git add "app/auth/invite/[token]/page.tsx"
git commit -m "feat(auth): add invite acceptance page"
```

---

## Task 11: `components/wizard/SetupWizard.tsx`

Replace mock auth finish with real `setupAction` call on step 2 submit.

**Files:**
- Modify: `components/wizard/SetupWizard.tsx`

- [ ] **Step 1: Rewrite `components/wizard/SetupWizard.tsx`**

```tsx
'use client'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { setupAction } from '@/lib/auth-actions'

type Step = 1 | 2 | 3

export default function SetupWizard() {
  const router = useRouter()
  const [step, setStep] = useState<Step>(1)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [orgName, setOrgName] = useState('')
  const [contactEmail, setContactEmail] = useState('')
  const [schemeName, setSchemeName] = useState('')
  const [address, setAddress] = useState('')
  const [unitCount, setUnitCount] = useState('')

  function handleStep1(e: React.FormEvent) {
    e.preventDefault()
    setStep(2)
  }

  async function handleStep2(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    const result = await setupAction({
      org_name: orgName,
      contact_email: contactEmail,
      scheme_name: schemeName,
      scheme_address: address,
      unit_count: parseInt(unitCount, 10),
    })
    setLoading(false)

    if ('error' in result) {
      setError(result.error)
      return
    }

    setStep(3)
  }

  function handleFinish() {
    router.replace('/agent')
  }

  const STEP_LABELS = ['Organisation', 'First scheme', 'Done']

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-md py-8 sm:py-12">
        <div className="flex items-center gap-2 mb-10">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">StrataHQ</span>
        </div>

        <div className="flex flex-wrap items-center gap-2 mb-8">
          {STEP_LABELS.map((label, i) => {
            const s = (i + 1) as Step
            const active = s === step
            const done = s < step
            return (
              <div key={label} className="flex items-center gap-2">
                <div className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-semibold flex-shrink-0 ${done ? 'bg-green text-white' : active ? 'bg-accent text-white' : 'bg-border text-muted'}`}>
                  {done ? '✓' : s}
                </div>
                <span className={`text-[12px] hidden sm:inline ${active ? 'text-ink font-semibold' : 'text-muted'}`}>{label}</span>
                {i < STEP_LABELS.length - 1 && <div className="w-8 h-px bg-border mx-1" />}
              </div>
            )
          })}
        </div>

        {step === 1 && (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">Set up your organisation</h1>
            <p className="text-muted text-sm mb-8">Tell us about your property management company.</p>
            <form onSubmit={handleStep1} className="space-y-5">
              <div>
                <label htmlFor="orgName" className="block text-sm font-medium text-ink mb-1">Organisation name</label>
                <input
                  id="orgName"
                  type="text"
                  required
                  value={orgName}
                  onChange={e => setOrgName(e.target.value)}
                  placeholder="e.g. Acme Property Management"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label htmlFor="contactEmail" className="block text-sm font-medium text-ink mb-1">Contact email</label>
                <input
                  id="contactEmail"
                  type="email"
                  required
                  value={contactEmail}
                  onChange={e => setContactEmail(e.target.value)}
                  placeholder="admin@acme.co.za"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <button type="submit" className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity">
                Continue →
              </button>
            </form>
          </div>
        )}

        {step === 2 && (
          <div>
            <h1 className="font-serif text-2xl font-semibold text-ink mb-1">Add your first scheme</h1>
            <p className="text-muted text-sm mb-8">You can add more schemes later from your dashboard.</p>
            <form onSubmit={handleStep2} className="space-y-5">
              <div>
                <label htmlFor="schemeName" className="block text-sm font-medium text-ink mb-1">Scheme name</label>
                <input
                  id="schemeName"
                  type="text"
                  required
                  value={schemeName}
                  onChange={e => setSchemeName(e.target.value)}
                  placeholder="e.g. Sunridge Heights"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label htmlFor="address" className="block text-sm font-medium text-ink mb-1">Physical address</label>
                <input
                  id="address"
                  type="text"
                  required
                  value={address}
                  onChange={e => setAddress(e.target.value)}
                  placeholder="e.g. 14 Ocean Drive, Cape Town"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              <div>
                <label htmlFor="unitCount" className="block text-sm font-medium text-ink mb-1">Number of units</label>
                <input
                  id="unitCount"
                  type="number"
                  min="1"
                  required
                  value={unitCount}
                  onChange={e => setUnitCount(e.target.value)}
                  placeholder="e.g. 24"
                  className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
                />
              </div>
              {error && <p className="text-sm text-red">{error}</p>}
              <div className="flex gap-3">
                <button type="button" onClick={() => setStep(1)} className="px-4 py-2.5 text-sm font-medium text-muted hover:text-ink border border-border rounded transition-colors">
                  ← Back
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="flex-1 rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
                >
                  {loading ? 'Creating…' : 'Continue →'}
                </button>
              </div>
            </form>
          </div>
        )}

        {step === 3 && (
          <div>
            <div className="bg-green-bg border border-green/20 rounded-xl px-6 py-8 text-center mb-6">
              <div className="text-3xl mb-3">✓</div>
              <h1 className="font-serif text-2xl font-semibold text-ink mb-2">You&apos;re all set!</h1>
              <p className="text-sm text-muted">
                <strong className="text-ink">{orgName}</strong> and your first scheme <strong className="text-ink">{schemeName}</strong> have been created. You can now invite trustees and residents from the Members page.
              </p>
            </div>
            <button
              onClick={handleFinish}
              className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity"
            >
              Go to dashboard →
            </button>
          </div>
        )}
      </div>
    </main>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add components/wizard/SetupWizard.tsx
git commit -m "feat(auth): wire onboarding wizard to real backend"
```

---

## Task 12: Route Guards — `app/agent/layout.tsx`

Swap `useMockAuth` → `useAuth`. Update field names.

**Files:**
- Modify: `app/agent/layout.tsx`

- [ ] **Step 1: Update `app/agent/layout.tsx`**

```tsx
'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth'
import AppShell from '@/components/AppShell'
import Sidebar from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'

export default function AgentLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (loading) return
    if (user === null) {
      router.replace('/auth/login')
    } else if (user.role !== 'admin') {
      router.replace(`/app/${user.scheme_memberships[0]?.scheme_id ?? ''}`)
    }
  }, [user, loading, router])

  if (loading || !user || user.role !== 'admin') return null

  return (
    <ToastProvider>
      <AppShell
        headerLabel="My Organisation"
        sidebar={
          <Sidebar
            role="agent-portfolio"
            headerLabel="My Organisation"
          />
        }
      >
        {children}
      </AppShell>
      <Copilot />
    </ToastProvider>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/agent/layout.tsx
git commit -m "feat(auth): update agent layout guard to use real auth"
```

---

## Task 13: Route Guards — `app/agent/setup/page.tsx`

**Files:**
- Modify: `app/agent/setup/page.tsx`

- [ ] **Step 1: Update `app/agent/setup/page.tsx`**

```tsx
'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth'
import SetupWizard from '@/components/wizard/SetupWizard'

export default function SetupPage() {
  const { user, loading } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (loading) return
    if (!user) router.replace('/auth/login')
    else if (user.wizard_complete) router.replace('/agent')
  }, [user, loading, router])

  if (loading || !user || user.wizard_complete) return null

  return <SetupWizard />
}
```

- [ ] **Step 2: Commit**

```bash
git add app/agent/setup/page.tsx
git commit -m "feat(auth): update setup page guard to use real auth"
```

---

## Task 14: `app/agent/page.tsx` — Fix `orgName`

`orgName` is not in the session cookie. Replace with static fallback since the portfolio page still uses mock scheme data.

**Files:**
- Modify: `app/agent/page.tsx`

- [ ] **Step 1: Update import in `app/agent/page.tsx`**

Change:
```tsx
import { useMockAuth } from '@/lib/mock-auth'
```
To:
```tsx
import { useAuth } from '@/lib/auth'
```

Change:
```tsx
const { user } = useMockAuth()
```
To:
```tsx
const { user } = useAuth()
```

Change the subtitle line:
```tsx
{user?.orgName}. {mockPortfolio.length} schemes under management.
```
To:
```tsx
{mockPortfolio.length} schemes under management.
```

- [ ] **Step 2: Commit**

```bash
git add app/agent/page.tsx
git commit -m "feat(auth): update agent portfolio page to use real auth"
```

---

## Task 15: `app/agent/invitations/page.tsx` — Real API

Replace `mockInvitations` with `apiFetch` calls. Loading skeleton + error state.

**Files:**
- Modify: `app/agent/invitations/page.tsx`

- [ ] **Step 1: Rewrite `app/agent/invitations/page.tsx`**

```tsx
'use client'
import { useState, useEffect } from 'react'
import { apiFetch } from '@/lib/api'
import { useToast } from '@/lib/toast'

interface Invitation {
  id: string
  full_name: string
  email: string
  role: 'trustee' | 'resident'
  scheme_name?: string
  unit_id?: string | null
  status: string
  expires_at: string
  created_at: string
}

const ROLE_STYLES: Record<string, string> = {
  trustee:  'bg-accent-bg text-accent',
  resident: 'bg-green-bg text-green',
}

export default function InvitationsPage() {
  const { addToast } = useToast()
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    apiFetch('/api/v1/invitations')
      .then(async res => {
        if (!res.ok) throw new Error('Failed to load')
        const data = await res.json()
        setInvitations(data ?? [])
      })
      .catch(() => addToast('Failed to load invitations', 'error' as never))
      .finally(() => setLoading(false))
  }, [addToast])

  async function handleAction(id: string, action: 'resend' | 'revoke') {
    if (action === 'revoke') {
      const res = await apiFetch(`/api/v1/invitations/${id}`, { method: 'DELETE' })
      if (res.ok) {
        setInvitations(prev => prev.filter(i => i.id !== id))
        addToast('Invitation revoked', 'info' as never)
      } else {
        addToast('Failed to revoke invitation', 'error' as never)
      }
    } else {
      const res = await apiFetch(`/api/v1/invitations/${id}/resend`, { method: 'POST' })
      if (res.ok) {
        addToast('Invitation resent', 'success' as never)
      } else {
        addToast('Failed to resend invitation', 'error' as never)
      }
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Portfolio › Invitations</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Invitations</h1>
      <p className="text-[14px] text-muted mb-8">Pending trustee and resident invitations.</p>

      {loading ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading…
        </div>
      ) : invitations.length === 0 ? (
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No pending invitations
        </div>
      ) : (
        <div className="bg-surface border border-border rounded-lg">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">Pending</span>
            <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">{invitations.length} pending</span>
          </div>
          <div className="overflow-x-auto">
            <div className="px-5 min-w-[480px]">
              <div className="grid grid-cols-[1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Invitee</span><span>Role</span><span>Actions</span>
              </div>
              {invitations.map((inv, i) => (
                <div key={inv.id} className={`grid grid-cols-[1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < invitations.length - 1 ? 'border-b border-border' : ''}`}>
                  <div>
                    <div className="font-medium text-ink">{inv.full_name}</div>
                    <div className="text-[12px] text-muted">{inv.email}</div>
                  </div>
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
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/agent/invitations/page.tsx
git commit -m "feat(auth): wire invitations page to real backend API"
```

---

## Task 16: `app/app/[schemeId]/layout.tsx` — Scheme Route Guard

Swap `useMockAuth` → `useAuth`. Update field references. Fix scheme membership check.

**Files:**
- Modify: `app/app/[schemeId]/layout.tsx`

- [ ] **Step 1: Rewrite `app/app/[schemeId]/layout.tsx`**

```tsx
'use client'
import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'

export default function SchemeLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  const router = useRouter()
  const params = useParams()
  const schemeId = params.schemeId as string

  useEffect(() => {
    if (loading) return
    if (!user) { router.replace('/auth/login'); return }

    // Admins can access any scheme; trustee/resident must be a member of this specific scheme
    if (user.role !== 'admin') {
      const isMember = user.scheme_memberships.some(m => m.scheme_id === schemeId)
      if (!isMember) {
        router.replace(`/app/${user.scheme_memberships[0]?.scheme_id ?? ''}`)
      }
    }
  }, [user, loading, router, schemeId])

  if (loading || !user) return null

  const currentScheme = user.role === 'admin'
    ? user.scheme_memberships.find(m => m.scheme_id === schemeId) ?? user.scheme_memberships[0]
    : user.scheme_memberships.find(m => m.scheme_id === schemeId)

  const sidebarRole: SidebarRole =
    user.role === 'admin' ? 'agent-scheme' :
    user.role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel = currentScheme?.scheme_name ?? schemeId

  return (
    <ToastProvider>
      <AppShell
        headerLabel={headerLabel}
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
      <Copilot />
    </ToastProvider>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add "app/app/[schemeId]/layout.tsx"
git commit -m "feat(auth): update scheme layout guard to use real auth"
```

---

## Task 17: Delete `lib/mock-auth.tsx`

Only do this after Tasks 2–16 are all complete and TypeScript compiles cleanly.

**Files:**
- Delete: `lib/mock-auth.tsx`

- [ ] **Step 1: Verify no remaining imports**

```bash
grep -r "mock-auth" /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app --include="*.tsx" --include="*.ts" | grep -v node_modules | grep -v .next
```

Expected: no output (zero matches).

- [ ] **Step 2: Delete the file**

```bash
rm lib/mock-auth.tsx
```

- [ ] **Step 3: Verify TypeScript**

```bash
cd /home/nyasha-hama/projects/stratahq-nextjs-app/stratahq-app && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore(auth): remove mock auth — all consumers migrated to real auth"
```
