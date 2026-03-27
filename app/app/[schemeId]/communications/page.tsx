'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockNotices, type Notice } from '@/lib/mock/communications'

const TYPE_STYLES: Record<Notice['type'], string> = {
  general: 'bg-[#f0efe9] text-muted',
  urgent:  'bg-red-bg text-red',
  agm:     'bg-accent-bg text-accent',
  levy:    'bg-yellowbg text-[#92400e]',
}

const TYPE_LABELS: Record<Notice['type'], string> = {
  general: 'General',
  urgent:  'Urgent',
  agm:     'AGM',
  levy:    'Levy',
}

export default function CommunicationsPage() {
  const { user } = useMockAuth()
  const [expanded, setExpanded] = useState<string | null>(null)
  const canCompose = user?.role === 'agent'

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Communications</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Communications</h1>
      <p className="text-[14px] text-muted mb-8">Notices, announcements, and correspondence.</p>

      <div className="flex items-center justify-between mb-6">
        <span className="text-[13px] text-muted">{mockNotices.length} notices</span>
        {canCompose && (
          <button className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:bg-[#245a96] transition-colors">
            + Compose notice
          </button>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {mockNotices.map(notice => (
          <div key={notice.id} className="bg-white border border-border rounded-lg overflow-hidden">
            <button
              className="w-full px-5 py-4 flex items-start justify-between gap-4 text-left hover:bg-page transition-colors"
              onClick={() => setExpanded(expanded === notice.id ? null : notice.id)}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`text-[10px] font-semibold px-2 py-[2px] rounded-full ${TYPE_STYLES[notice.type]}`}>
                    {TYPE_LABELS[notice.type]}
                  </span>
                  <span className="text-[11px] text-muted">
                    {new Date(notice.sent_at).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })}
                  </span>
                </div>
                <div className="text-[13px] font-semibold text-ink">{notice.title}</div>
                <div className="text-[12px] text-muted mt-0.5">{notice.sent_by_name}</div>
              </div>
              <span className="text-muted text-[12px] flex-shrink-0 mt-1">{expanded === notice.id ? '▲' : '▼'}</span>
            </button>
            {expanded === notice.id && (
              <div className="px-5 pb-4 border-t border-border">
                <p className="text-[13px] text-ink leading-relaxed whitespace-pre-line pt-4">{notice.body}</p>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
