'use client'
import { useMockAuth } from '@/lib/mock-auth'

export default function AgentPortfolioPage() {
  const { user } = useMockAuth()
  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        Portfolio overview
      </h1>
      <p className="text-[14px] text-muted mb-8">
        Welcome back{user?.orgName ? `, ${user.orgName}` : ''}. All schemes under your management.
      </p>
      {/* Placeholder metric cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Active schemes', value: '3' },
          { label: 'Units managed', value: '47' },
          { label: 'Open maintenance', value: '8' },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>
      <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
        Portfolio dashboard — full view coming soon
      </div>
    </div>
  )
}
