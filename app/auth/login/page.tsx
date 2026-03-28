'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { useMockAuth } from '@/lib/mock-auth'

type Role = 'agent' | 'trustee' | 'resident'

const ROLE_LABELS: Record<Role, string> = {
  agent: 'Managing agent',
  trustee: 'Trustee',
  resident: 'Resident',
}

export default function LoginPage() {
  const router = useRouter()
  const { login } = useMockAuth()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('agent')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    if (role === 'agent') {
      login({
        role: 'agent',
        orgName: 'Acme Property Management',
        schemeName: 'Sunridge Heights',
        schemeId: 'scheme-001',
        isWizardComplete: true,
      })
      router.push('/agent')
    } else if (role === 'trustee') {
      login({
        role: 'trustee',
        orgName: 'Acme Property Management',
        schemeName: 'Sunridge Heights',
        schemeId: 'scheme-001',
        isWizardComplete: true,
      })
      router.push('/app/scheme-001')
    } else {
      login({
        role: 'resident',
        orgName: 'Acme Property Management',
        schemeName: 'Sunridge Heights',
        schemeId: 'scheme-001',
        unitIdentifier: '4B',
        isWizardComplete: true,
      })
      router.push('/app/scheme-001')
    }
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        {/* Logo */}
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
          {/* Role segmented control */}
          <div>
            <label className="block text-xs font-semibold text-muted uppercase tracking-wide mb-2">
              I am a
            </label>
            <div className="flex rounded border border-border overflow-hidden">
              {(Object.keys(ROLE_LABELS) as Role[]).map((r) => (
                <button
                  key={r}
                  type="button"
                  onClick={() => setRole(r)}
                  className={`flex-1 py-2.5 text-sm font-medium transition-colors ${
                    role === r
                      ? 'bg-accent text-white'
                      : 'bg-surface text-muted hover:text-ink hover:bg-hover-subtle'
                  }`}
                >
                  {ROLE_LABELS[r]}
                </button>
              ))}
            </div>
          </div>

          {/* Email */}
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-ink mb-1"
            >
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

          {/* Password */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label
                htmlFor="password"
                className="block text-sm font-medium text-ink"
              >
                Password
              </label>
              <Link
                href="/auth/forgot-password"
                className="text-xs text-accent hover:underline"
              >
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

          {/* Submit */}
          <button
            type="submit"
            className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity"
          >
            Log in
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
