'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockMaintenanceRequests, type MaintenanceRequest } from '@/lib/mock/maintenance'

const STATUS_STYLES: Record<string, string> = {
  open:             'bg-red-bg text-red',
  in_progress:      'bg-yellowbg text-[#92400e]',
  pending_approval: 'bg-accent-bg text-accent',
  resolved:         'bg-green-bg text-green',
}

const STATUS_LABELS: Record<string, string> = {
  open:             'Open',
  in_progress:      'In progress',
  pending_approval: 'Pending',
  resolved:         'Resolved',
}

const CATEGORY_ICONS: Record<string, string> = {
  plumbing:   '🚿',
  electrical: '💡',
  structural: '🏗️',
  garden:     '🌿',
  pool:       '🏊',
  other:      '🔧',
}

function isSlaBreached(req: MaintenanceRequest): boolean {
  if (req.status === 'resolved') return false
  const created = new Date(req.created_at).getTime()
  const now = new Date('2025-10-16T12:00:00Z').getTime() // fixed "now" for mock
  const hoursElapsed = (now - created) / (1000 * 60 * 60)
  return hoursElapsed > req.sla_hours
}

export default function MaintenancePage() {
  const { user } = useMockAuth()

  // Resident: show only their unit's requests
  if (user?.role === 'resident') {
    const myRequests = mockMaintenanceRequests.filter(
      r => r.submitted_by_unit === user.unitIdentifier
    )
    return (
      <div className="px-8 py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
        <p className="text-[14px] text-muted mb-8">Your maintenance requests for Unit {user.unitIdentifier}.</p>

        <div className="flex items-center justify-between mb-6">
          <span className="text-[13px] text-muted">{myRequests.length} request{myRequests.length !== 1 ? 's' : ''}</span>
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Submit request
          </button>
        </div>

        {myRequests.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
            No maintenance requests submitted yet.
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {myRequests.map(req => (
              <div key={req.id} className="bg-white border border-border rounded-lg px-5 py-4 flex gap-3">
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[16px]">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[13px] font-semibold text-ink">{req.title}</span>
                    <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_STYLES[req.status]}`}>
                      {STATUS_LABELS[req.status]}
                    </span>
                  </div>
                  <div className="text-[12px] text-muted">{req.contractor_name ?? 'No contractor assigned'}</div>
                  <div className="text-[11px] text-muted mt-1">
                    Submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                    {req.resolved_at && ` · Resolved ${new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // Agent / Trustee view
  const canEdit = user?.role === 'agent'
  const open = mockMaintenanceRequests.filter(r => r.status !== 'resolved')
  const breached = open.filter(isSlaBreached)
  const pendingApproval = open.filter(r => r.status === 'pending_approval')
  const resolvedCount = mockMaintenanceRequests.filter(r => r.status === 'resolved').length

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Maintenance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Maintenance</h1>
      <p className="text-[14px] text-muted mb-8">Log and track maintenance requests.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {[
          { label: 'Open jobs',         value: String(open.length) },
          { label: 'SLA breaches',      value: String(breached.length) },
          { label: 'Pending approval',  value: String(pendingApproval.length) },
          { label: 'Resolved this month', value: String(resolvedCount) },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Work orders */}
      <div className="bg-white border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">Work Orders</span>
          {canEdit && (
            <button className="text-[12px] font-semibold bg-accent text-white px-3 py-1.5 rounded hover:bg-[#245a96] transition-colors">
              + New job
            </button>
          )}
        </div>
        <div className="px-5 py-3 flex flex-col gap-0">
          {mockMaintenanceRequests.map((req, i) => {
            const breached = isSlaBreached(req)
            return (
              <div key={req.id} className={`flex gap-3 items-start py-4 ${i < mockMaintenanceRequests.length - 1 ? 'border-b border-border' : ''}`}>
                <div className="w-9 h-9 rounded bg-page border border-border flex-shrink-0 grid place-items-center text-[15px] mt-0.5">
                  {CATEGORY_ICONS[req.category]}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold text-ink mb-[2px]">{req.title}</div>
                  <div className="text-[11px] text-muted">
                    {req.contractor_name ? `${req.contractor_name}` : 'No contractor assigned'}
                    {req.submitted_by_unit && ` · Unit ${req.submitted_by_unit}`}
                  </div>
                  {breached && (
                    <div className="text-[11px] text-red font-medium mt-[2px]">⚠ SLA breached</div>
                  )}
                  {!breached && req.status !== 'resolved' && (
                    <div className="text-[11px] text-muted mt-[2px]">
                      SLA: {req.sla_hours}h · submitted {new Date(req.created_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                  {req.status === 'resolved' && req.resolved_at && (
                    <div className="text-[11px] text-green mt-[2px]">
                      Resolved {new Date(req.resolved_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full ${STATUS_STYLES[req.status]}`}>
                    {STATUS_LABELS[req.status]}
                  </span>
                  {canEdit && req.status === 'pending_approval' && (
                    <button className="text-[11px] text-accent font-medium hover:underline">Approve</button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
