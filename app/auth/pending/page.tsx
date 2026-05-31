'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import LogoIcon from '@/components/LogoIcon'
import { postLoginPath } from '@/lib/session'
import type { SessionUser } from '@/lib/session'
import { readBrowserCSRFToken } from '@/lib/csrf'

export default function PendingPage() {
  const [checking, setChecking] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const router = useRouter()

  async function handleCheckAgain() {
    setChecking(true)
    setError(null)

    try {
      const csrfToken = readBrowserCSRFToken()
      const response = await fetch('/api/session/refresh', {
        method: 'POST',
        headers: csrfToken ? { 'x-csrf-token': csrfToken } : undefined,
      })

      if (response.status === 401 || response.status === 403) {
        setError('Your account is still pending setup. Please try again.')
        return
      }

      if (!response.ok) {
        throw new Error('Failed to check account status. Please try again.')
      }

      const session = (await response.json()) as SessionUser | null
      if (!session) {
        setError('Your account is still pending setup. Please try again.')
        return
      }

      router.replace(postLoginPath(session))
    } catch {
      setError('Failed to check account status. Please try again.')
    } finally {
      setChecking(false)
    }
  }

  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-container">
      <div className="w-full max-w-sm py-12 text-center">
        {/* Logo */}
        <div className="flex items-center justify-center gap-2 mb-10">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </div>

        {/* Icon */}
        <div className="w-14 h-14 rounded-full bg-accent-dim flex items-center justify-center mx-auto mb-6">
          <svg
            viewBox="0 0 24 24"
            className="w-7 h-7 fill-none stroke-accent"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25H4.5a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75" />
          </svg>
        </div>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
          Your account is being set up
        </h1>
        <p className="text-muted text-sm leading-relaxed mb-8">
          You&apos;ll receive an email once your account is activated by your managing agent.
        </p>

        <button
          type="button"
          onClick={handleCheckAgain}
          disabled={checking}
          className="inline-flex items-center gap-2 rounded border border-border bg-surface px-5 py-2.5 text-sm font-medium text-ink hover:bg-border transition-colors disabled:opacity-60"
        >
          {checking ? (
            <>
              <svg
                className="w-4 h-4 animate-spin text-muted"
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                />
              </svg>
              Checking…
            </>
          ) : (
            'Check again'
          )}
        </button>

        {error ? <p className="mt-4 text-xs text-red">{error}</p> : null}
      </div>
    </main>
  )
}
