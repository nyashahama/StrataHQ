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
        <p className="text-[14px] text-muted mb-8">This page doesn&apos;t exist or you don&apos;t have access.</p>
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
