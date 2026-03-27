'use client'
import Link from 'next/link'
import { useMockAuth } from '@/lib/mock-auth'
import { mockPortfolio } from '@/lib/mock/scheme'

const HEALTH_STYLES = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-[#92400e]',
  poor: 'bg-red-bg text-red',
}

export default function SchemesPage() {
  useMockAuth()
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">All schemes</h1>
      <p className="text-[14px] text-muted mb-8">Schemes managed by your organisation.</p>
      <div className="flex flex-col gap-3">
        {mockPortfolio.map(scheme => (
          <div key={scheme.id} className="bg-white border border-border rounded-lg px-5 py-4 flex items-center justify-between">
            <div>
              <div className="text-[14px] font-semibold text-ink">{scheme.name}</div>
              <div className="text-[12px] text-muted mt-0.5">
                {scheme.unit_count} units · {scheme.address}
              </div>
              <div className="text-[12px] text-muted mt-0.5">
                {scheme.levy_collection_pct}% collected · {scheme.open_maintenance_count} open jobs
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
              </span>
              <Link href={`/app/${scheme.id}`} className="text-[12px] text-accent font-medium">View →</Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
