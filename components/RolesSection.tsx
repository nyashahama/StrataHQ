'use client'

import { useState } from 'react'

interface RoleFeature {
  icon: string
  title: string
  desc: string
}

interface Role {
  id: string
  label: string
  icon: string
  heading: string
  sub: string
  features: RoleFeature[]
}

const roles: Role[] = [
  {
    id: 'agents',
    label: 'Managing Agents',
    icon: '🏢',
    heading: 'Run your full portfolio from one place.',
    sub: "See every scheme's levy status, open maintenance and upcoming AGMs from a single dashboard. Stop switching between spreadsheets.",
    features: [
      { icon: '⚡', title: 'Portfolio dashboard', desc: 'All schemes, arrears and open work orders at a glance.' },
      { icon: '🔄', title: 'Automated workflows', desc: 'Levy reminders, maintenance escalation and AGM notices run themselves.' },
      { icon: '📋', title: 'Trustee-ready reports', desc: 'STSMA-compliant financials and management packs in minutes.' },
      { icon: '👥', title: 'Team access control', desc: 'Assign staff to schemes with role-based permissions.' },
    ],
  },
  {
    id: 'trustees',
    label: 'Trustees',
    icon: '👤',
    heading: 'Govern with confidence.',
    sub: "See your scheme's finances, maintenance backlog and compliance status at any time — without emailing the managing agent.",
    features: [
      { icon: '👁️', title: 'Real-time visibility', desc: 'Live financials, arrears and open maintenance — always current.' },
      { icon: '✅', title: 'Approve digitally', desc: 'Quote approvals, expense sign-offs and trustee votes from your phone.' },
      { icon: '📣', title: 'Official communications', desc: "Send verified notices residents trust — not another WhatsApp message." },
      { icon: '📅', title: 'AGM management', desc: 'Schedule meetings, review proxies and monitor quorum in real time.' },
    ],
  },
  {
    id: 'residents',
    label: 'Residents',
    icon: '🏠',
    heading: "Know what's happening in your scheme.",
    sub: 'Pay your levy, log a maintenance request and stay informed — all from your phone. No app store required.',
    features: [
      { icon: '💳', title: 'Pay levies instantly', desc: 'See your balance, download statements and pay in seconds.' },
      { icon: '📸', title: 'Log issues with photos', desc: 'Submit maintenance requests and track them to resolution.' },
      { icon: '🔔', title: 'Official notices', desc: 'AGM dates, bylaw changes and emergency alerts — direct to you.' },
      { icon: '📁', title: 'Scheme documents', desc: 'Rules, insurance and minutes — always accessible, always current.' },
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
                className={`w-full text-left px-[14px] py-[9px] rounded border-none text-[14px] cursor-pointer flex items-center gap-[10px] transition-colors duration-150
                  ${activeId === role.id
                    ? 'bg-[rgba(55,53,47,0.06)] text-ink font-medium'
                    : 'bg-transparent text-muted font-normal hover:bg-[rgba(55,53,47,0.06)] hover:text-ink'
                  }`}
              >
                <span className="text-[15px]">{role.icon}</span>
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
              {active.features.map(({ icon, title, desc }) => (
                <div
                  key={title}
                  className="bg-white border border-border rounded p-4 hover:border-border-2 hover:shadow-sm transition-all duration-150"
                >
                  <span className="text-[18px] mb-2 block">{icon}</span>
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
