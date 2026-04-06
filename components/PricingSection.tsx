import Link from 'next/link'

interface PricingTier {
  name: string
  amount: string
  per: string
  features: string[]
  cta: string
  featured?: boolean
}

const tiers: PricingTier[] = [
  {
    name: 'Starter',
    amount: 'R 18',
    per: 'per unit / month · up to 30 units',
    features: [
      'Levy tracking & statements',
      'Maintenance requests',
      'Resident communications',
      'Document vault (5GB)',
      'Email support',
    ],
    cta: 'Get started',
  },
  {
    name: 'Professional',
    amount: 'R 28',
    per: 'per unit / month · unlimited units',
    features: [
      'Everything in Starter',
      'AGM & digital voting',
      'Full financial reporting',
      'PayFast integration',
      'Priority support',
    ],
    cta: 'Start free trial',
    featured: true,
  },
  {
    name: 'Portfolio',
    amount: 'Custom',
    per: 'for managing agents · 5+ schemes',
    features: [
      'Everything in Professional',
      'Multi-scheme dashboard',
      'Team access controls',
      'Accounting integrations',
      'Dedicated account manager',
    ],
    cta: 'Talk to sales',
  },
]

export default function PricingSection() {
  return (
    <section id="pricing" className="padding-section border-t border-border">
      <div className="max-w-[1080px] mx-auto px-container">
        <div className="text-center max-w-[600px] mx-auto mb-[clamp(40px,6vw,56px)]">
          <p className="reveal eyebrow text-[11px] font-semibold tracking-[0.14em] uppercase text-muted mb-3">
            Pricing
          </p>
          <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.12] tracking-[-0.02em] text-ink mb-4">
            Simple, transparent pricing.
          </h2>
          <p className="reveal text-clamp-p text-ink-2 leading-[1.7]">
            Per-unit pricing that scales with your scheme. No setup fees, no hidden costs.
          </p>
        </div>

        {/* Cards */}
        <div className="stagger grid grid-cols-1 sm:grid-cols-3 gap-4 max-w-[400px] sm:max-w-none mx-auto">
          {tiers.map(({ name, amount, per, features, cta, featured }) => (
            <div
              key={name}
              className={`relative rounded-xl p-[clamp(24px,3vw,32px)] border transition-all duration-300
                ${featured
                  ? 'pricing-featured border-white/[0.08] shadow-[0_0_60px_rgba(91,156,246,0.08)]'
                  : 'bg-surface border-border card-lift'
                }`}
            >
              {featured && (
                <span className="absolute -top-[12px] left-1/2 -translate-x-1/2 text-[11px] font-semibold text-white bg-[var(--color-hero-accent)] px-4 py-1 rounded-full whitespace-nowrap tracking-[0.04em] shadow-[0_0_16px_rgba(91,156,246,0.3)]">
                  Most popular
                </span>
              )}

              <div className={`text-[13px] font-semibold mb-3 ${featured ? 'text-white/45' : 'text-muted'}`}>
                {name}
              </div>
              <div className={`font-serif text-clamp-price font-bold tracking-[-0.03em] leading-none mb-1 ${featured ? 'text-white' : 'text-ink'}`}>
                {amount}
              </div>
              <div className={`text-[13px] mb-6 ${featured ? 'text-white/45' : 'text-muted'}`}>
                {per}
              </div>

              <hr className={`border-none h-[1px] my-5 ${featured ? 'bg-white/[0.08]' : 'bg-border'}`} />

              <ul className="flex flex-col gap-2.5">
                {features.map((f) => (
                  <li key={f} className={`text-[13px] flex items-center gap-2.5 ${featured ? 'text-white/65' : 'text-ink-2'}`}>
                    <span className={`text-[11px] flex-shrink-0 font-bold ${featured ? 'text-[var(--color-hero-accent)]' : 'text-green'}`}>✓</span>
                    {f}
                  </li>
                ))}
              </ul>

              <Link
                href="/early-access"
                className={`block w-full mt-7 py-3 text-center text-[14px] font-medium rounded-lg border no-underline transition-all duration-200
                  ${featured
                    ? 'bg-[var(--color-hero-accent)] border-[var(--color-hero-accent)] text-white hover:brightness-110 hover:shadow-[0_0_24px_rgba(91,156,246,0.25)]'
                    : 'bg-page border-border-2 text-ink hover:bg-hover-subtle'
                  }`}
              >
                {cta}
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
