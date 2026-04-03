import Link from 'next/link'
import LogoIcon from '@/components/LogoIcon'

export default function EarlyAccessSuccessPage() {
  return (
    <main className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm py-12 text-center">
        <Link href="/" className="flex items-center justify-center gap-2 mb-10 no-underline">
          <LogoIcon className="w-6 h-6 fill-ink" />
          <span className="font-serif font-semibold text-ink text-lg tracking-tight">
            StrataHQ
          </span>
        </Link>

        <div className="w-14 h-14 rounded-full bg-accent-dim flex items-center justify-center mx-auto mb-6">
          <svg
            viewBox="0 0 24 24"
            className="w-7 h-7 fill-none stroke-accent"
            strokeWidth={1.5}
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>

        <h1 className="font-serif text-2xl font-semibold text-ink mb-3">
          Request received
        </h1>
        <p className="text-muted text-sm leading-relaxed mb-8">
          We&apos;ll review your request and send you an email when your access is ready. This usually takes 1–2 business days.
        </p>

        <Link
          href="/"
          className="inline-flex items-center gap-2 rounded border border-border bg-surface px-5 py-2.5 text-sm font-medium text-ink hover:bg-border transition-colors no-underline"
        >
          Back to home
        </Link>
      </div>
    </main>
  )
}
