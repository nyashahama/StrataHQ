'use client'
import { useState } from 'react'
import { useAuth } from '@/lib/auth'
import { mockAgmMeeting, mockAgmResolutions, mockUpcomingAgm, mockUpcomingResolutions, type AgmResolution } from '@/lib/mock/agm'
import { useToast } from '@/lib/toast'
import Modal from '@/components/Modal'

export default function AgmVotingPage() {
  const { user } = useAuth()
  const { addToast } = useToast()

  const [resolutions, setResolutions] = useState<AgmResolution[]>([...mockAgmResolutions])
  const [voted, setVoted] = useState<Set<string>>(new Set())
  const [upcomingResolutions, setUpcomingResolutions] = useState<AgmResolution[]>([...mockUpcomingResolutions])
  const [upcomingVoted, setUpcomingVoted] = useState<Set<string>>(new Set())
  const [showScheduleModal, setShowScheduleModal] = useState(false)
  const [scheduleForm, setScheduleForm] = useState({ date: '', venue: '' })

  const meeting = mockAgmMeeting
  const upcoming = mockUpcomingAgm
  const quorumPct = Math.round((meeting.quorum_present / (meeting.quorum_required * 2)) * 100)

  function castVote(resId: string, inFavour: boolean) {
    if (voted.has(resId)) return
    setResolutions(prev => prev.map(r => {
      if (r.id !== resId) return r
      const newFor = inFavour ? r.votes_for + 1 : r.votes_for
      const newAgainst = inFavour ? r.votes_against : r.votes_against + 1
      return { ...r, votes_for: newFor, votes_against: newAgainst }
    }))
    setVoted(prev => new Set([...prev, resId]))
    addToast(inFavour ? 'Vote in favour recorded' : 'Vote against recorded', 'success')
  }

  function castUpcomingVote(resId: string, inFavour: boolean) {
    if (upcomingVoted.has(resId)) return
    setUpcomingResolutions(prev => prev.map(r => {
      if (r.id !== resId) return r
      return {
        ...r,
        votes_for: inFavour ? r.votes_for + 1 : r.votes_for,
        votes_against: inFavour ? r.votes_against : r.votes_against + 1,
      }
    }))
    setUpcomingVoted(prev => new Set([...prev, resId]))
    addToast(inFavour ? 'Vote in favour recorded' : 'Vote against recorded', 'success')
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › AGM & Voting</p>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-1">
        <h1 className="font-serif text-[28px] font-semibold text-ink">AGM & Voting</h1>
        {user?.role === 'admin' && (
          <button
            onClick={() => setShowScheduleModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-3 py-2 rounded hover:opacity-90 transition-colors"
          >
            + Schedule AGM
          </button>
        )}
      </div>
      <p className="text-[14px] text-muted mb-8">Annual general meetings and trustee resolutions.</p>

      <div className="grid grid-cols-3 gap-3 sm:gap-4 mb-8">
        {[
          { label: 'Last AGM',  value: new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Next AGM',  value: new Date(upcoming.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Quorum',    value: `${meeting.quorum_present}/${meeting.quorum_required * 2}` },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-3 sm:px-5 py-4">
            <div className="text-[16px] sm:text-[22px] font-semibold text-ink font-serif mb-1 leading-tight">{value}</div>
            <div className="text-[11px] sm:text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">
            AGM — {new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
          </span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">Closed</span>
        </div>
        <div className="px-5 py-4 border-b border-border">
          <div className="flex justify-between text-[11px] text-muted mb-2">
            <span className="font-semibold text-ink">Quorum reached ✓</span>
            <span>{meeting.quorum_present} of {meeting.quorum_required * 2} owners present</span>
          </div>
          <div className="h-[6px] bg-border rounded-full overflow-hidden">
            <div className="h-full bg-green rounded-full" style={{ width: `${Math.min(quorumPct, 100)}%` }} />
          </div>
        </div>

        <div className="px-5 py-4 flex flex-col gap-4">
          {resolutions.map((res, i) => {
            const forPct = Math.round((res.votes_for / res.total_eligible) * 100)
            const hasVoted = voted.has(res.id)
            const canVote = user?.role === 'resident' && res.status === 'open'
            return (
              <div key={res.id} className={`${i < resolutions.length - 1 ? 'pb-4 border-b border-border' : ''}`}>
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div>
                    <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {i + 1} OF {resolutions.length}</div>
                    <div className="text-[13px] font-semibold text-ink">{res.title}</div>
                    <div className="text-[12px] text-muted mt-1">{res.description}</div>
                  </div>
                  <span className={`flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full ${res.status === 'passed' ? 'bg-green-bg text-green' : res.status === 'failed' ? 'bg-red-bg text-red' : 'bg-yellowbg text-amber'}`}>
                    {res.status === 'open' ? 'Voting open' : res.status.charAt(0).toUpperCase() + res.status.slice(1)}
                  </span>
                </div>
                <div className="flex justify-between text-[11px] text-muted mb-1">
                  <span>In favour · {res.votes_for} votes ({forPct}%)</span>
                  <span>Against · {res.votes_against}</span>
                </div>
                <div className="h-[6px] bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-accent rounded-full transition-all duration-300" style={{ width: `${forPct}%` }} />
                </div>
                {canVote && (
                  <div className="flex gap-2 mt-3">
                    {hasVoted ? (
                      <span className="text-[12px] text-green font-medium">✓ Vote recorded</span>
                    ) : (
                      <>
                        <button
                          onClick={() => castVote(res.id, true)}
                          className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:opacity-90 transition-colors"
                        >
                          Vote in favour
                        </button>
                        <button
                          onClick={() => castVote(res.id, false)}
                          className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors"
                        >
                          Vote against
                        </button>
                      </>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* Upcoming AGM resolutions */}
      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">
            AGM — {new Date(upcoming.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
          </span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">Upcoming · Voting open</span>
        </div>
        {user?.role === 'resident' && (
          <div className="px-5 py-3 border-b border-border bg-accent-bg/40">
            <p className="text-[12px] text-accent font-medium">Voting is open — cast your vote on the resolutions below before the AGM.</p>
          </div>
        )}
        <div className="px-5 py-4 flex flex-col gap-4">
          {upcomingResolutions.map((res, i) => {
            const forPct = res.total_eligible > 0 ? Math.round((res.votes_for / res.total_eligible) * 100) : 0
            const hasVoted = upcomingVoted.has(res.id)
            const canVote = user?.role === 'resident' && res.status === 'open'
            return (
              <div key={res.id} className={`${i < upcomingResolutions.length - 1 ? 'pb-4 border-b border-border' : ''}`}>
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div>
                    <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {i + 1} OF {upcomingResolutions.length}</div>
                    <div className="text-[13px] font-semibold text-ink">{res.title}</div>
                    <div className="text-[12px] text-muted mt-1">{res.description}</div>
                  </div>
                  <span className="flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">
                    Voting open
                  </span>
                </div>
                <div className="flex justify-between text-[11px] text-muted mb-1">
                  <span>In favour · {res.votes_for} votes ({forPct}%)</span>
                  <span>Against · {res.votes_against}</span>
                </div>
                <div className="h-[6px] bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-accent rounded-full transition-all duration-300" style={{ width: `${forPct}%` }} />
                </div>
                {canVote && (
                  <div className="flex gap-2 mt-3">
                    {hasVoted ? (
                      <span className="text-[12px] text-green font-medium">✓ Vote recorded</span>
                    ) : (
                      <>
                        <button
                          onClick={() => castUpcomingVote(res.id, true)}
                          className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:opacity-90 transition-colors"
                        >
                          Vote in favour
                        </button>
                        <button
                          onClick={() => castUpcomingVote(res.id, false)}
                          className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors"
                        >
                          Vote against
                        </button>
                      </>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <Modal open={showScheduleModal} onClose={() => setShowScheduleModal(false)} title="Schedule AGM">
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Date</label>
            <input
              type="date"
              value={scheduleForm.date}
              onChange={e => setScheduleForm(f => ({ ...f, date: e.target.value }))}
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">Venue</label>
            <input
              type="text"
              value={scheduleForm.venue}
              onChange={e => setScheduleForm(f => ({ ...f, venue: e.target.value }))}
              placeholder="e.g. Sunridge Heights Common Room"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>
          <div className="flex justify-end gap-3 pt-1">
            <button
              onClick={() => setShowScheduleModal(false)}
              className="text-[12px] text-muted hover:text-ink px-3 py-2"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                if (!scheduleForm.date || !scheduleForm.venue) return
                setShowScheduleModal(false)
                setScheduleForm({ date: '', venue: '' })
                addToast('AGM scheduled', 'success')
              }}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
            >
              Schedule
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
