'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'

import { useAuth } from '@/lib/auth'
import { listSchemes, type SchemeSummary } from '@/lib/scheme-api'
import { useToast } from '@/lib/toast'

const HEALTH_STYLES: Record<SchemeSummary['health'], string> = {
  good: 'bg-green-bg text-green',
  fair: 'bg-yellowbg text-amber',
  poor: 'bg-red-bg text-red',
}

export default function SchemesPage() {
  useAuth()
  const { addToast } = useToast()
  const [schemes, setSchemes] = useState<SchemeSummary[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      try {
        setSchemes(await listSchemes())
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load schemes',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast])

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">All schemes</h1>
      <p className="text-[14px] text-muted mb-8">Schemes managed by your organisation.</p>

      {loading ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading schemes…
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {schemes.map(scheme => (
            <div key={scheme.id} className="bg-surface border border-border rounded-lg px-5 py-4 flex items-center justify-between gap-4">
              <div>
                <div className="text-[14px] font-semibold text-ink">{scheme.name}</div>
                <div className="text-[12px] text-muted mt-0.5">
                  {scheme.unit_count} units · {scheme.address}
                </div>
                <div className="text-[12px] text-muted mt-0.5">
                  {scheme.levy_collection_pct}% collected · {scheme.open_maintenance_count} open jobs · {scheme.total_members} linked members
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}>
                  {scheme.health.charAt(0).toUpperCase() + scheme.health.slice(1)}
                </span>
                <Link href={`/app/${scheme.id}`} className="text-[12px] text-accent font-medium">View →</Link>
              </div>
            </div>
          ))}
          {schemes.length === 0 && (
            <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-[14px] text-muted">
              No schemes found.
            </div>
          )}
        </div>
      )}
    </div>
  )
}
