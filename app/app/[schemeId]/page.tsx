'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'

import { useAuth } from '@/lib/auth'
import { getScheme, type SchemeDetail } from '@/lib/scheme-api'
import { useToast } from '@/lib/toast'

const HEALTH_STYLES = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-amber',
  poor: 'bg-red-bg text-red',
} as const

function formatDate(value: string) {
  return new Date(value).toLocaleDateString('en-ZA', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

export default function SchemeOverviewPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string
  const [scheme, setScheme] = useState<SchemeDetail | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      try {
        setScheme(await getScheme(schemeId))
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load scheme',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  const isResident = user?.role === 'resident'
  const unitLabel = scheme?.unit_identifier
    ? `Unit ${scheme.unit_identifier}`
    : 'No unit linked yet'

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading scheme overview…
        </div>
      </div>
    )
  }

  if (!scheme) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Scheme details could not be loaded.
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        {scheme.name}
      </h1>
      <p className="text-[14px] text-muted mb-8">
        {isResident ? `${unitLabel} · Welcome back.` : 'Scheme at a glance.'}
      </p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 mb-8">
        {(isResident
          ? [
              { label: 'My unit', value: scheme.unit_identifier ?? '—' },
              { label: 'Scheme members', value: String(scheme.total_members) },
              { label: 'Open maintenance', value: String(scheme.open_maintenance_count) },
              { label: 'Days to AGM', value: scheme.days_to_agm != null ? String(scheme.days_to_agm) : '—' },
            ]
          : [
              { label: 'Total units', value: String(scheme.unit_count) },
              { label: 'Linked members', value: String(scheme.total_members) },
              { label: 'Open maintenance', value: String(scheme.open_maintenance_count) },
              { label: 'Collection rate', value: `${scheme.levy_collection_pct}%` },
            ]).map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-[1.3fr_0.9fr] gap-4 md:gap-6">
        <div>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-[13px] font-semibold text-ink">Unit register</h2>
            {!isResident && (
              <Link href={`/app/${scheme.id}/settings`} className="text-[12px] text-accent font-medium">
                Manage →
              </Link>
            )}
          </div>
          <div className="bg-surface border border-border rounded-lg overflow-hidden">
            {scheme.units.slice(0, isResident ? 1 : 5).map((unit, index) => (
              <div key={unit.id} className={`px-5 py-3 ${index < Math.min(scheme.units.length, isResident ? 1 : 5) - 1 ? 'border-b border-border' : ''}`}>
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <div className="text-[13px] font-medium text-ink">Unit {unit.identifier}</div>
                    <div className="text-[11px] text-muted mt-0.5">{unit.owner_name}</div>
                  </div>
                  <div className="text-right">
                    <div className="text-[12px] text-ink">Floor {unit.floor}</div>
                    <div className="text-[11px] text-muted">{unit.section_value_pct.toFixed(2)}% PQ</div>
                  </div>
                </div>
              </div>
            ))}
            {scheme.units.length === 0 && (
              <div className="px-5 py-8 text-center text-[13px] text-muted">
                No units have been captured for this scheme yet.
              </div>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-4">
          <div className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="flex items-center justify-between gap-3 mb-3">
              <div>
                <p className="text-[12px] text-muted">Scheme health</p>
                <p className="text-[14px] font-semibold text-ink mt-0.5">
                  {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
                </p>
              </div>
              <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                {scheme.health}
              </span>
            </div>
            <div className="text-[12px] text-muted leading-5">
              {scheme.trustee_count} trustee memberships, {scheme.resident_count} resident memberships, and {scheme.notice_count} published notices.
            </div>
          </div>

          <div className="bg-surface border border-border rounded-lg px-5 py-4">
            <p className="text-[12px] text-muted mb-1">Next AGM</p>
            <p className="text-[14px] font-semibold text-ink">
              {scheme.next_agm_date ? formatDate(scheme.next_agm_date) : 'Not scheduled'}
            </p>
            <p className="text-[12px] text-muted mt-1">
              {scheme.days_to_agm != null ? `${scheme.days_to_agm} days remaining` : 'Add an AGM in the AGM module when ready.'}
            </p>
          </div>

          <div className="bg-surface border border-border rounded-lg overflow-hidden">
            <div className="px-5 py-3 border-b border-border">
              <span className="text-[13px] font-semibold text-ink">Recent notices</span>
            </div>
            {scheme.recent_notices.length === 0 ? (
              <div className="px-5 py-8 text-center text-[13px] text-muted">
                No notices have been published yet.
              </div>
            ) : (
              scheme.recent_notices.map((notice, index) => (
                <div key={notice.id} className={`px-5 py-3 ${index < scheme.recent_notices.length - 1 ? 'border-b border-border' : ''}`}>
                  <div className="text-[13px] font-medium text-ink">{notice.title}</div>
                  <div className="text-[11px] text-muted mt-0.5">
                    {notice.type} · {formatDate(notice.sent_at)}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
