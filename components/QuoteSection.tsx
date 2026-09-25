const workflows = [
  {
    label: 'FINANCE',
    title: 'Review the exceptions.',
    detail: 'Statement rows move through matching and manual review before payments are applied to levy accounts.',
  },
  {
    label: 'ACCESS',
    title: 'Keep roles in scope.',
    detail: 'Managing agents, trustees, and residents see different scheme and unit workflows according to their access.',
  },
  {
    label: 'OPERATIONS',
    title: 'Keep work traceable.',
    detail: 'Maintenance requests, background imports, and governance records remain connected to the relevant scheme.',
  },
]

export default function QuoteSection() {
  return (
    <section className="relative border-t border-border overflow-hidden">
      <div className="absolute inset-0 bg-premium-surface pointer-events-none" />
      <div className="relative max-w-[1080px] mx-auto px-container py-[clamp(64px,10vh,112px)]">
        <div className="text-center mb-[clamp(40px,5vw,56px)]">
          <p className="reveal text-[11px] font-semibold tracking-[0.14em] uppercase text-muted mb-3">
            What the beta demonstrates
          </p>
          <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.12] tracking-[-0.02em] text-ink max-w-[650px] mx-auto">
            One workspace for decisions that need a record.
          </h2>
          <p className="reveal text-[14px] text-muted mt-5 max-w-[600px] mx-auto leading-[1.7]">
            Explore these workflows with seeded demo accounts. The example data describes product behavior, not customer outcomes.
          </p>
        </div>
        <div className="stagger grid grid-cols-1 md:grid-cols-3 gap-4">
          {workflows.map(({ label, title, detail }) => (
            <div key={label} className="bg-surface border border-border rounded-xl p-[clamp(24px,3vw,32px)] card-lift">
              <p className="text-[11px] font-semibold tracking-[0.12em] text-accent mb-5">{label}</p>
              <h3 className="font-serif text-[22px] font-bold text-ink mb-3">{title}</h3>
              <p className="text-[14px] text-ink-2 leading-[1.7]">{detail}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
