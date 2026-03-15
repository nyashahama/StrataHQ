const jobs = [
  {
    icon: '🚿',
    title: 'Shower drain blocked — Unit 3B',
    meta: 'Submitted 2h ago · Assigned to Rapid Plumbing',
    status: 'In progress',
    pillClass: 'bg-yellowbg text-[#92400e]',
  },
  {
    icon: '💡',
    title: 'Parking bay lights not working',
    meta: 'Submitted yesterday · Awaiting contractor',
    status: 'Unassigned',
    pillClass: 'bg-red-bg text-red',
  },
  {
    icon: '🏊',
    title: 'Pool pump replacement',
    meta: 'Trustee approval required · Quote: R 8,400',
    status: 'Pending',
    pillClass: 'bg-yellowbg text-[#92400e]',
  },
  {
    icon: '🌿',
    title: 'Garden service — monthly',
    meta: 'Recurring · GreenThumb Gardens · SLA: 48h',
    status: 'Scheduled',
    pillClass: 'bg-green-bg text-green',
    last: true,
  },
]

export default function MaintenanceMockPanel() {
  return (
    <div className="bg-white border border-border rounded-lg shadow-lg overflow-hidden">
      {/* Header */}
      <div className="border-b border-border px-[18px] py-[14px] flex items-center justify-between bg-white">
        <div className="flex items-center gap-2">
          <span className="text-[15px]">🔧</span>
          <span className="text-[13px] font-semibold text-ink">Open Work Orders</span>
        </div>
        <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">
          7 open
        </span>
      </div>

      {/* Body */}
      <div className="px-[18px] py-4">
        {jobs.map(({ icon, title, meta, status, pillClass, last }) => (
          <div
            key={title}
            className={`flex gap-3 items-start py-[10px] ${last ? '' : 'border-b border-border'}`}
          >
            <div className="w-[42px] h-[42px] rounded-md bg-border flex-shrink-0 grid place-items-center text-[18px] border border-border-2">
              {icon}
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-[13px] font-semibold text-ink mb-[2px]">{title}</div>
              <div className="text-[11px] text-muted">{meta}</div>
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
