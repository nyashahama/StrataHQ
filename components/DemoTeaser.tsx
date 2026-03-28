import Link from 'next/link'

const roles = [
  {
    icon: '🏢',
    label: 'Managing Agent',
    color: 'bg-[rgba(43,108,176,0.06)] border-[rgba(43,108,176,0.15)]',
    dot: 'bg-accent',
    desc: 'Portfolio, levies, maintenance, AGMs',
  },
  {
    icon: '👤',
    label: 'Trustee',
    color: 'bg-[rgba(39,103,73,0.06)] border-[rgba(39,103,73,0.15)]',
    dot: 'bg-green',
    desc: 'Oversight, approvals, live voting',
  },
  {
    icon: '🏠',
    label: 'Resident',
    color: 'bg-[rgba(55,53,47,0.04)] border-border',
    dot: 'bg-muted',
    desc: 'Levy payments, notices, requests',
  },
]

export default function DemoTeaser() {
  return (
    <section className="padding-section bg-surface border-t border-border">
      <div className="max-w-container mx-auto px-container">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-[clamp(40px,6vw,80px)] items-center">

          {/* Left: copy */}
          <div>
            <div className="inline-flex items-center gap-2 text-[12px] font-semibold text-accent bg-accent-bg border border-[rgba(43,108,176,0.18)] rounded-full px-3 py-1 mb-6 tracking-[0.02em]">
              <span className="relative flex h-2 w-2 flex-shrink-0">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent opacity-60" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-accent" />
              </span>
              Interactive demo — live now
            </div>

            <h2 className="font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-4 max-w-[520px]">
              See the platform in action — no sign-up needed.
            </h2>
            <p className="text-clamp-p text-ink-2 leading-[1.7] mb-8 max-w-[480px]">
              Pick a role and explore a fully working demo with realistic data. Every button does something. Switch roles any time to see how each stakeholder experiences the platform.
            </p>

            {/* Role chips */}
            <div className="flex flex-wrap gap-2 mb-8">
              {roles.map(({ icon, label, color, dot, desc }) => (
                <div
                  key={label}
                  className={`flex items-center gap-2 px-3 py-2 rounded-lg border text-[13px] ${color}`}
                >
                  <span>{icon}</span>
                  <div>
                    <div className="font-semibold text-ink leading-none mb-[2px]">{label}</div>
                    <div className="text-[11px] text-muted">{desc}</div>
                  </div>
                </div>
              ))}
            </div>

            <Link
              href="/demo"
              className="inline-flex items-center gap-2 px-7 py-3 text-[15px] font-medium text-white bg-ink
                border border-ink rounded hover:bg-ink-2 transition-colors duration-150 no-underline"
            >
              Open interactive demo →
            </Link>
            <p className="mt-3 text-[12px] text-muted-2">Opens in-browser · No app install · Switch roles any time</p>
          </div>

          {/* Right: faux app preview */}
          <div className="reveal relative">
            {/* Browser chrome */}
            <div className="bg-surface border border-border rounded-lg shadow-lg overflow-hidden">
              {/* Title bar */}
              <div className="flex items-center gap-[6px] px-3 py-[10px] border-b border-border bg-page">
                <span className="w-[10px] h-[10px] rounded-full bg-[#FF5F57]" />
                <span className="w-[10px] h-[10px] rounded-full bg-[#FEBC2E]" />
                <span className="w-[10px] h-[10px] rounded-full bg-[#28C840]" />
                <div className="flex-1 mx-3">
                  <div className="h-[20px] bg-surface border border-border rounded flex items-center px-2 gap-2">
                    <div className="w-2 h-2 rounded-full bg-green flex-shrink-0" />
                    <span className="text-[10px] text-muted font-mono">stratahq.co.za/demo</span>
                  </div>
                </div>
              </div>

              {/* App shell preview */}
              <div className="flex" style={{ height: 280 }}>
                {/* Sidebar */}
                <div className="w-[140px] flex-shrink-0 bg-surface border-r border-border flex flex-col py-2 px-1.5 gap-[1px]">
                  <div className="flex items-center gap-1.5 px-2 py-1.5 mb-1">
                    <div className="w-[18px] h-[18px] bg-ink rounded-[3px] grid place-items-center flex-shrink-0">
                      <svg viewBox="0 0 16 16" className="w-[9px] h-[9px] fill-white"><path d="M2 3h5v5H2V3zm7 0h5v5H9V3zM2 10h5v4H2v-4zm7 1h2v-2h2v2h2v2h-2v2H9v-2H7v-2h2v-2z"/></svg>
                    </div>
                    <span className="text-[11px] font-semibold text-ink">StrataHQ</span>
                  </div>
                  {[
                    { icon: '◆', label: 'Dashboard', active: true },
                    { icon: '💳', label: 'Levies', badge: '2' },
                    { icon: '🔧', label: 'Maintenance', badge: '4' },
                    { icon: '📢', label: 'Comms' },
                    { icon: '🗳️', label: 'AGM & Voting' },
                    { icon: '📊', label: 'Financials' },
                  ].map(({ icon, label, active, badge }) => (
                    <div
                      key={label}
                      className={`flex items-center gap-1.5 px-2 py-[5px] rounded text-[11px] ${active ? 'bg-[rgba(55,53,47,0.07)] text-ink font-medium' : 'text-ink-2'}`}
                    >
                      <span className="text-[11px] w-[14px] text-center flex-shrink-0">{icon}</span>
                      <span className="flex-1 truncate">{label}</span>
                      {badge && (
                        <span className="text-[9px] font-bold bg-[#EBF2FA] text-accent px-[5px] py-[1px] rounded-full">{badge}</span>
                      )}
                    </div>
                  ))}
                </div>

                {/* Main content preview */}
                <div className="flex-1 bg-page p-3 overflow-hidden flex flex-col gap-2">
                  {/* Stat row */}
                  <div className="grid grid-cols-4 gap-2">
                    {[
                      { label: 'Schemes', val: '12' },
                      { label: 'Collection', val: '94%' },
                      { label: 'Open Jobs', val: '4' },
                      { label: 'Arrears', val: 'R 14K' },
                    ].map(({ label, val }) => (
                      <div key={label} className="bg-surface border border-border rounded-md p-2">
                        <div className="text-[8px] font-semibold text-muted uppercase tracking-[0.06em] mb-1">{label}</div>
                        <div className="text-[14px] font-bold text-ink font-serif leading-none">{val}</div>
                      </div>
                    ))}
                  </div>

                  {/* Mini table */}
                  <div className="flex-1 bg-surface border border-border rounded-md overflow-hidden">
                    <div className="px-3 py-2 border-b border-border flex items-center justify-between">
                      <span className="text-[10px] font-semibold text-ink">Schemes Overview</span>
                      <span className="text-[9px] px-1.5 py-[1px] rounded-full bg-page border border-border text-muted font-medium">12 active</span>
                    </div>
                    {[
                      { icon: '🌿', name: 'Rosewood Estate', tag: '94%', t: 'text-green bg-green-bg' },
                      { icon: '🏙️', name: 'Parkview Towers', tag: '89%', t: 'text-amber bg-yellowbg' },
                      { icon: '🌊', name: 'Seapoint Villas', tag: '97%', t: 'text-green bg-green-bg' },
                    ].map(({ icon, name, tag, t }) => (
                      <div key={name} className="flex items-center gap-2 px-3 py-[6px] border-b border-border last:border-b-0">
                        <span className="text-[12px]">{icon}</span>
                        <span className="text-[10px] text-ink flex-1 truncate">{name}</span>
                        <span className={`text-[9px] font-semibold px-1.5 py-[1px] rounded-full ${t}`}>{tag}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>

            {/* Floating role badge */}
            <div className="absolute -bottom-3 -right-3 bg-green text-white text-[11px] font-semibold px-3 py-1.5 rounded-full shadow-lg flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-surface opacity-80" />
              3 roles to explore
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
