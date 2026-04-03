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
            We&apos;re onboarding a limited number of schemes. Request early access and we&apos;ll be in touch.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-[10px] mb-[14px]">
            <Link
              href="/early-access"
              className="px-7 py-3 text-[15px] font-medium text-white bg-accent border border-accent
                rounded hover:opacity-90 transition-opacity duration-150 no-underline"
            >
              Request early access →
            </Link>
          </div>
          <p className="text-[13px] text-muted-2">
            Limited spots · STSMA compliant · No credit card needed
          </p>
        </div>
      </div>
    </section>
  )
}
