import Link from 'next/link'

export default function CTASection() {
  return (
    <section className="bg-surface border-t border-border">
      <div className="max-w-container mx-auto px-container">
        <div className="reveal max-w-[600px] mx-auto text-center py-[clamp(56px,9vh,80px)]">
          <h2 className="font-serif text-clamp-cta font-bold tracking-[-0.02em] text-ink leading-[1.2] mb-4">
            Ready to bring order to your scheme?
          </h2>
          <p className="text-[16px] text-ink-2 mb-8 leading-[1.65]">
            Start your free 30-day trial. Full access to all modules, no credit
            card required.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-[10px] mb-[14px]">
            <Link
              href="#"
              className="px-7 py-3 text-[15px] font-medium text-white bg-ink border border-ink
                rounded hover:bg-ink-2 transition-colors duration-150 no-underline"
            >
              Start free trial →
            </Link>
            <Link
              href="/demo"
              className="inline-flex items-center gap-2 px-7 py-3 text-[15px] font-medium text-ink-2 bg-surface border border-border-2
                rounded hover:bg-page transition-colors duration-150 no-underline"
            >
              Try interactive demo
            </Link>
          </div>
          <p className="text-[13px] text-muted-2">
            30-day free trial · Cancel any time · STSMA compliant
          </p>
        </div>
      </div>
    </section>
  )
}
