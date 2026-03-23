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

export default function RegisterPage() {
  const router = useRouter()
  const { login } = useMockAuth()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('agent')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    if (role === 'agent') {
      login({
        role: 'agent',
        orgName: '',
        schemeName: '',
        schemeId: 'scheme-001',
        isWizardComplete: false,
      })
      router.push('/agent/setup')
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
    <main className="min-h-screen bg-page flex items-center justify-center px-container">
      <div className="w-full max-w-sm py-12">
        {/* Logo */}
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
                  className={`flex-1 py-2 text-sm font-medium transition-colors ${
                    role === r
                      ? 'bg-ink text-page'
                      : 'bg-white text-ink-2 hover:bg-border'
                  }`}
                >
                  {ROLE_LABELS[r]}
                </button>
              ))}
            </div>
          </div>

          {/* Full name */}
          <div>
            <label
              htmlFor="name"
              className="block text-sm font-medium text-ink mb-1"
            >
              Full name
            </label>
            <input
              id="name"
              type="text"
              autoComplete="name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded border border-border bg-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="Jane Smith"
            />
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
              className="w-full rounded border border-border bg-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="you@example.com"
            />
          </div>

          {/* Password */}
          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-ink mb-1"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded border border-border bg-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="••••••••"
            />
          </div>

          {/* Submit */}
          <button
            type="submit"
            className="w-full rounded bg-ink text-page py-2.5 text-sm font-semibold hover:bg-ink-2 transition-colors"
          >
            Create account
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
