const stats = [
  { num: '01', label: 'Scheme-scoped access' },
  { num: '02', label: 'Levy review and reconciliation' },
  { num: '03', label: 'Maintenance workflows' },
  { num: '04', label: 'Governance records' },
]

export default function StatsBar() {
  return (
    <div className="relative bg-surface border-y border-border py-[clamp(28px,4vw,40px)]">
      <div className="max-w-[1080px] mx-auto px-container">
        <div className="stagger grid grid-cols-2 md:grid-cols-4 gap-y-6 gap-x-4">
          {stats.map(({ num, label }) => (
            <div key={label} className="text-center md:text-left">
              <div className="font-serif text-clamp-stat font-bold text-ink tracking-[-0.03em] leading-none mb-1.5">
                {num}
              </div>
              <div className="text-[13px] text-muted">{label}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
