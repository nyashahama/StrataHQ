'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockPortfolio } from '@/lib/mock/scheme'

const HEALTH_STYLES = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-[#92400e]',
  poor: 'bg-red-bg text-red',
}

export default function AgentPortfolioPage() {
  const { user } = useMockAuth()

  const totalUnits = mockPortfolio.reduce((s, p) => s + p.unit_count, 0)
  const totalMaintenance = mockPortfolio.reduce((s, p) => s + p.open_maintenance_count, 0)
  const avgCollection = Math.round(
    mockPortfolio.reduce((s, p) => s + p.levy_collection_pct, 0) / mockPortfolio.length
  )

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Portfolio overview</h1>
      <p className="text-[14px] text-muted mb-8">
        {user?.orgName}. {mockPortfolio.length} schemes under management.
      </p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Active schemes',     value: String(mockPortfolio.length) },
          { label: 'Units managed',      value: String(totalUnits) },
          { label: 'Open maintenance',   value: String(totalMaintenance) },
          { label: 'Avg collection rate', value: `${avgCollection}%` },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Scheme list */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border">
          <span className="text-[13px] font-semibold text-ink">All schemes</span>
        </div>
        <div className="px-5">
          <div className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
            <span>Scheme</span><span>Units</span><span>Collection</span><span>Maintenance</span><span>Health</span>
          </div>
          {mockPortfolio.map((scheme, i) => (
            <div key={scheme.id} className={`grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 items-center py-3 text-[13px] ${i < mockPortfolio.length - 1 ? 'border-b border-border' : ''}`}>
              <div>
                <div className="font-semibold text-ink">{scheme.name}</div>
                <div className="text-[12px] text-muted">{scheme.address}</div>
              </div>
              <span className="text-muted">{scheme.unit_count}</span>
              <span className="font-medium text-ink">{scheme.levy_collection_pct}%</span>
              <span className="text-muted">{scheme.open_maintenance_count} open</span>
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
