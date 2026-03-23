'use client'
import { useMockAuth } from '@/lib/mock-auth'

export default function SchemeOverviewPage() {
  const { user } = useMockAuth()
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        {user?.schemeName ?? 'Overview'}
      </h1>
      <p className="text-[14px] text-muted mb-8">Scheme at a glance.</p>
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Total units', value: '24' },
          { label: 'Levies collected', value: '91%' },
          { label: 'Open issues', value: '3' },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Scheme overview dashboard — coming soon
      </div>
    </div>
  )
}
