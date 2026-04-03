'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'
import { submitEarlyAccessRequest } from '@/lib/early-access-actions'

export default function EarlyAccessPage() {
  const router = useRouter()
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [schemeName, setSchemeName] = useState('')
  const [unitCount, setUnitCount] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const count = parseInt(unitCount, 10)
    if (isNaN(count) || count <= 0) {
      setError('Unit count must be a positive number')
      return
    }
    setLoading(true)
    const result = await submitEarlyAccessRequest({
      full_name: fullName,
      email,
      scheme_name: schemeName,
      unit_count: count,
    })
    setLoading(false)
    if ('error' in result) {
      setError(result.error)
      return
    }
    router.push('/early-access/success')
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12">
        <Link href="/" className="flex items-center gap-2 mb-8 no-underline">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </Link>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-1">
          Request early access
        </h1>
        <p className="text-muted text-sm mb-8">
          We&apos;re onboarding schemes selectively. Fill in your details and we&apos;ll review your request.
        </p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="full_name" className="block text-sm font-medium text-ink mb-1">
              Full name
            </label>
            <input
              id="full_name"
              type="text"
              autoComplete="name"
              required
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
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
            <label htmlFor="scheme_name" className="block text-sm font-medium text-ink mb-1">
              Scheme name
            </label>
            <input
              id="scheme_name"
              type="text"
              required
              value={schemeName}
              onChange={(e) => setSchemeName(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="Rosewood Estate"
            />
          </div>

          <div>
            <label htmlFor="unit_count" className="block text-sm font-medium text-ink mb-1">
              Number of units
            </label>
            <input
              id="unit_count"
              type="number"
              min="1"
              required
              value={unitCount}
              onChange={(e) => setUnitCount(e.target.value)}
              className="w-full rounded border border-border bg-surface px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none focus:border-accent"
              placeholder="24"
            />
          </div>

          {error && <p className="text-sm text-red">{error}</p>}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-accent text-white py-2.5 text-sm font-semibold hover:opacity-90 transition-opacity disabled:opacity-60"
          >
            {loading ? 'Submitting…' : 'Submit request'}
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
