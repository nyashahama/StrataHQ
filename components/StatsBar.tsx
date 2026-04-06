const stats = [
  { num: '2,400+', label: 'Schemes managed' },
  { num: '94%', label: 'Average levy collection rate' },
  { num: '180K', label: 'Residents on platform' },
  { num: '48h', label: 'Average maintenance resolution' },
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
