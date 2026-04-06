import Link from 'next/link'

export default function CTASection() {
  return (
    <section className="relative overflow-hidden border-t border-border">
      {/* Dark cinematic background */}
      <div className="absolute inset-0 hero-gradient" />
      <div className="absolute inset-0 hero-grid" />

      {/* Glow accents */}
      <div className="glow-blob w-[400px] h-[400px] bg-[var(--color-hero-accent)] top-[-100px] left-[20%] animate-pulse-glow" />
      <div className="glow-blob w-[300px] h-[300px] bg-[var(--color-hero-warm)] bottom-[-80px] right-[10%] opacity-30 animate-pulse-glow" style={{ animationDelay: '2s' }} />

      <div className="relative z-10 max-w-[1080px] mx-auto px-container">
        <div className="reveal max-w-[640px] mx-auto text-center py-[clamp(80px,12vh,120px)]">
          <h2 className="font-serif text-clamp-cta font-bold tracking-[-0.02em] text-white leading-[1.15] mb-5">
            Ready to bring order{' '}
            <br className="hidden sm:block" />
            <span className="bg-gradient-to-r from-[var(--color-hero-accent)] to-[var(--color-hero-warm)] bg-clip-text text-transparent">
              to your scheme?
            </span>
          </h2>
          <p className="text-[16px] text-white/45 mb-10 leading-[1.7]">
            We&apos;re onboarding a limited number of schemes. Request early access and we&apos;ll be in touch.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3 mb-5">
            <Link
              href="/early-access"
              className="group px-8 py-3.5 text-[15px] font-medium text-white bg-[var(--color-hero-accent)]
                rounded-lg hover:shadow-[0_0_40px_rgba(91,156,246,0.3)] hover:brightness-110 transition-all duration-300 no-underline inline-flex items-center gap-2"
            >
              Request early access
              <span className="inline-block transition-transform duration-200 group-hover:translate-x-0.5">→</span>
            </Link>
          </div>
          <p className="text-[13px] text-white/25 tracking-[0.02em]">
            Limited spots · STSMA compliant · No credit card needed
          </p>
        </div>
      </div>
    </section>
  )
}
