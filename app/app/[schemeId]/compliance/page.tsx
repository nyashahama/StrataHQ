'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import { useAuth } from '@/lib/auth'
import { getComplianceDashboard } from '@/lib/compliance-api'
import {
  COMPLIANCE_CATEGORIES,
  type ComplianceCategory,
  type ComplianceDashboard,
  type ComplianceItem,
  type ComplianceStatus,
} from '@/lib/compliance'
import { useToast } from '@/lib/toast'

const STATUS_STYLES: Record<ComplianceStatus, string> = {
  compliant: 'bg-green-bg text-green',
  'at-risk': 'bg-yellowbg text-amber',
  'non-compliant': 'bg-red-bg text-red',
}

const STATUS_LABELS: Record<ComplianceStatus, string> = {
  compliant: 'Compliant',
  'at-risk': 'At Risk',
  'non-compliant': 'Non-Compliant',
}

function ScoreRing({ score }: { score: number }) {
  const radius = 42
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (score / 100) * circumference
  const color = score >= 80 ? '#2F855A' : score >= 60 ? '#B7791F' : '#C53030'

  return (
    <svg width="120" height="120" viewBox="0 0 100 100" className="flex-shrink-0">
      <circle
        cx="50"
        cy="50"
        r={radius}
        fill="none"
        stroke="currentColor"
        strokeWidth="8"
        className="text-border"
      />
      <circle
        cx="50"
        cy="50"
        r={radius}
        fill="none"
        stroke={color}
        strokeWidth="8"
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        transform="rotate(-90 50 50)"
        style={{ transition: 'stroke-dashoffset 0.6s ease' }}
      />
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
        onClick={() => setExpanded(current => !current)}
      >
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={`w-2 h-2 rounded-full flex-shrink-0 ${
              item.status === 'compliant'
                ? 'bg-green'
                : item.status === 'at-risk'
                  ? 'bg-amber'
                  : 'bg-red'
            }`}
          />
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
          <span className={`text-muted transition-transform ${expanded ? 'rotate-180' : ''}`}>▼</span>
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

function buildReport(dashboard: ComplianceDashboard, schemeId: string) {
  const lines = [
    `StrataHQ STSMA Compliance Report`,
    `Scheme: ${schemeId}`,
    `Score: ${dashboard.score}/100`,
    `Compliant: ${dashboard.compliant_count}`,
    `At Risk: ${dashboard.at_risk_count}`,
    `Non-Compliant: ${dashboard.non_compliant_count}`,
    `Last Assessed: ${new Date(dashboard.last_assessed_at).toLocaleString('en-ZA')}`,
    '',
  ]

  for (const item of dashboard.items) {
    lines.push(`[${item.category.toUpperCase()}] ${item.title}`)
    lines.push(`Status: ${STATUS_LABELS[item.status]}`)
    lines.push(`Requirement: ${item.requirement}`)
    lines.push(`Current Status: ${item.detail}`)
    lines.push(`Action: ${item.action}`)
    lines.push(`Due Date: ${item.due_date ?? 'N/A'}`)
    lines.push('')
  }

  return lines.join('\n')
}

export default function CompliancePage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<ComplianceDashboard | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (user?.role === 'resident') {
      setLoading(false)
      return
    }

    async function load() {
      try {
        setLoading(true)
        setDashboard(await getComplianceDashboard(schemeId))
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load compliance dashboard',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId, user?.role])

  const groupedItems = useMemo(() => {
    if (!dashboard) return new Map<ComplianceCategory, ComplianceItem[]>()

    return COMPLIANCE_CATEGORIES.reduce((map, category) => {
      map.set(
        category.key,
        dashboard.items.filter(item => item.category === category.key),
      )
      return map
    }, new Map<ComplianceCategory, ComplianceItem[]>())
  }, [dashboard])

  if (user?.role === 'resident') {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Compliance</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Compliance</h1>
        <p className="text-[14px] text-muted">Compliance information is managed by your scheme&apos;s trustees and managing agent.</p>
      </div>
    )
  }

  if (loading || !dashboard) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading compliance…
        </div>
      </div>
    )
  }

  const currentDashboard = dashboard
  const scoreLabel = dashboard.score >= 80 ? 'Good standing' : dashboard.score >= 60 ? 'Needs attention' : 'Action required'
  const scoreLabelColor = dashboard.score >= 80 ? 'text-green' : dashboard.score >= 60 ? 'text-amber' : 'text-red'

  function exportReport() {
    const blob = new Blob([buildReport(currentDashboard, schemeId)], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `compliance-report-${schemeId}.txt`
    link.click()
    URL.revokeObjectURL(url)
    addToast('Compliance report exported.', 'success')
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › STSMA Compliance</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">STSMA Compliance</h1>
      <p className="text-[14px] text-muted mb-8">
        Compliance status under the Sectional Titles Schemes Management Act.
      </p>

      <div className="bg-surface border border-border rounded-lg px-6 py-5 mb-6 flex items-center gap-6">
        <ScoreRing score={currentDashboard.score} />
        <div className="flex-1 min-w-0">
          <p className={`font-serif text-[22px] font-semibold mb-1 ${scoreLabelColor}`}>{scoreLabel}</p>
          <p className="text-[13px] text-muted mb-4">
            This scheme scores {currentDashboard.score}/100 based on {currentDashboard.total} STSMA compliance items.
          </p>
          <div className="flex gap-4">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-green" />
              <span className="text-[12px] text-muted">{currentDashboard.compliant_count} compliant</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-amber" />
              <span className="text-[12px] text-muted">{currentDashboard.at_risk_count} at risk</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-red" />
              <span className="text-[12px] text-muted">{currentDashboard.non_compliant_count} non-compliant</span>
            </div>
          </div>
        </div>
        <button
          onClick={exportReport}
          className="flex-shrink-0 text-[12px] text-accent font-medium border border-accent rounded-lg px-4 py-2 hover:bg-accent-dim transition-colors hidden sm:block"
        >
          Export report
        </button>
      </div>

      {currentDashboard.non_compliant_count > 0 && (
        <div className="bg-red-bg border border-red rounded-lg px-5 py-4 mb-6">
          <p className="text-[13px] font-semibold text-red">
            {currentDashboard.non_compliant_count} item{currentDashboard.non_compliant_count > 1 ? 's' : ''} require immediate attention
          </p>
          <p className="text-[12px] text-red/80 mt-0.5">
            Non-compliant items expose the body corporate and trustees to avoidable compliance risk under the STSMA.
          </p>
        </div>
      )}

      <div className="space-y-5">
        {COMPLIANCE_CATEGORIES.map(({ key, label }) => {
          const items = groupedItems.get(key) ?? []
          return (
            <div key={key} className="bg-surface border border-border rounded-lg overflow-hidden">
              <div className="px-5 py-3 border-b border-border flex items-center justify-between">
                <span className="text-[13px] font-semibold text-ink">{label}</span>
                <div className="flex gap-1.5">
                  {items.map(item => (
                    <div
                      key={item.id}
                      className={`w-2 h-2 rounded-full ${
                        item.status === 'compliant'
                          ? 'bg-green'
                          : item.status === 'at-risk'
                            ? 'bg-amber'
                            : 'bg-red'
                      }`}
                      title={item.title}
                    />
                  ))}
                </div>
              </div>
              {items.length === 0 ? (
                <div className="px-5 py-6 text-[13px] text-muted">No items recorded for this category.</div>
              ) : items.map(item => (
                <ComplianceItemRow key={item.id} item={item} />
              ))}
            </div>
          )
        })}
      </div>

      <p className="text-[11px] text-muted text-center mt-6">
        Last assessed: {new Date(currentDashboard.last_assessed_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })} · Powered by StrataHQ · Not legal advice
      </p>
    </div>
  )
}
