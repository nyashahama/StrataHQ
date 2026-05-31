'use client'

import { useParams } from 'next/navigation'
import { useState } from 'react'

import { useAuth } from '@/lib/auth'
import { schemeKeys } from '@/lib/query-keys'
import { useAuthenticatedQuery } from '@/hooks/useAuthenticatedQuery'
import { getSchemeAuditEvents } from '@/lib/audit-api'
import type { AuditEventInfo } from '@/lib/audit'
import RetryState from '@/components/RetryState'

function formatAction(action: string): string {
  return action
    .replace(/\./g, ' ')
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-ZA', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function EventRow({ event }: { event: AuditEventInfo }) {
  return (
    <div className="bg-surface border border-border rounded-lg px-5 py-4">
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <span className="text-[12px] font-semibold bg-accent-bg text-accent px-2 py-[2px] rounded">
          {formatAction(event.action)}
        </span>
        <span className="text-[12px] text-muted">
          {event.resource_type}
          {event.resource_id ? ` · ${event.resource_id.slice(0, 8)}` : ''}
        </span>
        <span className="text-[12px] text-muted ml-auto">
          {formatDate(event.occurred_at)}
        </span>
      </div>
      <div className="text-[12px] text-muted mb-2">
        {event.actor_role}
        {event.actor_user_id ? ` · ${event.actor_user_id.slice(0, 8)}` : ''}
      </div>
      <details className="text-[12px]">
        <summary className="cursor-pointer text-muted hover:text-ink transition-colors">
          Details
        </summary>
        <div className="mt-2 space-y-2">
          {event.before_state && (
            <div>
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">Before</span>
              <pre className="mt-1 bg-page border border-border rounded px-3 py-2 overflow-x-auto text-[11px] text-ink">
                {JSON.stringify(event.before_state, null, 2)}
              </pre>
            </div>
          )}
          {event.after_state && (
            <div>
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">After</span>
              <pre className="mt-1 bg-page border border-border rounded px-3 py-2 overflow-x-auto text-[11px] text-ink">
                {JSON.stringify(event.after_state, null, 2)}
              </pre>
            </div>
          )}
          {event.metadata && (
            <div>
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">Metadata</span>
              <pre className="mt-1 bg-page border border-border rounded px-3 py-2 overflow-x-auto text-[11px] text-ink">
                {JSON.stringify(event.metadata, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </details>
    </div>
  )
}

export default function AuditPage() {
  const [limit, setLimit] = useState(50)
  const { user } = useAuth()
  const params = useParams()
  const schemeId = params.schemeId as string

  const {
    data,
    isLoading,
    error,
    refetch,
  } = useAuthenticatedQuery<{ events: AuditEventInfo[]; total: number; limit: number }>({
    queryKey: schemeKeys.audit(schemeId, limit),
    queryFn: () => getSchemeAuditEvents(schemeId, limit),
    staleTime: 30_000,
  })

  if (user?.role === 'resident') {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Audit log</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Audit log</h1>
        <p className="text-[14px] text-muted">Residents do not have access to the audit log.</p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <p className="text-[12px] text-muted mb-4">Scheme › Audit log</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Audit log</h1>
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading audit events…
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <RetryState
        title="Could not load audit events"
        message="Temporary service issue. Try again."
        onRetry={refetch}
      />
    )
  }

  const events = data?.events ?? []
  const total = data?.total ?? 0
  const responseLimit = data?.limit ?? limit
  const truncated = total > responseLimit
  const canLoadMore = total > responseLimit && responseLimit < 200
  const hasMore = total > events.length

  const loadMore = () => {
    setLimit(total > 200 ? 200 : total)
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Audit log</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Audit log</h1>
      <p className="text-[14px] text-muted mb-8">
        Scheme activity trail showing recent changes.
      </p>
      {truncated && (
        <p className="text-[12px] text-muted mb-6">
          Showing latest {events.length} of {total} audit events.
          {hasMore ? " More older events are available." : ""}
        </p>
      )}
      {canLoadMore && (
        <div className="mb-8">
          <button
            type="button"
            onClick={loadMore}
            className="text-[13px] font-medium text-accent hover:underline">
            Show more events
          </button>
        </div>
      )}

      {events.length === 0 ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-[14px] text-muted">
          No audit events yet.
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {events.map((event) => (
            <EventRow key={event.id} event={event} />
          ))}
        </div>
      )}
    </div>
  )
}
