import Link from 'next/link'
import ScrollRevealInit from './ScrollRevealInit'

export default function Hero() {
  return (
    <section className="padding-hero">
      <ScrollRevealInit />
      <div className="max-w-container mx-auto px-container">
        {/* Label pill */}
        <div className="inline-flex items-center gap-[7px] text-[12px] font-medium text-accent bg-accent-bg border border-[rgba(43,108,176,0.18)] rounded-full px-3 py-1 mb-7 tracking-[0.02em]">
          <span className="w-[5px] h-[5px] rounded-full bg-accent flex-shrink-0" />
          Built for South African sectional title
        </div>

        {/* Heading */}
        <h1 className="font-serif text-clamp-hero font-bold leading-[1.12] tracking-[-0.02em] text-ink mb-6 max-w-[820px]">
          Stop managing schemes.{' '}
          <br />
          <em className="text-muted not-italic italic">Start running them.</em>
        </h1>

        {/* Subheading */}
        <p className="text-clamp-sub text-ink-2 max-w-[560px] leading-[1.65] mb-9">
          StrataHQ replaces the levy spreadsheets, the WhatsApp maintenance threads,
          and the AGM chaos — with one platform that actually knows what needs your
          attention next.
        </p>

        {/* CTAs */}
        <div className="flex flex-wrap items-center gap-[10px] mb-4">
          <Link
            href="#"
            className="px-6 py-[10px] text-[15px] font-medium text-white bg-accent border border-accent
              rounded hover:opacity-90 transition-opacity duration-150 no-underline"
          >
            Start free trial →
          </Link>
          <Link
            href="/demo"
            className="inline-flex items-center gap-2 px-6 py-[10px] text-[15px] font-medium text-accent
              bg-accent-bg border border-[rgba(43,108,176,0.2)] rounded hover:bg-accent-dim transition-colors duration-150 no-underline"
          >
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent opacity-60" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-accent" />
            </span>
            Try interactive demo
          </Link>
          <Link
            href="#features"
            className="px-6 py-[10px] text-[15px] font-medium text-ink-2 bg-surface border border-border-2
              rounded hover:bg-page transition-colors duration-150 no-underline hidden sm:inline-flex"
          >
            See how it works
          </Link>
        </div>

        {/* Note */}
        <p className="text-[13px] text-muted-2">
          30-day free trial · No credit card · Full access from day one
        </p>
      </div>
    </section>
  )
}
