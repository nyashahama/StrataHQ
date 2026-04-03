'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'
import { createWhatsAppBroadcast, getWhatsAppDashboard } from '@/lib/whatsapp-api'
import type {
  WhatsAppBroadcast,
  WhatsAppBroadcastType,
  WhatsAppDashboard,
  WhatsAppMessage,
  WhatsAppThread,
} from '@/lib/whatsapp'

const BROADCAST_TYPE_STYLES: Record<WhatsAppBroadcastType, string> = {
  levy: 'bg-accent-bg text-accent',
  agm: 'bg-green-bg text-green',
  maintenance: 'bg-yellowbg text-amber',
  general: 'bg-border text-muted',
}

function timeAgo(isoDate: string): string {
  const diff = Date.now() - new Date(isoDate).getTime()
  const days = Math.floor(diff / 86400000)
  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  return `${days} days ago`
}

function messageTime(isoDate: string): string {
  return new Date(isoDate).toLocaleTimeString('en-ZA', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function ChatBubble({ message }: { message: WhatsAppMessage }) {
  return (
    <div className={`flex ${message.from === 'resident' ? 'justify-end' : 'justify-start'}`}>
      <div
        className={[
          'max-w-[85%] rounded-lg px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap shadow-sm',
          message.from === 'resident'
            ? 'bg-[#DCF8C6] text-ink rounded-tr-sm'
            : 'bg-surface text-ink rounded-tl-sm',
        ].join(' ')}
      >
        {message.text}
        <span className="block text-[10px] text-muted/60 text-right mt-0.5">{messageTime(message.sent_at)}</span>
      </div>
    </div>
  )
}

function ResidentView({ thread, phoneNumber }: { thread?: WhatsAppThread | null; phoneNumber: string }) {
  if (!thread || !thread.connected) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[600px]">
        <p className="text-[12px] text-muted mb-4">Scheme › WhatsApp</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">WhatsApp</h1>
        <p className="text-[14px] text-muted mb-8">Connect your WhatsApp to manage your levy and maintenance on the go.</p>

        <div className="bg-surface border border-border rounded-xl px-6 py-8 text-center mb-6">
          <div className="w-16 h-16 rounded-full bg-[#25D366] flex items-center justify-center mx-auto mb-4">
            <span className="text-white text-[22px] font-bold">WA</span>
          </div>
          <h2 className="font-serif text-[20px] font-semibold text-ink mb-2">Connect via WhatsApp</h2>
          <p className="text-[13px] text-muted mb-6">
            Save this number on your phone and send &quot;Hi&quot; to get started.
          </p>
          <div className="inline-block bg-page border border-border rounded-lg px-6 py-3 font-mono text-[18px] font-semibold text-ink mb-6">
            {phoneNumber}
          </div>
          <div className="text-[12px] text-muted space-y-1">
            <p>Check your levy balance instantly</p>
            <p>Submit maintenance requests with photos</p>
            <p>Receive scheme notices and AGM reminders</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[600px]">
      <p className="text-[12px] text-muted mb-4">Scheme › WhatsApp</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">WhatsApp</h1>
      <p className="text-[14px] text-muted mb-6">Your conversation with the scheme bot.</p>

      <div className="flex items-center gap-3 mb-6 bg-green-bg border border-green rounded-lg px-4 py-3">
        <div className="w-2 h-2 rounded-full bg-green flex-shrink-0" />
        <div>
          <p className="text-[13px] font-semibold text-green">Connected to WhatsApp</p>
          <p className="text-[12px] text-green/70">Bot number: {phoneNumber}</p>
        </div>
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-3 border-b border-border flex items-center gap-2">
          <div className="w-7 h-7 rounded-full bg-[#25D366] flex items-center justify-center flex-shrink-0">
            <span className="text-white text-[11px] font-bold">SH</span>
          </div>
          <div>
            <p className="text-[12px] font-semibold text-ink">Scheme Bot</p>
            <p className="text-[10px] text-muted">{phoneNumber}</p>
          </div>
        </div>
        <div className="px-4 py-4 space-y-3 bg-[#ECE5DD] min-h-[200px]">
          {thread.messages.length === 0 ? (
            <p className="text-[12px] text-muted text-center">No messages yet.</p>
          ) : thread.messages.map(message => (
            <ChatBubble key={message.id} message={message} />
          ))}
        </div>
        <div className="px-4 py-3 border-t border-border bg-surface">
          <p className="text-[12px] text-muted text-center">
            Reply directly on WhatsApp at {phoneNumber}
          </p>
        </div>
      </div>
    </div>
  )
}

function ThreadCard({ thread }: { thread: WhatsAppThread }) {
  const [expanded, setExpanded] = useState(false)
  const lastMsg = thread.messages[thread.messages.length - 1]

  return (
    <div className={`border-b border-border last:border-b-0 ${!thread.connected ? 'opacity-50' : ''}`}>
      <button
        className="w-full flex items-start gap-3 px-5 py-3 hover:bg-hover-subtle transition-colors text-left"
        onClick={() => thread.messages.length > 0 && setExpanded(current => !current)}
      >
        <div className="w-8 h-8 rounded-full bg-accent-bg flex items-center justify-center flex-shrink-0 mt-0.5">
          <span className="text-[11px] font-bold text-accent">{thread.unit_identifier}</span>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[13px] font-semibold text-ink">{thread.owner_name}</span>
            <span className="text-[11px] text-muted flex-shrink-0">{timeAgo(thread.last_active)}</span>
          </div>
          <p className="text-[12px] text-muted truncate mt-0.5">
            {!thread.connected
              ? 'Not connected to WhatsApp'
              : lastMsg
                ? lastMsg.text.split('\n')[0]
                : 'No messages yet'}
          </p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {thread.unread > 0 && (
            <span className="w-5 h-5 rounded-full bg-[#25D366] text-white text-[10px] font-bold flex items-center justify-center">
              {thread.unread}
            </span>
          )}
          {thread.messages.length > 0 && (
            <span className={`text-muted text-[12px] transition-transform ${expanded ? 'rotate-180' : ''}`}>
              ▼
            </span>
          )}
        </div>
      </button>

      {expanded && thread.messages.length > 0 && (
        <div className="px-5 pb-4">
          <div className="bg-[#ECE5DD] rounded-lg px-4 py-3 space-y-2 ml-11">
            {thread.messages.map(message => (
              <ChatBubble key={message.id} message={message} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function BroadcastCard({ broadcast }: { broadcast: WhatsAppBroadcast }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border-b border-border last:border-b-0">
      <button
        className="w-full flex items-start gap-3 px-5 py-3 hover:bg-hover-subtle transition-colors text-left"
        onClick={() => setExpanded(current => !current)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5">
            <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full uppercase tracking-wide ${BROADCAST_TYPE_STYLES[broadcast.type]}`}>
              {broadcast.type}
            </span>
            <span className="text-[11px] text-muted">{timeAgo(broadcast.sent_at)}</span>
          </div>
          <p className="text-[13px] text-ink truncate">{broadcast.message.split('\n')[0]}</p>
          <p className="text-[11px] text-muted mt-0.5">
            Sent to {broadcast.recipient_count} residents
            {broadcast.sent_by_name ? ` by ${broadcast.sent_by_name}` : ''}
          </p>
        </div>
        <span className={`text-muted text-[12px] flex-shrink-0 mt-1 transition-transform ${expanded ? 'rotate-180' : ''}`}>
          ▼
        </span>
      </button>
      {expanded && (
        <div className="px-5 pb-4">
          <div className="bg-page border border-border rounded-lg px-4 py-3">
            <pre className="text-[12px] text-ink whitespace-pre-wrap font-sans leading-relaxed">{broadcast.message}</pre>
          </div>
        </div>
      )}
    </div>
  )
}

function OperatorView({
  dashboard,
  schemeId,
  onReload,
}: {
  dashboard: WhatsAppDashboard
  schemeId: string
  onReload: () => Promise<void>
}) {
  const { addToast } = useToast()
  const [activeTab, setActiveTab] = useState<'inbox' | 'broadcasts'>('inbox')
  const [broadcastOpen, setBroadcastOpen] = useState(false)
  const [broadcastText, setBroadcastText] = useState('')
  const [broadcastType, setBroadcastType] = useState<WhatsAppBroadcastType>('general')
  const [sending, setSending] = useState(false)

  async function submitBroadcast(schemeId: string) {
    if (!broadcastText.trim()) return

    try {
      setSending(true)
      const created = await createWhatsAppBroadcast(schemeId, {
        message: broadcastText.trim(),
        type: broadcastType,
      })
      setBroadcastOpen(false)
      setBroadcastText('')
      setBroadcastType('general')
      await onReload()
      addToast(`WhatsApp broadcast sent to ${created.recipient_count} connected residents.`, 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to send WhatsApp broadcast',
        'error',
      )
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <div className="flex items-start justify-between mb-8">
        <div>
          <p className="text-[12px] text-muted mb-1">Scheme › WhatsApp</p>
          <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">WhatsApp</h1>
          <p className="text-[14px] text-muted">Resident conversations and broadcasts via WhatsApp.</p>
        </div>
        <button
          onClick={() => setBroadcastOpen(true)}
          className="flex-shrink-0 flex items-center gap-2 text-[13px] font-semibold bg-[#25D366] text-white px-4 py-2 rounded-lg hover:bg-[#22c55e] transition-colors mt-1"
        >
          Broadcast
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        {[
          { label: 'Residents connected', value: `${dashboard.connected_count}/${dashboard.total_residents}` },
          { label: 'Unread messages', value: String(dashboard.unread_count) },
          { label: 'Bot number', value: dashboard.phone_number },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[20px] font-semibold text-ink font-serif mb-1 truncate">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="flex border-b border-border mb-0">
        {(['inbox', 'broadcasts'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={[
              'px-4 py-2 text-[13px] font-medium border-b-2 transition-colors capitalize',
              activeTab === tab ? 'border-ink text-ink' : 'border-transparent text-muted hover:text-ink',
            ].join(' ')}
          >
            {tab}
            {tab === 'inbox' && dashboard.unread_count > 0 && (
              <span className="ml-2 w-4 h-4 rounded-full bg-[#25D366] text-white text-[10px] font-bold inline-flex items-center justify-center">
                {dashboard.unread_count}
              </span>
            )}
          </button>
        ))}
      </div>

      <div className="bg-surface border border-border border-t-0 rounded-b-lg overflow-hidden">
        {activeTab === 'inbox' ? (
          <>
            <div className="px-5 py-2 border-b border-border bg-page">
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">
                {dashboard.threads.length} conversations
              </span>
            </div>
            {dashboard.threads.length === 0 ? (
              <div className="px-5 py-12 text-[13px] text-muted text-center">No WhatsApp conversations yet.</div>
            ) : dashboard.threads.map(thread => (
              <ThreadCard key={thread.id} thread={thread} />
            ))}
          </>
        ) : (
          <>
            <div className="px-5 py-2 border-b border-border bg-page">
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">
                Recent broadcasts
              </span>
            </div>
            {dashboard.broadcasts.length === 0 ? (
              <div className="px-5 py-12 text-[13px] text-muted text-center">No WhatsApp broadcasts yet.</div>
            ) : dashboard.broadcasts.map(broadcast => (
              <BroadcastCard key={broadcast.id} broadcast={broadcast} />
            ))}
          </>
        )}
      </div>

      <Modal open={broadcastOpen} onClose={() => !sending && setBroadcastOpen(false)} title="Send WhatsApp broadcast">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Type</label>
            <select
              value={broadcastType}
              onChange={event => setBroadcastType(event.target.value as WhatsAppBroadcastType)}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            >
              <option value="general">General</option>
              <option value="agm">AGM</option>
              <option value="levy">Levy</option>
              <option value="maintenance">Maintenance</option>
            </select>
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Message</label>
            <textarea
              value={broadcastText}
              onChange={event => setBroadcastText(event.target.value)}
              placeholder="Type your message..."
              rows={5}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent resize-none"
            />
            <p className="text-[11px] text-muted mt-1">
              Message will be sent to {dashboard.connected_count} connected residents.
            </p>
          </div>
          <div className="flex gap-2 pt-1">
            <button
              onClick={() => submitBroadcast(schemeId)}
              disabled={!broadcastText.trim() || sending}
              className="flex-1 bg-[#25D366] text-white text-[13px] font-semibold py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {sending ? 'Sending...' : `Send to ${dashboard.connected_count} residents`}
            </button>
            <button
              onClick={() => setBroadcastOpen(false)}
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

export default function WhatsAppPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<WhatsAppDashboard | null>(null)
  const [loading, setLoading] = useState(true)

  async function loadDashboard() {
    try {
      setLoading(true)
      setDashboard(await getWhatsAppDashboard(schemeId))
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to load WhatsApp dashboard',
        'error',
      )
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadDashboard()
  }, [addToast, schemeId])

  if (loading || !dashboard) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading WhatsApp…
        </div>
      </div>
    )
  }

  if (user?.role === 'resident') {
    return <ResidentView thread={dashboard.resident_thread} phoneNumber={dashboard.phone_number} />
  }

  return <OperatorView dashboard={dashboard} schemeId={schemeId} onReload={loadDashboard} />
}
