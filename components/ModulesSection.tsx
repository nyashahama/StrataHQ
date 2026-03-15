const modules = [
  {
    icon: '💳',
    name: 'Levy & Payments',
    desc: 'Automated statements, arrears escalation and EFT reconciliation. Your debtors ledger, in real time.',
  },
  {
    icon: '🔧',
    name: 'Maintenance',
    desc: 'Photo-documented work orders, contractor assignment and SLA tracking from submission to close.',
  },
  {
    icon: '📢',
    name: 'Communications',
    desc: 'Official announcements, emergency alerts and unit-specific messages — with delivery confirmation.',
  },
  {
    icon: '🗳️',
    name: 'AGM & Voting',
    desc: 'Proxy collection, quorum tracking, live digital voting and auto-generated signed minutes.',
  },
  {
    icon: '📊',
    name: 'Financial Reporting',
    desc: "Income & expenditure statements, budget vs actual and reserve fund analysis trustees actually understand.",
  },
  {
    icon: '📁',
    name: 'Document Vault',
    desc: 'Rules, insurance, minutes and compliance documents — searchable, versioned, permission-controlled.',
  },
]

export default function ModulesSection() {
  return (
    <section id="modules" className="padding-section bg-white">
      <div className="max-w-container mx-auto px-container">
        <p className="reveal eyebrow text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
          Platform modules
        </p>
        <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[620px]">
          Everything your scheme needs.
        </h2>
        <p className="reveal text-clamp-p text-ink-2 max-w-[520px] leading-[1.7]">
          Start with your biggest pain point. Add more when you're ready.
        </p>

        {/* Grid */}
        <div
          className="stagger mt-[clamp(32px,5vw,48px)] grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-[1px]
            bg-border border border-border rounded-lg overflow-hidden"
        >
          {modules.map(({ icon, name, desc }) => (
            <div
              key={name}
              className="bg-white p-[clamp(20px,3vw,28px)] hover:bg-page transition-colors duration-150 cursor-default"
            >
              <div className="w-9 h-9 rounded-lg border border-border bg-page grid place-items-center text-[16px] mb-[14px] shadow-sm">
                {icon}
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
