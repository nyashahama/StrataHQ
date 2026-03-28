'use client'

import { useState } from 'react'
import LogoIcon from '@/components/LogoIcon'

export default function PendingPage() {
  const [checking, setChecking] = useState(false)

  function handleCheckAgain() {
    setChecking(true)
    setTimeout(() => setChecking(false), 1500)
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
      </div>
    </main>
  )
}
