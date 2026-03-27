'use client'
import Link from 'next/link'
import { useMockAuth } from '@/lib/mock-auth'
import { mockScheme } from '@/lib/mock/scheme'
import { mockLevyRoll, mockCollectionTrend } from '@/lib/mock/levy'
import { mockMaintenanceRequests } from '@/lib/mock/maintenance'
import { mockUpcomingAgm } from '@/lib/mock/agm'
import { mockNotices } from '@/lib/mock/communications'

function daysUntil(dateStr: string): number {
  const now = new Date()
  const target = new Date(dateStr)
  return Math.ceil((target.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

export default function SchemeOverviewPage() {
  const { user } = useMockAuth()

  const openMaintenance = mockMaintenanceRequests.filter(r => r.status !== 'resolved')
  const collectionPct = mockCollectionTrend[mockCollectionTrend.length - 1].pct
  const daysToAgm = daysUntil(mockUpcomingAgm.date)

  // Resident view
  if (user?.role === 'resident') {
    const myLevyAccount = mockLevyRoll.find(a => a.unit_identifier === user.unitIdentifier)
    const myRequests = mockMaintenanceRequests.filter(
      r => r.submitted_by_unit === user.unitIdentifier && r.status !== 'resolved'
    )
    const recentNotices = mockNotices.slice(0, 3)

    return (
      <div className="px-8 py-8 max-w-[900px]">
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
          {mockScheme.name}
        </h1>
        <p className="text-[14px] text-muted mb-8">Unit {user.unitIdentifier} · Welcome back.</p>

        <div className="grid grid-cols-3 gap-4 mb-8">
          {[
            { label: 'My levy', value: myLevyAccount ? (myLevyAccount.status === 'paid' ? 'Paid ✓' : 'Due') : '—' },
            { label: 'Open requests', value: String(myRequests.length) },
            { label: 'Days to AGM', value: String(daysToAgm) },
          ].map(({ label, value }) => (
            <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
              <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
              <div className="text-[12px] text-muted">{label}</div>
            </div>
          ))}
        </div>

        <h2 className="text-[13px] font-semibold text-ink mb-3">Recent notices</h2>
        <div className="flex flex-col gap-2">
          {recentNotices.map(n => (
            <Link key={n.id} href={`/app/${mockScheme.id}/communications`} className="bg-white border border-border rounded-lg px-5 py-3 flex items-center justify-between hover:bg-page transition-colors">
              <div>
                <div className="text-[13px] font-medium text-ink">{n.title}</div>
                <div className="text-[11px] text-muted mt-0.5">
                  {new Date(n.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                </div>
              </div>
              <span className="text-[12px] text-accent font-medium">Read →</span>
            </Link>
          ))}
        </div>
      </div>
    )
  }

  // Agent / Trustee view
  const slaBreaches = openMaintenance.filter(r => {
    const created = new Date(r.created_at).getTime()
    const now = new Date().getTime()
    return (now - created) / (1000 * 60 * 60) > r.sla_hours
  })

  const overdueLevy = mockLevyRoll.filter(a => a.status === 'overdue')

  const maintenanceEvents = mockMaintenanceRequests
    .filter(r => r.status !== 'resolved')
    .slice(0, 3)
    .map(r => ({
      text: `${r.title} — ${r.status === 'pending_approval' ? 'awaiting approval' : r.status === 'in_progress' ? 'in progress' : 'open'}`,
      time: new Date(r.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' }),
      type: 'maintenance' as const,
    }))

  const noticeEvents = mockNotices.slice(0, 2).map(n => ({
    text: `Notice: ${n.title}`,
    time: new Date(n.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' }),
    type: 'comms' as const,
  }))

  const recentActivity = [...maintenanceEvents, ...noticeEvents]
    .slice(0, 5)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        {mockScheme.name}
      </h1>
      <p className="text-[14px] text-muted mb-8">Scheme at a glance.</p>

      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Total units',          value: String(mockScheme.unit_count) },
          { label: 'Levies collected',     value: `${collectionPct}%` },
          { label: 'Open maintenance',     value: String(openMaintenance.length) },
          { label: 'Days to AGM',          value: String(daysToAgm) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Activity feed */}
        <div>
          <h2 className="text-[13px] font-semibold text-ink mb-3">Recent activity</h2>
          <div className="bg-white border border-border rounded-lg overflow-hidden">
            {recentActivity.map((a, i) => (
              <div key={i} className={`px-5 py-3 ${i < recentActivity.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="text-[12px] text-ink">{a.text}</div>
                <div className="text-[11px] text-muted mt-0.5">{a.time}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Attention items */}
        <div>
          <h2 className="text-[13px] font-semibold text-ink mb-3">Needs attention</h2>
          <div className="flex flex-col gap-2">
            {slaBreaches.length > 0 && (
              <div className="bg-red-bg border border-red/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-red">{slaBreaches.length} SLA breach{slaBreaches.length > 1 ? 'es' : ''}</div>
                <div className="text-[11px] text-red/70 mt-0.5">Maintenance jobs past SLA</div>
              </div>
            )}
            {overdueLevy.length > 0 && (
              <div className="bg-yellowbg border border-[#92400e]/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-[#92400e]">{overdueLevy.length} overdue lev{overdueLevy.length > 1 ? 'ies' : 'y'}</div>
                <div className="text-[11px] text-[#92400e]/70 mt-0.5">
                  Units: {overdueLevy.map(a => a.unit_identifier).join(', ')}
                </div>
              </div>
            )}
            {slaBreaches.length === 0 && overdueLevy.length === 0 && (
              <div className="bg-green-bg border border-green/20 rounded-lg px-4 py-3">
                <div className="text-[12px] font-semibold text-green">All clear</div>
                <div className="text-[11px] text-green/70 mt-0.5">No urgent items</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
