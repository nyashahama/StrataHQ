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
    <section id="pricing" className="padding-section bg-page border-b border-border">
      <div className="max-w-container mx-auto px-container">
        <p className="reveal eyebrow text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
          Pricing
        </p>
        <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[620px]">
          Simple, transparent pricing.
        </h2>
        <p className="reveal text-clamp-p text-ink-2 max-w-[520px] leading-[1.7]">
          Per-unit pricing that scales with your scheme. No setup fees, no hidden costs.
        </p>

        {/* Cards */}
        <div className="stagger mt-[clamp(32px,5vw,48px)] grid grid-cols-1 sm:grid-cols-3 gap-3 max-w-[380px] sm:max-w-none">
          {tiers.map(({ name, amount, per, features, cta, featured }) => (
            <div
              key={name}
              className={`relative rounded-lg p-[clamp(22px,3vw,32px)] border transition-shadow duration-200 hover:shadow
                ${featured
                  ? 'bg-sidebar-header border-sidebar-header'
                  : 'bg-surface border-border'
                }`}
            >
              {featured && (
                <span className="absolute -top-[11px] left-1/2 -translate-x-1/2 text-[11px] font-semibold text-white bg-accent px-3 py-[3px] rounded-full whitespace-nowrap tracking-[0.04em]">
                  Most popular
                </span>
              )}

              <div className={`text-[13px] font-semibold mb-3 ${featured ? 'text-white/50' : 'text-muted'}`}>
                {name}
              </div>
              <div className={`font-serif text-clamp-price font-bold tracking-[-0.03em] leading-none mb-1 ${featured ? 'text-white' : 'text-ink'}`}>
                {amount}
              </div>
              <div className={`text-[13px] mb-5 ${featured ? 'text-white/50' : 'text-muted'}`}>
                {per}
              </div>

              <hr className={`border-none border-t my-4 ${featured ? 'border-white/[0.12]' : 'border-border'}`} style={{ borderTopWidth: 1 }} />

              <ul className="flex flex-col gap-2">
                {features.map((f) => (
                  <li key={f} className={`text-[13px] flex items-center gap-2 ${featured ? 'text-white/75' : 'text-ink-2'}`}>
                    <span className={`text-[11px] flex-shrink-0 ${featured ? 'text-green' : 'text-green'}`}>✓</span>
                    {f}
                  </li>
                ))}
              </ul>

              <Link
                href="#"
                className={`block w-full mt-6 py-[10px] text-center text-[14px] font-medium rounded border no-underline transition-colors duration-150
                  ${featured
                    ? 'bg-white/10 border-white/20 text-white hover:bg-white/20'
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
