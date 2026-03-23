export default function SchemesPage() {
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">All schemes</h1>
      <p className="text-[14px] text-muted mb-8">Schemes managed by your organisation.</p>
      {/* Mock scheme cards */}
      <div className="flex flex-col gap-3">
        {['Sunridge Heights', 'Bayside Manor', 'The Palms Estate'].map((name) => (
          <div key={name} className="bg-white border border-border rounded-lg px-5 py-4 flex items-center justify-between">
            <div>
              <div className="text-[14px] font-medium text-ink">{name}</div>
              <div className="text-[12px] text-muted mt-0.5">12 units · Cape Town</div>
            </div>
            <div className="text-[12px] text-accent font-medium">View →</div>
          </div>
        ))}
      </div>
    </div>
  )
}
