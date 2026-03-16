const timeline = [
  {
    time: 'Mon 7:14 AM',
    icon: '📱',
    color: 'bg-red-bg border-[rgba(155,44,44,0.15)] text-red',
    dotColor: 'bg-red',
    event: '47 unread WhatsApp messages',
    detail: 'Residents asking about the burst pipe, the parking gate, the AGM date, three levy queries. No way to triage. No audit trail.',
  },
  {
    time: 'Tue 10:40 AM',
    icon: '📊',
    color: 'bg-yellowbg border-[rgba(146,64,14,0.15)] text-[#92400E]',
    dotColor: 'bg-[#92400E]',
    event: 'Monthly levy run — 3.5 hours in Excel',
    detail: 'EFT references don\'t match. Two units won\'t reconcile. You send statements manually, one by one. Unit 5C is 90 days overdue — you\'ll deal with it next week.',
  },
  {
    time: 'Wed 2:15 PM',
    icon: '📞',
    color: 'bg-yellowbg border-[rgba(146,64,14,0.15)] text-[#92400E]',
    dotColor: 'bg-[#92400E]',
    event: 'Trustee calls about the pool pump quote',
    detail: 'You submitted it three weeks ago. They\'ve had no visibility. You spend 20 minutes on the phone explaining what you emailed. Again.',
  },
  {
    time: 'Thu 4:00 PM',
    icon: '🗳️',
    color: 'bg-accent-bg border-[rgba(43,108,176,0.15)] text-accent',
    dotColor: 'bg-accent',
    event: 'AGM prep — proxy forms via WhatsApp photos',
    detail: 'Counting proxies from blurry screenshots. Someone disputes the quorum. You don\'t have a signed audit trail. This is a legal liability.',
  },
]

export default function ProblemSection() {
  return (
    <section id="features" className="padding-section">
      <div className="max-w-container mx-auto px-container">

        <div className="grid grid-cols-1 md:grid-cols-[360px_1fr] gap-[clamp(40px,6vw,80px)] items-start">

          {/* Left */}
          <div>
            <p className="eyebrow reveal text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
              The problem
            </p>
            <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[560px]">
              A typical week without StrataHQ.
            </h2>
            <p className="reveal text-clamp-p text-ink-2 leading-[1.7] mb-6">
              Not a worst-case scenario. Just Tuesday.
            </p>
            <div className="reveal hidden md:block">
              <div className="bg-page border border-border rounded-lg p-4 mt-2">
                <div className="text-[11px] font-semibold text-muted uppercase tracking-[0.08em] mb-3">
                  Hours lost per month (avg)
                </div>
                {[
                  { label: 'Levy reconciliation', hours: 14, max: 20 },
                  { label: 'Maintenance follow-ups', hours: 8, max: 20 },
                  { label: 'AGM admin', hours: 6, max: 20 },
                  { label: 'Trustee reporting', hours: 7, max: 20 },
                ].map(({ label, hours, max }) => (
                  <div key={label} className="mb-3 last:mb-0">
                    <div className="flex justify-between text-[12px] text-ink-2 mb-[5px]">
                      <span>{label}</span>
                      <span className="font-semibold text-ink">{hours}h</span>
                    </div>
                    <div className="h-[5px] bg-border rounded-full overflow-hidden">
                      <div
                        className="h-full rounded-full bg-ink"
                        style={{ width: `${(hours / max) * 100}%` }}
                      />
                    </div>
                  </div>
                ))}
                <div className="mt-4 pt-3 border-t border-border text-[12px] text-muted">
                  <span className="font-semibold text-ink">35 hours/month</span> of recoverable admin — per managing agent.
                </div>
              </div>
            </div>
          </div>

          {/* Timeline */}
          <div className="stagger relative">
            {/* Vertical line */}
            <div className="absolute left-[19px] top-6 bottom-6 w-[1px] bg-border hidden sm:block" />

            <div className="flex flex-col gap-[2px]">
              {timeline.map(({ time, icon, color, dotColor, event, detail }) => (
                <div
                  key={time}
                  className="flex gap-4 px-[18px] py-4 rounded-lg hover:bg-[rgba(55,53,47,0.03)] transition-colors duration-150 cursor-default"
                >
                  {/* Dot */}
                  <div className="relative flex-shrink-0 pt-[3px] hidden sm:block">
                    <div className={`w-[9px] h-[9px] rounded-full ${dotColor} mt-[5px]`} />
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2 mb-2">
                      <span className={`inline-flex items-center gap-[6px] text-[11px] font-semibold px-2 py-[3px] rounded border ${color}`}>
                        <span>{icon}</span>
                        <span className="font-mono tracking-[0.02em]">{time}</span>
                      </span>
                    </div>
                    <div className="text-[14px] font-semibold text-ink mb-[4px]">{event}</div>
                    <div className="text-[13px] text-muted leading-[1.55]">{detail}</div>
                  </div>
                </div>
              ))}

              {/* Footer note */}
              <div className="flex gap-4 px-[18px] py-4">
                <div className="flex-shrink-0 hidden sm:block w-[9px]" />
                <div className="text-[13px] text-muted italic border-t border-border pt-4 w-full">
                  This is the job right now. StrataHQ doesn&apos;t change what you manage —
                  it changes how much of it needs you.
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
