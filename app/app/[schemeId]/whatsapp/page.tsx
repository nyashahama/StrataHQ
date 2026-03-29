'use client'
import { useState } from 'react'
import { useAuth } from '@/lib/auth'
import { useToast } from '@/lib/toast'
import {
  mockWhatsAppThreads,
  mockWhatsAppBroadcasts,
  getWhatsAppStats,
  SCHEME_WHATSAPP_NUMBER,
  type WhatsAppThread,
  type WhatsAppBroadcast,
} from '@/lib/mock/whatsapp'

const BROADCAST_TYPE_STYLES: Record<string, string> = {
  levy:        'bg-accent-bg text-accent',
  agm:         'bg-green-bg text-green',
  maintenance: 'bg-yellowbg text-amber',
  general:     'bg-border text-muted',
}

function timeAgo(isoDate: string): string {
  const diff = Date.now() - new Date(isoDate).getTime()
  const days = Math.floor(diff / 86400000)
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  return `${days} days ago`
}

// ── Resident view ──────────────────────────────────────────────────────────

function ResidentView({ unitIdentifier }: { unitIdentifier: string }) {
  const thread = mockWhatsAppThreads.find(t => t.unit_identifier === unitIdentifier)

  if (!thread || !thread.connected) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[600px]">
        <p className="text-[12px] text-muted mb-4">Scheme › WhatsApp</p>
        <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">WhatsApp</h1>
        <p className="text-[14px] text-muted mb-8">Connect your WhatsApp to manage your levy and maintenance on the go.</p>

        <div className="bg-surface border border-border rounded-xl px-6 py-8 text-center mb-6">
          {/* WhatsApp logo placeholder */}
          <div className="w-16 h-16 rounded-full bg-[#25D366] flex items-center justify-center mx-auto mb-4">
            <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
              <path
                d="M16 3C8.82 3 3 8.82 3 16c0 2.36.64 4.57 1.75 6.47L3 29l6.72-1.73A12.9 12.9 0 0016 29c7.18 0 13-5.82 13-13S23.18 3 16 3z"
                fill="white"
              />
              <path
                d="M22.4 19.5c-.3-.15-1.8-.9-2.08-.99-.28-.1-.48-.15-.68.15-.2.3-.77.99-.95 1.2-.17.2-.35.22-.65.07a8.24 8.24 0 01-2.43-1.5 9.1 9.1 0 01-1.68-2.1c-.18-.3-.02-.47.13-.62.14-.14.3-.35.45-.52.15-.17.2-.3.3-.5.1-.2.05-.37-.02-.52-.08-.15-.68-1.63-.93-2.23-.24-.59-.49-.5-.68-.51l-.58-.01c-.2 0-.52.07-.8.37-.27.3-1.02.99-1.02 2.43s1.05 2.82 1.2 3.02c.15.2 2.06 3.14 4.99 4.4.7.3 1.24.48 1.67.62.7.22 1.34.19 1.84.11.56-.08 1.73-.71 1.97-1.39.25-.68.25-1.26.17-1.38-.07-.12-.27-.19-.57-.34z"
                fill="#25D366"
              />
            </svg>
          </div>
          <h2 className="font-serif text-[20px] font-semibold text-ink mb-2">Connect via WhatsApp</h2>
          <p className="text-[13px] text-muted mb-6">
            Save this number on your phone and send "Hi" to get started.
          </p>
          <div className="inline-block bg-page border border-border rounded-lg px-6 py-3 font-mono text-[18px] font-semibold text-ink mb-6">
            {SCHEME_WHATSAPP_NUMBER}
          </div>
          <div className="text-[12px] text-muted space-y-1">
            <p>✅ Check your levy balance instantly</p>
            <p>✅ Submit maintenance requests with photos</p>
            <p>✅ Receive scheme notices and AGM reminders</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[600px]">
      <p className="text-[12px] text-muted mb-4">Scheme › WhatsApp</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">WhatsApp</h1>
      <p className="text-[14px] text-muted mb-6">Your conversation with the Sunridge Heights bot.</p>

      {/* Connected badge */}
      <div className="flex items-center gap-3 mb-6 bg-green-bg border border-green rounded-lg px-4 py-3">
        <div className="w-2 h-2 rounded-full bg-green flex-shrink-0" />
        <div>
          <p className="text-[13px] font-semibold text-green">Connected to WhatsApp</p>
          <p className="text-[12px] text-green/70">Bot number: {SCHEME_WHATSAPP_NUMBER}</p>
        </div>
      </div>

      {/* Quick actions */}
      <div className="grid grid-cols-3 gap-3 mb-6">
        {[
          { label: 'Check balance', emoji: '💳' },
          { label: 'Log request', emoji: '🔧' },
          { label: 'Scheme notices', emoji: '📋' },
        ].map(({ label, emoji }) => (
          <div
            key={label}
            className="bg-surface border border-border rounded-lg px-3 py-3 text-center text-[12px] font-medium text-muted"
          >
            <div className="text-[20px] mb-1">{emoji}</div>
            {label}
          </div>
        ))}
      </div>

      {/* Chat thread */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-3 border-b border-border flex items-center gap-2">
          <div className="w-7 h-7 rounded-full bg-[#25D366] flex items-center justify-center flex-shrink-0">
            <span className="text-white text-[11px] font-bold">SH</span>
          </div>
          <div>
            <p className="text-[12px] font-semibold text-ink">Sunridge Heights Bot</p>
            <p className="text-[10px] text-muted">{SCHEME_WHATSAPP_NUMBER}</p>
          </div>
        </div>
        <div className="px-4 py-4 space-y-3 bg-[#ECE5DD] dark:bg-page min-h-[200px]">
          {thread.messages.map((msg, i) => (
            <div key={i} className={`flex ${msg.from === 'resident' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={[
                  'max-w-[80%] rounded-lg px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap shadow-sm',
                  msg.from === 'resident'
                    ? 'bg-[#DCF8C6] text-ink rounded-tr-sm'
                    : 'bg-surface text-ink rounded-tl-sm',
                ].join(' ')}
              >
                {msg.text}
                <span className="block text-[10px] text-muted/60 text-right mt-0.5">{msg.time}</span>
              </div>
            </div>
          ))}
        </div>
        <div className="px-4 py-3 border-t border-border bg-surface">
          <p className="text-[12px] text-muted text-center">
            Reply directly on WhatsApp at {SCHEME_WHATSAPP_NUMBER}
          </p>
        </div>
      </div>
    </div>
  )
}

// ── Thread card (agent inbox) ──────────────────────────────────────────────

function ThreadCard({ thread }: { thread: WhatsAppThread }) {
  const [expanded, setExpanded] = useState(false)
  const lastMsg = thread.messages[thread.messages.length - 1]

  return (
    <div className={`border-b border-border last:border-b-0 ${!thread.connected ? 'opacity-50' : ''}`}>
      <button
        className="w-full flex items-start gap-3 px-5 py-3 hover:bg-hover-subtle transition-colors text-left"
        onClick={() => thread.messages.length > 0 && setExpanded(e => !e)}
      >
        {/* Avatar */}
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
            <svg
              width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5"
              className={`text-muted transition-transform ${expanded ? 'rotate-180' : ''}`}
            >
              <polyline points="2,4 6,8 10,4" />
            </svg>
          )}
        </div>
      </button>

      {expanded && thread.messages.length > 0 && (
        <div className="px-5 pb-4">
          <div className="bg-[#ECE5DD] dark:bg-page rounded-lg px-4 py-3 space-y-2 ml-11">
            {thread.messages.map((msg, i) => (
              <div key={i} className={`flex ${msg.from === 'resident' ? 'justify-end' : 'justify-start'}`}>
                <div
                  className={[
                    'max-w-[85%] rounded-lg px-3 py-2 text-[12px] leading-relaxed whitespace-pre-wrap',
                    msg.from === 'resident'
                      ? 'bg-[#DCF8C6] text-ink rounded-tr-sm'
                      : 'bg-white dark:bg-surface text-ink rounded-tl-sm',
                  ].join(' ')}
                >
                  {msg.text}
                  <span className="block text-[10px] text-muted/60 text-right mt-0.5">{msg.time}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ── Broadcast card ─────────────────────────────────────────────────────────

function BroadcastCard({ broadcast }: { broadcast: WhatsAppBroadcast }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border-b border-border last:border-b-0">
      <button
        className="w-full flex items-start gap-3 px-5 py-3 hover:bg-hover-subtle transition-colors text-left"
        onClick={() => setExpanded(e => !e)}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5">
            <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full uppercase tracking-wide ${BROADCAST_TYPE_STYLES[broadcast.type]}`}>
              {broadcast.type}
            </span>
            <span className="text-[11px] text-muted">{timeAgo(broadcast.sent_at)}</span>
          </div>
          <p className="text-[13px] text-ink truncate">{broadcast.message.split('\n')[1] || broadcast.message.split('\n')[0]}</p>
          <p className="text-[11px] text-muted mt-0.5">Sent to {broadcast.recipient_count} residents</p>
        </div>
        <svg
          width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5"
          className={`text-muted flex-shrink-0 mt-1 transition-transform ${expanded ? 'rotate-180' : ''}`}
        >
          <polyline points="2,4 6,8 10,4" />
        </svg>
      </button>
      {expanded && (
        <div className="px-5 pb-4 ml-0">
          <div className="bg-page border border-border rounded-lg px-4 py-3">
            <pre className="text-[12px] text-ink whitespace-pre-wrap font-sans leading-relaxed">{broadcast.message}</pre>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Agent/Trustee view ─────────────────────────────────────────────────────

function AgentView() {
  const { addToast } = useToast()
  const [activeTab, setActiveTab] = useState<'inbox' | 'broadcasts'>('inbox')
  const [broadcastOpen, setBroadcastOpen] = useState(false)
  const [broadcastText, setBroadcastText] = useState('')

  const stats = getWhatsAppStats(mockWhatsAppThreads)
  const unreadThreads = mockWhatsAppThreads.filter(t => t.unread > 0)

  function handleSendBroadcast() {
    if (!broadcastText.trim()) return
    setBroadcastOpen(false)
    setBroadcastText('')
    addToast(`WhatsApp broadcast sent to ${stats.connected} connected residents.`, 'success')
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
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M1 7l5-5 5 5M6 13V2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          Broadcast
        </button>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        {[
          { label: 'Residents connected', value: `${stats.connected}/${stats.total_residents}` },
          { label: 'Unread messages', value: String(stats.unread) },
          { label: 'Bot number', value: SCHEME_WHATSAPP_NUMBER },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-5 py-4">
            <div className="text-[20px] font-semibold text-ink font-serif mb-1 truncate">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-border mb-0">
        {(['inbox', 'broadcasts'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={[
              'px-4 py-2 text-[13px] font-medium border-b-2 transition-colors capitalize',
              activeTab === tab
                ? 'border-ink text-ink'
                : 'border-transparent text-muted hover:text-ink',
            ].join(' ')}
          >
            {tab}
            {tab === 'inbox' && stats.unread > 0 && (
              <span className="ml-2 w-4 h-4 rounded-full bg-[#25D366] text-white text-[10px] font-bold inline-flex items-center justify-center">
                {stats.unread}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="bg-surface border border-border border-t-0 rounded-b-lg overflow-hidden">
        {activeTab === 'inbox' && (
          <>
            <div className="px-5 py-2 border-b border-border bg-page">
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">
                {mockWhatsAppThreads.length} conversations
              </span>
            </div>
            {mockWhatsAppThreads.map(thread => (
              <ThreadCard key={thread.id} thread={thread} />
            ))}
          </>
        )}

        {activeTab === 'broadcasts' && (
          <>
            <div className="px-5 py-2 border-b border-border bg-page">
              <span className="text-[11px] font-semibold text-muted uppercase tracking-wide">
                Recent broadcasts
              </span>
            </div>
            {mockWhatsAppBroadcasts.map(broadcast => (
              <BroadcastCard key={broadcast.id} broadcast={broadcast} />
            ))}
          </>
        )}
      </div>

      {/* Broadcast modal */}
      {broadcastOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-ink/50 backdrop-blur-[2px]">
          <div className="bg-surface border border-border rounded-xl shadow-2xl w-full max-w-[520px]">
            <div className="flex items-center justify-between px-6 py-4 border-b border-border">
              <h2 className="font-serif text-[18px] font-semibold text-ink">Send WhatsApp broadcast</h2>
              <button onClick={() => setBroadcastOpen(false)} className="text-muted hover:text-ink">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <line x1="3" y1="3" x2="13" y2="13" /><line x1="13" y1="3" x2="3" y2="13" />
                </svg>
              </button>
            </div>
            <div className="px-6 py-4">
              <p className="text-[13px] text-muted mb-3">
                Message will be sent to <strong>{stats.connected} connected residents</strong>.
              </p>
              <textarea
                value={broadcastText}
                onChange={e => setBroadcastText(e.target.value)}
                placeholder="Type your message…"
                rows={5}
                className="w-full bg-page border border-border rounded-lg px-3 py-2 text-[13px] text-ink placeholder-muted outline-none focus:border-accent transition-colors resize-none"
              />
              <p className="text-[11px] text-muted mt-1">{broadcastText.length} characters</p>
            </div>
            <div className="flex justify-between px-6 py-4 border-t border-border">
              <button onClick={() => setBroadcastOpen(false)} className="text-[13px] text-muted hover:text-ink">
                Cancel
              </button>
              <button
                onClick={handleSendBroadcast}
                disabled={!broadcastText.trim()}
                className="text-[13px] font-semibold bg-[#25D366] text-white px-5 py-2 rounded-lg hover:bg-[#22c55e] transition-colors disabled:opacity-40"
              >
                Send to {stats.connected} residents
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Page entry ─────────────────────────────────────────────────────────────

export default function WhatsAppPage() {
  const { user } = useAuth()

  if (user?.role === 'resident') {
    return <ResidentView unitIdentifier={''} />
  }

  return <AgentView />
}
