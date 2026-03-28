'use client'
import { useState } from 'react'
import { useMockAuth } from '@/lib/mock-auth'
import { mockNotices, type Notice } from '@/lib/mock/communications'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

const TYPE_STYLES: Record<Notice['type'], string> = {
  general: 'bg-[#f0efe9] text-muted',
  urgent:  'bg-red-bg text-red',
  agm:     'bg-accent-bg text-accent',
  levy:    'bg-yellowbg text-amber',
}

const TYPE_LABELS: Record<Notice['type'], string> = {
  general: 'General',
  urgent:  'Urgent',
  agm:     'AGM',
  levy:    'Levy',
}

export default function CommunicationsPage() {
  const { user } = useMockAuth()
  const { addToast } = useToast()

  const [notices, setNotices] = useState<Notice[]>([...mockNotices])
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [expanded, setExpanded] = useState<string | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState({ title: '', body: '', type: 'general' as Notice['type'] })

  const canCompose = user?.role === 'agent'

  const filteredNotices = typeFilter === 'all' ? notices : notices.filter(n => n.type === typeFilter)

  function handleCompose() {
    if (!form.title.trim() || !form.body.trim()) return
    const newNotice: Notice = {
      id: `notice-${Date.now()}`,
      scheme_id: 'scheme-001',
      title: form.title.trim(),
      body: form.body.trim(),
      sent_at: new Date().toISOString(),
      sent_by_name: user?.orgName ?? 'Managing Agent',
      type: form.type,
    }
    setNotices(prev => [newNotice, ...prev])
    setExpanded(newNotice.id)
    setShowModal(false)
    setForm({ title: '', body: '', type: 'general' })
    addToast('Notice sent to all residents', 'success')
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Communications</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Communications</h1>
      <p className="text-[14px] text-muted mb-8">Notices, announcements, and correspondence.</p>

      <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
        <span className="text-[13px] text-muted">{notices.length} notices</span>
        {canCompose && (
          <button
            onClick={() => setShowModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
          >
            + Compose notice
          </button>
        )}
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="text-[12px] font-semibold text-muted">Filter:</label>
        <select
          value={typeFilter}
          onChange={e => setTypeFilter(e.target.value)}
          className="border border-border rounded px-3 py-1.5 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
        >
          <option value="all">All notices</option>
          <option value="general">General</option>
          <option value="urgent">Urgent</option>
          <option value="agm">AGM</option>
          <option value="levy">Levy</option>
        </select>
      </div>

      <div className="flex flex-col gap-3">
        {filteredNotices.length === 0 ? (
          <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">No notices match the selected filter.</div>
        ) : filteredNotices.map(notice => (
          <div key={notice.id} className="bg-surface border border-border rounded-lg overflow-hidden">
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


      <Modal open={showModal} onClose={() => setShowModal(false)} title="Compose notice">
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Type</label>
              <select
                value={form.type}
                onChange={e => setForm(f => ({ ...f, type: e.target.value as Notice['type'] }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                <option value="general">General</option>
                <option value="urgent">Urgent</option>
                <option value="agm">AGM</option>
                <option value="levy">Levy</option>
              </select>
            </div>
            <div />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Subject *</label>
            <input
              type="text"
              value={form.title}
              onChange={e => setForm(f => ({ ...f, title: e.target.value }))}
              placeholder="Notice subject"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Body *</label>
            <textarea
              value={form.body}
              onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
              placeholder="Write your notice here…"
              rows={5}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent resize-none"
            />
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={handleCompose}
              disabled={!form.title.trim() || !form.body.trim()}
              className="flex-1 bg-accent text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Send to all residents
            </button>
            <button
              onClick={() => setShowModal(false)}
              className="px-4 text-[13px] font-medium text-muted hover:text-ink border border-border rounded py-2 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
