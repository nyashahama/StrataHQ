'use client'
import { useState } from 'react'
import { useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'
import {
  mockComplianceItems,
  calcComplianceScore,
  COMPLIANCE_CATEGORIES,
  type ComplianceItem,
  type ComplianceStatus,
  type ComplianceCategory,
} from '@/lib/mock/compliance'

const STATUS_STYLES: Record<ComplianceStatus, string> = {
  compliant:       'bg-green-bg text-green',
  'at-risk':       'bg-yellowbg text-amber',
  'non-compliant': 'bg-red-bg text-red',
}

const STATUS_LABELS: Record<ComplianceStatus, string> = {
  compliant:       'Compliant',
  'at-risk':       'At Risk',
  'non-compliant': 'Non-Compliant',
}

const CATEGORY_LABELS: Record<ComplianceCategory, string> = {
  financial:      'Financial',
  governance:     'Governance',
  administrative: 'Administrative',
  insurance:      'Insurance',
}

function ScoreRing({ score }: { score: number }) {
  const radius = 42
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (score / 100) * circumference
  const color = score >= 80 ? '#2F855A' : score >= 60 ? '#B7791F' : '#C53030'

  return (
    <svg width="120" height="120" viewBox="0 0 100 100" className="flex-shrink-0">
      {/* Track */}
      <circle
        cx="50" cy="50" r={radius}
        fill="none"
        stroke="currentColor"
        strokeWidth="8"
        className="text-border"
      />
      {/* Progress */}
      <circle
        cx="50" cy="50" r={radius}
        fill="none"
        stroke={color}
        strokeWidth="8"
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        transform="rotate(-90 50 50)"
        style={{ transition: 'stroke-dashoffset 0.6s ease' }}
      />
      {/* Score text */}
      <text x="50" y="45" textAnchor="middle" fontSize="22" fontWeight="700" fill={color} fontFamily="Georgia, serif">
        {score}
      </text>
      <text x="50" y="60" textAnchor="middle" fontSize="10" fill="#A8A49E">
        / 100
      </text>
    </svg>
  )
}

function ComplianceItemRow({ item }: { item: ComplianceItem }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className={`border-b border-border last:border-b-0 ${item.status === 'non-compliant' ? 'bg-red-bg/30' : ''}`}>
      <button
        className="w-full flex items-center justify-between px-5 py-3 text-left hover:bg-hover-subtle transition-colors"
        onClick={() => setExpanded(e => !e)}
      >
        <div className="flex items-center gap-3 min-w-0">
          {/* Status dot */}
          <div className={`w-2 h-2 rounded-full flex-shrink-0 ${
            item.status === 'compliant' ? 'bg-green' :
            item.status === 'at-risk' ? 'bg-amber' : 'bg-red'
          }`} />
          <span className="text-[13px] font-medium text-ink truncate">{item.title}</span>
        </div>
        <div className="flex items-center gap-3 flex-shrink-0 ml-3">
          {item.due_date && item.status !== 'compliant' && (
            <span className="text-[11px] text-muted hidden sm:block">
              Due {new Date(item.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short' })}
            </span>
          )}
          <span className={`text-[11px] font-semibold px-2 py-[3px] rounded-full ${STATUS_STYLES[item.status]}`}>
            {STATUS_LABELS[item.status]}
          </span>
          <svg
            width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5"
            className={`text-muted transition-transform ${expanded ? 'rotate-180' : ''}`}
          >
            <polyline points="2,4 6,8 10,4" />
          </svg>
        </div>
      </button>

      {expanded && (
        <div className="px-5 pb-4 pt-0">
          <div className="ml-5 space-y-3">
            <div>
              <p className="text-[11px] font-semibold text-muted uppercase tracking-wide mb-1">Requirement</p>
              <p className="text-[13px] text-ink">{item.requirement}</p>
            </div>
            <div>
              <p className="text-[11px] font-semibold text-muted uppercase tracking-wide mb-1">Current status</p>
              <p className="text-[13px] text-ink">{item.detail}</p>
            </div>
            {item.status !== 'compliant' && (
              <div className={`rounded-lg px-4 py-3 ${item.status === 'non-compliant' ? 'bg-red-bg border border-red' : 'bg-yellowbg border border-amber'}`}>
                <p className={`text-[11px] font-semibold uppercase tracking-wide mb-1 ${item.status === 'non-compliant' ? 'text-red' : 'text-amber'}`}>
                  Action required
                </p>
                <p className={`text-[13px] ${item.status === 'non-compliant' ? 'text-red' : 'text-amber'}`}>
                  {item.action}
                </p>
                {item.due_date && (
                  <p className={`text-[11px] mt-1 font-medium ${item.status === 'non-compliant' ? 'text-red/70' : 'text-amber/70'}`}>
                    Recommended by: {new Date(item.due_date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
                  </p>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default function CompliancePage() {
  const { user } = useAuth()
  const { addToast } = useToast()

  // Compliance is for agents and trustees only
  if (user?.role === 'resident') {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Compliance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Compliance</h1>
        <p className="text-[14px] text-muted">Compliance information is managed by your scheme's trustees and managing agent.</p>
      </div>
    )
  }

  const score = calcComplianceScore(mockComplianceItems)
  const scoreLabel = score >= 80 ? 'Good standing' : score >= 60 ? 'Needs attention' : 'Action required'
  const scoreLabelColor = score >= 80 ? 'text-green' : score >= 60 ? 'text-amber' : 'text-red'

  const compliantCount     = mockComplianceItems.filter(i => i.status === 'compliant').length
  const atRiskCount        = mockComplianceItems.filter(i => i.status === 'at-risk').length
  const nonCompliantCount  = mockComplianceItems.filter(i => i.status === 'non-compliant').length

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › STSMA Compliance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">STSMA Compliance</h1>
      <p className="text-[14px] text-muted mb-8">
        Compliance status under the Sectional Titles Schemes Management Act.
      </p>

      {/* Score + summary */}
      <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6 flex items-center gap-6">
        <ScoreRing score={score} />
        <div className="flex-1 min-w-0">
          <p className={`font-serif text-[22px] font-semibold mb-1 ${scoreLabelColor}`}>{scoreLabel}</p>
          <p className="text-[13px] text-muted mb-4">
            Sunridge Heights scores {score}/100 based on {mockComplianceItems.length} STSMA compliance items.
          </p>
          <div className="flex gap-4">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-green" />
              <span className="text-[12px] text-muted">{compliantCount} compliant</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-amber" />
              <span className="text-[12px] text-muted">{atRiskCount} at risk</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-red" />
              <span className="text-[12px] text-muted">{nonCompliantCount} non-compliant</span>
            </div>
          </div>
        </div>
        <button
          onClick={() => addToast('Compliance report generated — check your downloads folder.', 'info')}
          className="flex-shrink-0 text-[12px] text-accent font-medium border border-accent rounded-lg px-4 py-2 hover:bg-accent-dim transition-colors hidden sm:block"
        >
          Export report
        </button>
      </div>

      {/* Non-compliant banner */}
      {nonCompliantCount > 0 && (
        <div className="bg-red-bg border border-red rounded-lg px-5 py-4 mb-6 flex items-start gap-3">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-red flex-shrink-0 mt-0.5">
            <circle cx="8" cy="8" r="7" />
            <line x1="8" y1="5" x2="8" y2="8.5" />
            <circle cx="8" cy="11" r="0.5" fill="currentColor" />
          </svg>
          <div>
            <p className="text-[13px] font-semibold text-red">
              {nonCompliantCount} item{nonCompliantCount > 1 ? 's' : ''} require immediate attention
            </p>
            <p className="text-[12px] text-red/80 mt-0.5">
              Non-compliant items expose the body corporate and trustees to personal liability under the STSMA.
            </p>
          </div>
        </div>
      )}

      {/* Items by category */}
      <div className="space-y-5">
        {COMPLIANCE_CATEGORIES.map(({ key, label }) => {
          const items = mockComplianceItems.filter(i => i.category === key)
          const hasIssues = items.some(i => i.status !== 'compliant')

          return (
            <div key={key} className="bg-surface border border-border rounded-lg overflow-hidden">
              <div className="px-5 py-3 border-b border-border flex items-center justify-between">
                <span className="text-[13px] font-semibold text-ink">{label}</span>
                <div className="flex gap-1.5">
                  {items.map(item => (
                    <div
                      key={item.id}
                      className={`w-2 h-2 rounded-full ${
                        item.status === 'compliant' ? 'bg-green' :
                        item.status === 'at-risk' ? 'bg-amber' : 'bg-red'
                      }`}
                      title={item.title}
                    />
                  ))}
                </div>
              </div>
              {items.map(item => (
                <ComplianceItemRow key={item.id} item={item} />
              ))}
            </div>
          )
        })}
      </div>

      <p className="text-[11px] text-muted text-center mt-6">
        Last assessed: 28 October 2025 · Powered by StrataHQ · Not legal advice
      </p>
    </div>
  )
}
