'use client'
import { useState, useRef, useEffect } from 'react'
import { useAuth } from '@/lib/auth'

interface Message {
  role: 'user' | 'assistant'
  content: string
}

const SUGGESTED_QUESTIONS = [
  'Which schemes have the lowest levy collection rates?',
  'Are there any maintenance jobs breaching SLA right now?',
  'What is the total outstanding levy debt across all schemes?',
  'Draft an overdue levy reminder for Unit 3A',
]

export default function Copilot() {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [streamingContent, setStreamingContent] = useState('')
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Only available for agents and trustees
  if (!user || user.role === 'resident') return null

  // eslint-disable-next-line react-hooks/rules-of-hooks
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [open])

  // eslint-disable-next-line react-hooks/rules-of-hooks
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  async function sendMessage(text: string) {
    if (!text.trim() || streaming) return

    const userMessage: Message = { role: 'user', content: text.trim() }
    const history = messages
    setMessages(prev => [...prev, userMessage])
    setInput('')
    setStreaming(true)
    setStreamingContent('')
    setError(null)

    try {
      const response = await fetch('/api/copilot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text.trim(), history }),
      })

      if (!response.ok || !response.body) {
        const errText = await response.text()
        setError(errText || 'Something went wrong. Please try again.')
        setStreaming(false)
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let accumulated = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        accumulated += decoder.decode(value, { stream: true })
        setStreamingContent(accumulated)
      }

      setMessages(prev => [...prev, { role: 'assistant', content: accumulated }])
      setStreamingContent('')
    } catch {
      setError('Network error — check your connection and try again.')
    } finally {
      setStreaming(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage(input)
    }
  }

  const showWelcome = messages.length === 0 && !streaming

  return (
    <>
      {/* Floating button */}
      <button
        onClick={() => setOpen(o => !o)}
        className={[
          'fixed bottom-6 right-6 z-50 w-12 h-12 rounded-full shadow-lg flex items-center justify-center transition-all duration-200',
          open ? 'bg-ink text-surface scale-90' : 'bg-ink text-surface hover:scale-105',
        ].join(' ')}
        aria-label="Open AI Copilot"
      >
        {open ? (
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
            <line x1="3" y1="3" x2="13" y2="13" />
            <line x1="13" y1="3" x2="3" y2="13" />
          </svg>
        ) : (
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path
              d="M9 1.5C4.86 1.5 1.5 4.36 1.5 8c0 1.74.72 3.33 1.9 4.52L2.5 16.5l4.4-1.5C7.8 15.33 8.38 15.5 9 15.5c4.14 0 7.5-3.36 7.5-7s-3.36-7-7.5-7z"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </button>

      {/* Chat panel */}
      {open && (
        <div
          className="fixed z-50 w-[380px] max-h-[520px] bg-surface border border-border rounded-xl shadow-2xl flex flex-col overflow-hidden"
          style={{ bottom: '80px', right: '24px' }}
        >
          {/* Header */}
          <div className="flex items-center gap-3 px-4 py-3 border-b border-border flex-shrink-0">
            <div className="w-7 h-7 rounded-full bg-ink flex items-center justify-center flex-shrink-0">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="white" strokeWidth="1.5">
                <path
                  d="M6 1C3.24 1 1 2.9 1 5.2c0 1.11.5 2.12 1.3 2.85L2 11l3-1c.3.1.63.16.97.16 2.76 0 5-1.9 5-4.24C11 3.52 8.76 1 6 1z"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-[13px] font-semibold text-ink">StrataHQ Copilot</p>
              <p className="text-[11px] text-muted">Ask anything about your schemes</p>
            </div>
            {messages.length > 0 && (
              <button
                onClick={() => { setMessages([]); setError(null) }}
                className="text-[11px] text-muted hover:text-ink transition-colors px-2 py-1 rounded hover:bg-page"
              >
                Clear
              </button>
            )}
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3 min-h-0">
            {showWelcome && (
              <div>
                <p className="text-[12px] text-muted mb-2">Try asking:</p>
                <div className="space-y-1.5">
                  {SUGGESTED_QUESTIONS.map(q => (
                    <button
                      key={q}
                      onClick={() => sendMessage(q)}
                      className="w-full text-left text-[12px] text-ink bg-page border border-border rounded-lg px-3 py-2 hover:border-accent hover:bg-accent-dim transition-colors leading-snug"
                    >
                      {q}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {messages.map((m, i) => (
              <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div
                  className={[
                    'max-w-[88%] rounded-xl px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap',
                    m.role === 'user'
                      ? 'bg-ink text-surface rounded-br-sm'
                      : 'bg-page border border-border text-ink rounded-bl-sm',
                  ].join(' ')}
                >
                  {m.content}
                </div>
              </div>
            ))}

            {streaming && streamingContent && (
              <div className="flex justify-start">
                <div className="max-w-[88%] rounded-xl rounded-bl-sm px-3 py-2 text-[13px] leading-relaxed whitespace-pre-wrap bg-page border border-border text-ink">
                  {streamingContent}
                  <span className="inline-block w-[3px] h-[14px] bg-accent ml-0.5 animate-pulse align-[-2px]" />
                </div>
              </div>
            )}

            {streaming && !streamingContent && (
              <div className="flex justify-start">
                <div className="bg-page border border-border rounded-xl rounded-bl-sm px-4 py-3">
                  <div className="flex gap-1 items-center">
                    {[0, 150, 300].map(delay => (
                      <span
                        key={delay}
                        className="w-1.5 h-1.5 bg-muted rounded-full animate-bounce"
                        style={{ animationDelay: `${delay}ms` }}
                      />
                    ))}
                  </div>
                </div>
              </div>
            )}

            {error && (
              <div className="text-[12px] text-red bg-red-bg border border-red rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="flex-shrink-0 border-t border-border px-3 py-2">
            <div className="flex items-end gap-2">
              <textarea
                ref={inputRef}
                value={input}
                onChange={e => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Ask anything…"
                rows={1}
                disabled={streaming}
                className="flex-1 resize-none bg-page border border-border rounded-lg px-3 py-2 text-[13px] text-ink placeholder-muted outline-none focus:border-accent transition-colors disabled:opacity-50"
                style={{ minHeight: '36px', maxHeight: '120px' }}
                onInput={e => {
                  const el = e.currentTarget
                  el.style.height = 'auto'
                  el.style.height = `${Math.min(el.scrollHeight, 120)}px`
                }}
              />
              <button
                onClick={() => sendMessage(input)}
                disabled={!input.trim() || streaming}
                className="flex-shrink-0 w-8 h-8 bg-ink text-surface rounded-lg flex items-center justify-center hover:bg-ink/80 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M1 7h12M8 2l5 5-5 5" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </button>
            </div>
            <p className="text-[10px] text-muted mt-1.5 text-center">Enter to send · Shift+Enter for new line</p>
          </div>
        </div>
      )}
    </>
  )
}
