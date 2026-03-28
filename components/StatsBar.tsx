const stats = [
  { num: '2,400+', label: 'Schemes managed' },
  { num: '94%', label: 'Average levy collection rate' },
  { num: '180K', label: 'Residents on platform' },
  { num: '48h', label: 'Average maintenance resolution' },
]

export default function StatsBar() {
  return (
    <div className="border-t border-b border-border bg-surface py-[clamp(20px,3vw,28px)]">
      <div className="max-w-container mx-auto px-container">
        <div className="stagger flex flex-wrap justify-between gap-[clamp(16px,3vw,32px)]">
          {stats.map(({ num, label }) => (
            <div key={label} className="text-left">
              <div className="font-serif text-clamp-stat font-bold text-ink tracking-[-0.03em] leading-none mb-1">
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
