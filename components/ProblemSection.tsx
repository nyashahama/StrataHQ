const problems = [
  {
    icon: '💳',
    title: 'Levy reconciliation is a monthly ordeal',
    desc: "EFT references don't match. Debtors age silently. Chasing arrears consumes hours that belong to management.",
  },
  {
    icon: '🔧',
    title: 'Maintenance disappears into WhatsApp',
    desc: 'A burst pipe reported at 7am is buried by noon. No assignment, no SLA, no record.',
  },
  {
    icon: '📣',
    title: 'Residents have no visibility',
    desc: "What's being spent, when the AGM is, where the conduct rules live — no one knows.",
  },
  {
    icon: '🗳️',
    title: 'AGMs are unnecessarily difficult',
    desc: 'Quorum disputes, proxy form confusion and handwritten minutes are a governance risk.',
  },
]

export default function ProblemSection() {
  return (
    <section id="features" className="padding-section">
      <div className="max-w-container mx-auto px-container">
        <div className="grid grid-cols-1 md:grid-cols-[380px_1fr] gap-[clamp(40px,6vw,80px)] items-start">
          {/* Left column */}
          <div>
            <p className="eyebrow reveal text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
              The problem
            </p>
            <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[620px]">
              Most schemes run on scattered tools.
            </h2>
            <p className="reveal text-clamp-p text-ink-2 max-w-[520px] leading-[1.7]">
              Excel levy sheets, WhatsApp maintenance groups, email threads for
              AGM notices. It works until it doesn't.
            </p>
          </div>

          {/* Problem items */}
          <div className="stagger flex flex-col gap-[2px]">
            {problems.map(({ icon, title, desc }) => (
              <div
                key={title}
                className="flex items-start gap-[14px] px-[18px] py-4 rounded hover:bg-[rgba(55,53,47,0.04)] transition-colors duration-150 cursor-default"
              >
                <div className="w-9 h-9 rounded-lg border border-border bg-white grid place-items-center flex-shrink-0 text-[16px] shadow-sm">
                  {icon}
                </div>
                <div>
                  <div className="text-[14px] font-semibold text-ink mb-[3px]">{title}</div>
                  <div className="text-[13px] text-muted leading-[1.55]">{desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
