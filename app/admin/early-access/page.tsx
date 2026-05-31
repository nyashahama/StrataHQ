'use client'

import { useEffect, useState } from 'react'
import {
  listEarlyAccessRequests,
  approveEarlyAccessRequest,
  rejectEarlyAccessRequest,
  type EarlyAccessRequest,
} from '@/lib/early-access-api'

const STATUS_COLORS: Record<string, string> = {
  pending: 'text-amber bg-yellowbg',
  approved: 'text-green bg-green-bg',
  rejected: 'text-muted bg-page',
}

export default function AdminEarlyAccessPage() {
  const [requests, setRequests] = useState<EarlyAccessRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [working, setWorking] = useState<string | null>(null)

  async function load() {
    setLoading(true)
    const data = await listEarlyAccessRequests()
    setRequests(data)
    setLoading(false)
  }

  useEffect(() => { load() }, [])

  async function handleApprove(id: string) {
    setWorking(id)
    await approveEarlyAccessRequest(id)
    await load()
    setWorking(null)
  }

  async function handleReject(id: string) {
    setWorking(id)
    await rejectEarlyAccessRequest(id)
    await load()
    setWorking(null)
  }

  if (loading) {
    return <div className="p-8 text-sm text-muted">Loading…</div>
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-10">
      <h1 className="font-serif text-2xl font-semibold text-ink mb-6">
        Early access requests
      </h1>

      {requests.length === 0 ? (
        <p className="text-sm text-muted">No requests yet.</p>
      ) : (
        <div className="space-y-3">
          {requests.map((req) => (
            <div
              key={req.id}
              className="bg-surface border border-border rounded-lg px-5 py-4 flex items-start justify-between gap-4"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-semibold text-ink">{req.full_name}</span>
                  <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${STATUS_COLORS[req.status] ?? ''}`}>
                    {req.status}
                  </span>
                </div>
                <p className="text-sm text-muted truncate">{req.email}</p>
                <p className="text-xs text-muted-2 mt-1">
                  {req.scheme_name} · {req.unit_count} units ·{' '}
                  {new Date(req.created_at).toLocaleDateString('en-ZA', {
                    day: 'numeric', month: 'short', year: 'numeric',
                  })}
                </p>
              </div>

              {req.status === 'pending' && (
                <div className="flex items-center gap-2 flex-shrink-0">
                  <button
                    onClick={() => handleReject(req.id)}
                    disabled={working === req.id}
                    className="px-3 py-1.5 text-xs font-medium text-muted border border-border rounded hover:bg-page transition-colors disabled:opacity-50"
                  >
                    Reject
                  </button>
                  <button
                    onClick={() => handleApprove(req.id)}
                    disabled={working === req.id}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-accent rounded hover:opacity-90 transition-opacity disabled:opacity-50"
                  >
                    {working === req.id ? 'Approving…' : 'Approve'}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
