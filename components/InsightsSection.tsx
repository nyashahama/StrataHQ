'use client'

import { useState } from 'react'

const alerts = [
  {
    icon: '⚖️',
    color: 'bg-red-bg border-[rgba(155,44,44,0.15)]',
    label: 'ARREARS INTELLIGENCE',
    labelColor: 'text-red',
    title: 'Unit 5C · Rosewood Estate',
    body: 'R 9,300 outstanding · 90+ days · Payment probability 9%. Attorney referral window closing in ~10 days.',
    action: 'Refer to attorney →',
    actionColor: 'bg-red-bg border-[rgba(155,44,44,0.25)] text-red hover:bg-[rgba(155,44,44,0.12)]',
  },
  {
    icon: '📉',
    color: 'bg-yellowbg border-[rgba(146,64,14,0.15)]',
    label: 'COLLECTION ALERT',
    labelColor: 'text-amber',
    title: 'Parkview Towers — collection dipped',
    body: '89% collected vs 94% last month. 7 units unpaid. Automated reminders not yet sent — act now.',
    action: 'Send reminders →',
    actionColor: 'bg-accent-bg border-[rgba(43,108,176,0.2)] text-accent hover:bg-accent-dim',
  },
  {
    icon: '📅',
    color: 'bg-accent-bg border-[rgba(43,108,176,0.15)]',
    label: 'RESERVE FUND FORECAST',
    labelColor: 'text-accent',
    title: 'Seapoint Villas — fund depletion risk',
    body: 'At current spend, reserve fund depletes in 7.4 years. Recommend R 120/unit/month increase to meet 10-year plan.',
    action: 'Model scenarios →',
    actionColor: 'bg-page border-border-2 text-ink-2 hover:bg-hover-subtle',
  },
]

function HealthRing({ score, color }: { score: number; color: string }) {
  const r = 36
  const circ = 2 * Math.PI * r
  const dash = (score / 100) * circ
  const offset = circ * 0.25
  return (
    <svg width="96" height="96" viewBox="0 0 96 96">
      <circle cx="48" cy="48" r={r} fill="none" style={{ stroke: 'var(--color-border)' }} strokeWidth="8" />
      <circle
        cx="48" cy="48" r={r}
        fill="none"
        style={{ stroke: color }}
        strokeWidth="8"
        strokeDasharray={`${dash.toFixed(1)} ${(circ - dash).toFixed(1)}`}
        strokeDashoffset={offset.toFixed(1)}
        strokeLinecap="round"
      />
    </svg>
  )
}

const schemes = [
  { name: 'Rosewood Estate',   pct: 94, color: 'var(--color-green)' },
  { name: 'Seapoint Villas',   pct: 97, color: 'var(--color-green)' },
  { name: 'Midrand Heights',   pct: 91, color: 'var(--color-green)' },
  { name: 'Parkview Towers',   pct: 89, color: 'var(--color-amber)' },
  { name: 'Bryanston Gardens', pct: 74, color: 'var(--color-red)'   },
]

function MiniRing({ pct, color }: { pct: number; color: string }) {
  const r = 10
  const circ = 2 * Math.PI * r
  const dash = (pct / 100) * circ
  const offset = circ * 0.25
  return (
    <svg width="28" height="28" viewBox="0 0 28 28" className="flex-shrink-0">
      <circle cx="14" cy="14" r={r} fill="none" style={{ stroke: 'var(--color-border)' }} strokeWidth="3" />
      <circle
        cx="14" cy="14" r={r}
        fill="none"
        style={{ stroke: color }}
        strokeWidth="3"
        strokeDasharray={`${dash.toFixed(1)} ${(circ - dash).toFixed(1)}`}
        strokeDashoffset={offset.toFixed(1)}
        strokeLinecap="round"
      />
      <text
        x="14" y="18"
        textAnchor="middle"
        fontSize="6.5"
        fontWeight="700"
        style={{ fill: color }}
        fontFamily="DM Sans, sans-serif"
      >{pct}</text>
    </svg>
  )
}

export default function InsightsSection() {
  const [activeAlert, setActiveAlert] = useState(0)

  return (
    <section className="padding-section border-t border-border bg-surface">
      <div className="max-w-container mx-auto px-container">

        {/* Header */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-end mb-[clamp(36px,5vw,52px)]">
          <div>
            <p className="reveal eyebrow text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
              StrataHQ Intelligence
            </p>
            <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink max-w-[560px]">
              Doesn&apos;t just store your data.{' '}
              <em className="not-italic text-muted">Reads it for you.</em>
            </h2>
          </div>
          <p className="reveal text-[15px] text-ink-2 leading-[1.7] max-w-[420px]">
            StrataHQ surfaces what needs your attention before it becomes a problem —
            attorney windows, collection dips, reserve fund risk. Not dashboards to interpret.
            Answers.
          </p>
        </div>

        {/* Three panel layout */}
        <div className="stagger grid grid-cols-1 lg:grid-cols-3 gap-4 items-start">

          {/* Panel 1: Portfolio Health Score */}
          <div className="bg-page border border-border rounded-lg p-5">
            <p className="text-[11px] font-semibold tracking-[0.08em] uppercase text-muted mb-4">
              Portfolio Health Score
            </p>

            <div className="flex items-center gap-4 mb-5">
              <div className="relative flex-shrink-0">
                <HealthRing score={87} color="var(--color-green)" />
                <div className="absolute inset-0 flex flex-col items-center justify-center">
                  <span className="font-serif text-[22px] font-bold text-ink leading-none">87</span>
                  <span className="text-[9px] font-semibold text-muted uppercase tracking-[0.05em] mt-[2px]">/ 100</span>
                </div>
              </div>
              <div>
                <div className="text-[15px] font-bold text-green mb-1">Strong</div>
                <div className="text-[12px] text-muted leading-[1.55]">
                  Composite score across levy collection, maintenance SLA, AGM compliance, and reserve health.
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-[6px]">
              {schemes.map(({ name, pct, color }) => (
                <div key={name} className="flex items-center gap-2">
                  <MiniRing pct={pct} color={color} />
                  <span className="text-[12px] text-ink-2 flex-1 truncate">{name}</span>
                  <span className="text-[12px] font-semibold" style={{ color }}>{pct}%</span>
                </div>
              ))}
              <div className="text-[11px] text-muted mt-1">Updated daily · Benchmarked against 2,400+ schemes</div>
            </div>
          </div>

          {/* Panel 2: Predictive Alerts (always-dark navy panel) */}
          <div className="bg-sidebar-header rounded-lg p-5 lg:col-span-1">
            <div className="flex items-center justify-between mb-4">
              <p className="text-[11px] font-semibold tracking-[0.08em] uppercase text-white/50">
                Predictive Alerts
              </p>
              <span className="text-[10px] font-semibold px-2 py-[3px] rounded-full bg-red-bg text-red">
                3 active
              </span>
            </div>

            {/* Alert tabs */}
            <div className="flex gap-1 mb-4">
              {alerts.map((_, i) => (
                <button
                  key={i}
                  onClick={() => setActiveAlert(i)}
                  className="flex-1 py-[10px] border-none bg-transparent cursor-pointer"
                  aria-label={`Alert ${i + 1}`}
                >
                  <span className={`block h-[3px] w-full rounded-full transition-colors duration-150 ${
                    activeAlert === i ? 'bg-white/80' : 'bg-white/20'
                  }`} />
                </button>
              ))}
            </div>

            {/* Active alert */}
            {alerts.map((a, i) => (
              <div
                key={i}
                className={`transition-opacity duration-200 ${activeAlert === i ? 'block' : 'hidden'}`}
              >
                <div className={`inline-flex items-center gap-2 rounded px-2 py-[3px] border mb-3 ${a.color}`}>
                  <span className="text-[12px]">{a.icon}</span>
                  <span className={`text-[10px] font-bold tracking-[0.07em] ${a.labelColor}`}>{a.label}</span>
                </div>
                <div className="text-[14px] font-semibold text-white leading-[1.3] mb-2">{a.title}</div>
                <div className="text-[12px] text-white/60 leading-[1.6] mb-4">{a.body}</div>
                <button
                  className={`text-[12px] font-medium px-3 py-[6px] rounded border transition-colors duration-150 cursor-pointer ${a.actionColor}`}
                  onClick={() => {}}
                >
                  {a.action}
                </button>
              </div>
            ))}

            <div className="mt-5 pt-4 border-t border-white/10">
              <div className="text-[12px] text-white/40 leading-[1.55]">
                Alerts are generated from live scheme data — not static rules. StrataHQ learns from payment patterns, SLA history, and fund trajectories.
              </div>
            </div>
          </div>

          {/* Panel 3: Portfolio Analytics preview */}
          <div className="bg-page border border-border rounded-lg p-5">
            <p className="text-[11px] font-semibold tracking-[0.08em] uppercase text-muted mb-4">
              Portfolio Analytics
            </p>

            <div className="grid grid-cols-2 gap-2 mb-4">
              {[
                { label: 'Avg Collection', val: '94%', delta: '↑ +7pp YoY', up: true },
                { label: 'Total Arrears', val: 'R 61K', delta: '↓ −18% vs Q3', up: true },
                { label: 'SLA On-Time', val: '91%', delta: 'Across all schemes', up: false },
                { label: 'AGMs This Yr', val: '11/12', delta: '1 pending', up: false },
              ].map(({ label, val, delta, up }) => (
                <div key={label} className="bg-surface border border-border rounded p-3">
                  <div className="text-[10px] font-semibold text-muted uppercase tracking-[0.06em] mb-1">{label}</div>
                  <div className="font-serif text-[18px] font-bold text-ink leading-none mb-1">{val}</div>
                  <div className={`text-[11px] ${up ? 'text-green' : 'text-muted'}`}>{delta}</div>
                </div>
              ))}
            </div>

            <div className="bg-surface border border-border rounded p-3">
              <div className="text-[11px] font-semibold text-ink mb-3">Monthly levy collection rate</div>
              <div className="flex items-end gap-[5px] h-[48px]">
                {[87, 89, 88, 91, 92, 94].map((v, i) => {
                  const last = i === 5
                  const heightPct = ((v - 80) / 20) * 100
                  return (
                    <div key={i} className="flex-1 flex flex-col items-center justify-end gap-[2px]">
                      <span className="text-[8px] font-semibold" style={{ color: last ? 'var(--color-accent)' : 'var(--color-muted)' }}>
                        {v}%
                      </span>
                      <div
                        className="w-full rounded-sm"
                        style={{
                          height: `${heightPct}%`,
                          background: last ? 'var(--color-accent)' : 'var(--color-border)',
                          minHeight: 4,
                        }}
                      />
                    </div>
                  )
                })}
              </div>
              <div className="flex justify-between text-[9px] text-muted-2 mt-1">
                {['May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct'].map(m => <span key={m}>{m}</span>)}
              </div>
            </div>
          </div>

        </div>
      </div>
    </section>
  )
}
