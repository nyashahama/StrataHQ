const slaStats = [
  { label: 'On-time this month', val: '91%', color: 'text-green' },
  { label: 'Breached', val: '2', color: 'text-red' },
  { label: 'Avg resolution', val: '36h', color: 'text-ink' },
]

const jobs = [
  {
    icon: '🚿',
    title: 'Shower drain blocked — Unit 3B',
    meta: 'Assigned: Rapid Plumbing Co.',
    sla: '22h remaining',
    slaOk: true,
    status: 'In progress',
    pillClass: 'bg-yellowbg text-amber',
  },
  {
    icon: '💡',
    title: 'Parking bay lights not working',
    meta: 'Submitted yesterday · No contractor assigned',
    sla: 'SLA breached',
    slaOk: false,
    status: 'Unassigned',
    pillClass: 'bg-red-bg text-red',
  },
  {
    icon: '🏊',
    title: 'Pool pump replacement',
    meta: 'AquaFix Pool Services · Quote R 8,400',
    sla: 'Awaiting approval',
    slaOk: true,
    status: 'Pending',
    pillClass: 'bg-accent-bg text-accent',
  },
  {
    icon: '🌿',
    title: 'Garden service — monthly',
    meta: 'GreenThumb Gardens · Recurring',
    sla: 'Completed in 4h',
    slaOk: true,
    status: 'Resolved',
    pillClass: 'bg-green-bg text-green',
    last: true,
  },
]

export default function MaintenanceMockPanel() {
  return (
    <div className="bg-surface border border-border rounded-lg shadow-lg overflow-hidden">
      {/* SLA summary strip */}
      <div className="border-b border-border px-[18px] py-[12px] bg-page flex items-center gap-5">
        {slaStats.map(({ label, val, color }) => (
          <div key={label} className="flex items-baseline gap-1">
            <span className={`font-serif text-[18px] font-bold leading-none ${color}`}>{val}</span>
            <span className="text-[11px] text-muted">{label}</span>
          </div>
        ))}
      </div>

      {/* Header */}
      <div className="border-b border-border px-[18px] py-[12px] flex items-center justify-between bg-surface">
        <div className="flex items-center gap-2">
          <span className="text-[15px]">🔧</span>
          <span className="text-[13px] font-semibold text-ink">Open Work Orders</span>
        </div>
        <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">
          7 open
        </span>
      </div>

      {/* Jobs */}
      <div className="px-[18px] py-3">
        {jobs.map(({ icon, title, meta, sla, slaOk, status, pillClass, last }) => (
          <div
            key={title}
            className={`flex gap-3 items-start py-[10px] ${last ? '' : 'border-b border-border'}`}
          >
            <div className="w-[38px] h-[38px] rounded-md bg-page flex-shrink-0 grid place-items-center text-[16px] border border-border">
              {icon}
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-[13px] font-semibold text-ink mb-[2px]">{title}</div>
              <div className="text-[11px] text-muted">{meta}</div>
              <div className={`text-[11px] mt-[2px] font-medium ${slaOk ? 'text-muted' : 'text-red'}`}>
                {!slaOk && <span className="mr-1">⚠</span>}{sla}
              </div>
            </div>
            <div className="flex-shrink-0">
              <span className={`text-[11px] font-semibold px-[10px] py-[3px] rounded-full tracking-[0.02em] ${pillClass}`}>
                {status}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
