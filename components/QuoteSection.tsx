const testimonials = [
  {
    quote: "We went from chasing 30% of our levies every month to collecting 96% on time. The arrears workflow alone saves us two full days of admin.",
    name: 'Lindiwe Dlamini',
    title: 'Managing Director, Pinnacle Property Management',
    location: 'Johannesburg',
    stat: '96%',
    statLabel: 'collection rate',
  },
  {
    quote: "Our trustees used to call me every week asking for updates. Now they just log in. I've reclaimed at least a day a week — and our AGM last year was the smoothest we've ever had.",
    name: 'Riaan Botha',
    title: 'Senior Portfolio Manager, Atlantic Scheme Managers',
    location: 'Cape Town',
    stat: '12h',
    statLabel: 'saved per AGM',
  },
  {
    quote: "The maintenance module changed everything. Residents know their requests are tracked. Contractors know their SLAs. I haven't lost a maintenance job in WhatsApp since day one.",
    name: 'Thandi Mkhize',
    title: 'Estate Manager, Summerview Body Corporate',
    location: 'Durban',
    stat: '0',
    statLabel: 'SLA breaches this quarter',
  },
]

export default function QuoteSection() {
  return (
    <section className="relative border-t border-border overflow-hidden">
      {/* Subtle background */}
      <div className="absolute inset-0 bg-premium-surface pointer-events-none" />

      <div className="relative max-w-[1080px] mx-auto px-container py-[clamp(64px,10vh,112px)]">
        {/* Section label */}
        <div className="text-center mb-[clamp(40px,5vw,56px)]">
          <p className="reveal text-[11px] font-semibold tracking-[0.14em] uppercase text-muted mb-3">
            What managers say
          </p>
          <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.12] tracking-[-0.02em] text-ink max-w-[520px] mx-auto">
            Real results from real schemes.
          </h2>
        </div>

        {/* Testimonial grid */}
        <div className="stagger grid grid-cols-1 md:grid-cols-3 gap-4">
          {testimonials.map(({ quote, name, title, location, stat, statLabel }) => (
            <div
              key={name}
              className="relative bg-surface border border-border rounded-xl p-[clamp(24px,3vw,32px)] flex flex-col gap-5 card-lift glow-border quote-mark"
            >
              {/* Stat callout */}
              <div className="flex items-baseline gap-2">
                <span className="font-serif text-[clamp(28px,4vw,36px)] font-bold text-ink tracking-[-0.03em] leading-none">
                  {stat}
                </span>
                <span className="text-[13px] text-muted">{statLabel}</span>
              </div>

              {/* Quote */}
              <blockquote className="font-serif text-[clamp(14px,1.6vw,16px)] italic text-ink-2 leading-[1.65] flex-1">
                &ldquo;{quote}&rdquo;
              </blockquote>

              {/* Attribution */}
              <div className="pt-4 border-t border-border">
                <div className="text-[13px] font-semibold text-ink">{name}</div>
                <div className="text-[12px] text-muted mt-[2px]">{title}</div>
                <div className="text-[11px] text-muted-2 mt-[1px]">{location}</div>
              </div>
            </div>
          ))}
        </div>

        {/* Social proof strip */}
        <div className="mt-[clamp(32px,5vw,48px)] flex flex-wrap items-center justify-center gap-[clamp(16px,3vw,32px)] text-[13px] text-muted">
          <span className="font-semibold text-ink">2,400+</span> schemes managed
          <span className="text-border-2">·</span>
          <span className="font-semibold text-ink">94%</span> average collection rate
          <span className="text-border-2">·</span>
          <span className="font-semibold text-ink">180K</span> residents on platform
          <span className="text-border-2">·</span>
          <span className="font-semibold text-ink">48h</span> avg maintenance resolution
        </div>
      </div>
    </section>
  )
}
