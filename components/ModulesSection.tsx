// SVG icons — inline for zero bundle overhead
const icons: Record<string, React.ReactNode> = {
  levy: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <rect x="2" y="4" width="16" height="12" rx="2" stroke="currentColor" strokeWidth="1.5"/>
      <path d="M2 8h16" stroke="currentColor" strokeWidth="1.5"/>
      <path d="M6 12h2M10 12h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
    </svg>
  ),
  maintenance: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <path d="M13.5 3.5a3 3 0 00-3 3c0 .4.08.78.22 1.13L3.5 14.86A1.5 1.5 0 005.64 17l7.23-7.22c.35.14.73.22 1.13.22a3 3 0 000-6z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round"/>
    </svg>
  ),
  comms: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <path d="M17 3H3a1 1 0 00-1 1v9a1 1 0 001 1h2v3l4-3h8a1 1 0 001-1V4a1 1 0 00-1-1z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round"/>
    </svg>
  ),
  agm: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <rect x="3" y="4" width="14" height="13" rx="1.5" stroke="currentColor" strokeWidth="1.5"/>
      <path d="M3 8h14M7 2v4M13 2v4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      <path d="M7 12l2 2 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  reporting: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <path d="M4 14l4-4 3 3 5-6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
      <rect x="2" y="2" width="16" height="16" rx="2" stroke="currentColor" strokeWidth="1.5"/>
    </svg>
  ),
  docs: (
    <svg viewBox="0 0 20 20" fill="none" className="w-[18px] h-[18px]" aria-hidden>
      <path d="M5 2h7l4 4v12a1 1 0 01-1 1H5a1 1 0 01-1-1V3a1 1 0 011-1z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round"/>
      <path d="M12 2v4h4M7 9h6M7 12h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
    </svg>
  ),
}

const modules = [
  {
    iconKey: 'levy',
    name: 'Levy & Payments',
    desc: 'Automated statements, arrears escalation and EFT reconciliation. Your debtors ledger, in real time.',
  },
  {
    iconKey: 'maintenance',
    name: 'Maintenance',
    desc: 'Photo-documented work orders, contractor assignment and SLA tracking from submission to close.',
  },
  {
    iconKey: 'comms',
    name: 'Communications',
    desc: 'Official announcements, emergency alerts and unit-specific messages — with delivery confirmation.',
  },
  {
    iconKey: 'agm',
    name: 'AGM & Voting',
    desc: 'Proxy collection, quorum tracking, live digital voting and auto-generated signed minutes.',
  },
  {
    iconKey: 'reporting',
    name: 'Financial Reporting',
    desc: "Income & expenditure statements, budget vs actual and reserve fund analysis trustees actually understand.",
  },
  {
    iconKey: 'docs',
    name: 'Document Vault',
    desc: 'Rules, insurance, minutes and compliance documents — searchable, versioned, permission-controlled.',
  },
]

export default function ModulesSection() {
  return (
    <section id="modules" className="padding-section bg-surface">
      <div className="max-w-container mx-auto px-container">
        <p className="reveal eyebrow text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
          Platform modules
        </p>
        <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[620px]">
          Everything your scheme needs.
        </h2>
        <p className="reveal text-clamp-p text-ink-2 max-w-[520px] leading-[1.7]">
          Start with your biggest pain point. Add more when you&apos;re ready.
        </p>

        {/* Grid */}
        <div
          className="stagger mt-[clamp(32px,5vw,48px)] grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-[1px]
            bg-border border border-border rounded-lg overflow-hidden"
        >
          {modules.map(({ iconKey, name, desc }) => (
            <div
              key={name}
              className="bg-surface p-[clamp(20px,3vw,28px)] hover:bg-page transition-colors duration-150 cursor-default"
            >
              <div className="w-9 h-9 rounded-lg border border-border bg-page grid place-items-center text-ink-2 mb-[14px] shadow-sm">
                {icons[iconKey]}
              </div>
              <div className="text-[15px] font-semibold text-ink mb-[6px] tracking-[-0.01em]">
                {name}
              </div>
              <div className="text-[13px] text-muted leading-[1.6]">{desc}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
