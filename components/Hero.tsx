import Link from 'next/link'
import ScrollRevealInit from './ScrollRevealInit'

export default function Hero() {
  return (
    <section className="relative overflow-hidden hero-gradient noise-overlay">
      <ScrollRevealInit />

      {/* Grid overlay */}
      <div className="absolute inset-0 hero-grid" />

      {/* Glow blobs */}
      <div className="glow-blob w-[500px] h-[500px] bg-[var(--color-hero-accent)] top-[-100px] right-[-100px] animate-pulse-glow" />
      <div className="glow-blob w-[400px] h-[400px] bg-[var(--color-hero-warm)] bottom-[-80px] left-[-80px] opacity-30 animate-pulse-glow" style={{ animationDelay: '2s' }} />

      <div className="relative z-10 max-w-[1080px] mx-auto px-container padding-hero pt-[clamp(120px,18vh,180px)]">
        {/* Label pill */}
        <div className="hero-enter hero-enter-1 inline-flex items-center gap-2 text-[12px] font-medium shimmer-badge border border-white/[0.08] rounded-full px-4 py-1.5 mb-8 tracking-[0.03em]">
          <span className="w-[6px] h-[6px] rounded-full bg-[var(--color-hero-accent)] flex-shrink-0 shadow-[0_0_8px_var(--color-hero-accent)]" />
          <span className="text-white/60">Built for South African sectional title</span>
        </div>

        {/* Heading */}
        <h1 className="hero-enter hero-enter-2 font-serif text-clamp-hero font-bold leading-[1.08] tracking-[-0.03em] text-white mb-7 max-w-[780px]">
          Stop managing schemes.{' '}
          <br className="hidden sm:block" />
          <span className="bg-gradient-to-r from-[var(--color-hero-accent)] to-[var(--color-hero-warm)] bg-clip-text text-transparent">
            Start running them.
          </span>
        </h1>

        {/* Subheading */}
        <p className="hero-enter hero-enter-3 text-clamp-sub text-white/50 max-w-[540px] leading-[1.7] mb-10">
          StrataHQ replaces the levy spreadsheets, the WhatsApp maintenance threads,
          and the AGM chaos — with one platform that actually knows what needs your
          attention next.
        </p>

        {/* CTAs */}
        <div className="hero-enter hero-enter-4 flex flex-wrap items-center gap-3 mb-5">
          <Link
            href="/early-access"
            className="group px-7 py-3 text-[15px] font-medium text-white bg-[var(--color-hero-accent)]
              rounded-lg hover:shadow-[0_0_32px_rgba(91,156,246,0.3)] hover:brightness-110 transition-all duration-300 no-underline inline-flex items-center gap-2"
          >
            Request early access
            <span className="inline-block transition-transform duration-200 group-hover:translate-x-0.5">→</span>
          </Link>
          <Link
            href="#features"
            className="px-7 py-3 text-[15px] font-medium text-white/60 bg-white/[0.04]
              border border-white/[0.08] rounded-lg hover:text-white hover:bg-white/[0.08] hover:border-white/[0.12] transition-all duration-300 hidden sm:inline-flex no-underline"
          >
            See how it works
          </Link>
        </div>

        {/* Note */}
        <p className="hero-enter hero-enter-5 text-[13px] text-white/30 tracking-[0.02em]">
          Limited early access · STSMA compliant · Built in South Africa
        </p>
      </div>

      {/* Bottom fade */}
      <div className="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-[var(--color-page)] to-transparent z-10 dark:from-[#0F0F0E]" />
    </section>
  )
}
