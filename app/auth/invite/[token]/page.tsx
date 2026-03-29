'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'
import { acceptInviteAction } from '@/lib/auth-actions'
import { setSessionCookie } from '@/lib/auth'

interface InviteInfo {
  email: string
  full_name: string
  role: 'trustee' | 'resident'
}

export default function AcceptInvitePage() {
  const params = useParams()
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

    setSessionCookie(result.user)
    const schemeId = result.user.scheme_memberships[0]?.scheme_id ?? ''
    window.location.replace(`/app/${schemeId}`)
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
