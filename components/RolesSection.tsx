'use client'

import { useState } from 'react'

interface RoleFeature {
  iconKey: string
  title: string
  desc: string
}

interface Role {
  id: string
  label: string
  heading: string
  sub: string
  features: RoleFeature[]
}

// Role tab icons — building, person, home
const roleIcons: Record<string, React.ReactNode> = {
  agents: (
    <svg viewBox="0 0 16 16" fill="none" className="w-4 h-4 flex-shrink-0" aria-hidden>
      <rect x="1" y="6" width="14" height="9" rx="1" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M5 15V11h6v4" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
      <path d="M8 1L1 6h14L8 1z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
    </svg>
  ),
  trustees: (
    <svg viewBox="0 0 16 16" fill="none" className="w-4 h-4 flex-shrink-0" aria-hidden>
      <circle cx="8" cy="5" r="3" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M2 14c0-3.314 2.686-6 6-6s6 2.686 6 6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  residents: (
    <svg viewBox="0 0 16 16" fill="none" className="w-4 h-4 flex-shrink-0" aria-hidden>
      <path d="M8 1L1 7v8h5v-4h4v4h5V7L8 1z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
    </svg>
  ),
}

// Feature card icons — minimal strokes
const featureIcons: Record<string, React.ReactNode> = {
  dashboard: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <rect x="1" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.3"/>
      <rect x="9" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.3"/>
      <rect x="1" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.3"/>
      <rect x="9" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.3"/>
    </svg>
  ),
  automation: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M8 1v3M8 12v3M1 8h3M12 8h3M3.1 3.1l2.1 2.1M10.8 10.8l2.1 2.1M3.1 12.9l2.1-2.1M10.8 5.2l2.1-2.1" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
      <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3"/>
    </svg>
  ),
  reports: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <rect x="2" y="1" width="12" height="14" rx="1.5" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M5 6h6M5 9h6M5 12h4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  access: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <circle cx="6" cy="7" r="3" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M9 7h6M12 5v4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  visibility: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M1 8s2.5-5 7-5 7 5 7 5-2.5 5-7 5-7-5-7-5z" stroke="currentColor" strokeWidth="1.3"/>
      <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3"/>
    </svg>
  ),
  approve: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M3 8l3.5 3.5 6.5-7" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  comms: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M14 2H2a1 1 0 00-1 1v8a1 1 0 001 1h2v2.5L7 12h7a1 1 0 001-1V3a1 1 0 00-1-1z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
    </svg>
  ),
  calendar: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <rect x="1" y="3" width="14" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M1 7h14M5 1v4M11 1v4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  payment: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <rect x="1" y="3" width="14" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M1 7h14" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M4 11h3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  photo: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <rect x="1" y="4" width="14" height="10" rx="1.5" stroke="currentColor" strokeWidth="1.3"/>
      <circle cx="8" cy="9" r="2.5" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M5 4l1.5-2h3L11 4" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
    </svg>
  ),
  notice: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M8 1v2M8 13v2M3 3l1.5 1.5M11.5 11.5 13 13M1 8h2M13 8h2M3 13l1.5-1.5M11.5 4.5 13 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
      <circle cx="8" cy="8" r="3" stroke="currentColor" strokeWidth="1.3"/>
    </svg>
  ),
  docs: (
    <svg viewBox="0 0 16 16" fill="none" className="w-[15px] h-[15px]" aria-hidden>
      <path d="M4 1h6l4 4v10H4V1z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
      <path d="M10 1v4h4" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/>
      <path d="M6 9h5M6 12h3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
}

const roles: Role[] = [
  {
    id: 'agents',
    label: 'Managing Agents',
    heading: 'Run your full portfolio from one place.',
    sub: "See every scheme's levy status, open maintenance and upcoming AGMs from a single dashboard. Stop switching between spreadsheets.",
    features: [
      { iconKey: 'dashboard', title: 'Portfolio dashboard', desc: 'All schemes, arrears and open work orders at a glance.' },
      { iconKey: 'automation', title: 'Automated workflows', desc: 'Levy reminders, maintenance escalation and AGM notices run themselves.' },
      { iconKey: 'reports', title: 'Trustee-ready reports', desc: 'STSMA-compliant financials and management packs in minutes.' },
      { iconKey: 'access', title: 'Team access control', desc: 'Assign staff to schemes with role-based permissions.' },
    ],
  },
  {
    id: 'trustees',
    label: 'Trustees',
    heading: 'Govern with confidence.',
    sub: "See your scheme's finances, maintenance backlog and compliance status at any time — without emailing the managing agent.",
    features: [
      { iconKey: 'visibility', title: 'Real-time visibility', desc: 'Live financials, arrears and open maintenance — always current.' },
      { iconKey: 'approve', title: 'Approve digitally', desc: 'Quote approvals, expense sign-offs and trustee votes from your phone.' },
      { iconKey: 'comms', title: 'Official communications', desc: "Send verified notices residents trust — not another WhatsApp message." },
      { iconKey: 'calendar', title: 'AGM management', desc: 'Schedule meetings, review proxies and monitor quorum in real time.' },
    ],
  },
  {
    id: 'residents',
    label: 'Residents',
    heading: "Know what's happening in your scheme.",
    sub: 'Pay your levy, log a maintenance request and stay informed — all from your phone. No app store required.',
    features: [
      { iconKey: 'payment', title: 'Pay levies instantly', desc: 'See your balance, download statements and pay in seconds.' },
      { iconKey: 'photo', title: 'Log issues with photos', desc: 'Submit maintenance requests and track them to resolution.' },
      { iconKey: 'notice', title: 'Official notices', desc: 'AGM dates, bylaw changes and emergency alerts — direct to you.' },
      { iconKey: 'docs', title: 'Scheme documents', desc: 'Rules, insurance and minutes — always accessible, always current.' },
    ],
  },
]

export default function RolesSection() {
  const [activeId, setActiveId] = useState('agents')
  const active = roles.find((r) => r.id === activeId)!

  return (
    <section id="roles" className="padding-section border-t border-border">
      <div className="max-w-container mx-auto px-container">
        <p className="reveal eyebrow text-[12px] font-semibold tracking-[0.1em] uppercase text-muted mb-3">
          Who it&apos;s for
        </p>
        <h2 className="reveal font-serif text-clamp-section font-bold leading-[1.15] tracking-[-0.02em] text-ink mb-[clamp(32px,5vw,48px)] max-w-[620px]">
          Built for everyone in the scheme.
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-[220px_1fr] gap-[clamp(24px,4vw,48px)] items-start">
          {/* Role nav */}
          <div className="reveal flex flex-col gap-[2px] md:sticky md:top-[72px]">
            {roles.map((role) => (
              <button
                key={role.id}
                onClick={() => setActiveId(role.id)}
                className={`w-full text-left px-[14px] py-[11px] rounded border-none text-[14px] cursor-pointer flex items-center gap-[10px] transition-colors duration-150
                  ${activeId === role.id
                    ? 'bg-[rgba(55,53,47,0.06)] text-ink font-medium'
                    : 'bg-transparent text-muted font-normal hover:bg-[rgba(55,53,47,0.06)] hover:text-ink'
                  }`}
              >
                {roleIcons[role.id]}
                {role.label}
              </button>
            ))}
          </div>

          {/* Role content */}
          <div>
            <h3 className="font-serif text-clamp-role font-bold tracking-[-0.015em] text-ink mb-2">
              {active.heading}
            </h3>
            <p className="text-[15px] text-ink-2 leading-[1.65] mb-7">{active.sub}</p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-[10px]">
              {active.features.map(({ iconKey, title, desc }) => (
                <div
                  key={title}
                  className="bg-surface border border-border rounded p-4 hover:border-border-2 hover:shadow-sm transition-all duration-150"
                >
                  <span className="text-ink-2 mb-2 block">{featureIcons[iconKey]}</span>
                  <div className="text-[13px] font-semibold text-ink mb-1">{title}</div>
                  <div className="text-[12px] text-muted leading-[1.55]">{desc}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
