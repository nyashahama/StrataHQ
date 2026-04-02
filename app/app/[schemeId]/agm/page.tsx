'use client'

import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'next/navigation'

import Modal from '@/components/Modal'
import { useAuth } from '@/lib/auth'
import { assignAgmProxy, castAgmVote, getAgmDashboard, scheduleAgmMeeting } from '@/lib/agm-api'
import type { AgmDashboard, AgmMeetingInfo, AgmResolutionInfo, AgmVoteChoice } from '@/lib/agm'
import { listSchemeMembers } from '@/lib/scheme-api'
import { useToast } from '@/lib/toast'

const EMPTY_RESOLUTION = { title: '', description: '' }

function formatDate(value?: string | null) {
  if (!value) return 'Not scheduled'
  return new Date(value).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' })
}

function meetingStatusLabel(meeting: AgmMeetingInfo) {
  if (meeting.status === 'closed') return 'Closed'
  if (meeting.status === 'in_progress') return 'In progress'
  return 'Upcoming'
}

function meetingStatusStyle(meeting: AgmMeetingInfo) {
  if (meeting.status === 'closed') return 'bg-green-bg text-green'
  if (meeting.status === 'in_progress') return 'bg-accent-bg text-accent'
  return 'bg-yellowbg text-amber'
}

function resolutionStatusLabel(resolution: AgmResolutionInfo) {
  if (resolution.status === 'open') return 'Voting open'
  return resolution.status.charAt(0).toUpperCase() + resolution.status.slice(1)
}

function resolutionStatusStyle(resolution: AgmResolutionInfo) {
  if (resolution.status === 'passed') return 'bg-green-bg text-green'
  if (resolution.status === 'failed') return 'bg-red-bg text-red'
  return 'bg-yellowbg text-amber'
}

export default function AgmVotingPage() {
  const { user } = useAuth()
  const { addToast } = useToast()
  const params = useParams()
  const schemeId = params.schemeId as string

  const [dashboard, setDashboard] = useState<AgmDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingMembers, setLoadingMembers] = useState(true)
  const [proxyCandidates, setProxyCandidates] = useState<Array<{ user_id: string; full_name: string; role: string }>>([])
  const [showScheduleModal, setShowScheduleModal] = useState(false)
  const [savingSchedule, setSavingSchedule] = useState(false)
  const [votingResolutionId, setVotingResolutionId] = useState<string | null>(null)
  const [assigningProxy, setAssigningProxy] = useState(false)
  const [proxyUserId, setProxyUserId] = useState('')
  const [scheduleForm, setScheduleForm] = useState({
    date: '',
    quorum_required: '1',
    resolutions: [
      { ...EMPTY_RESOLUTION },
      { ...EMPTY_RESOLUTION },
    ],
  })

  const isAdmin = user?.role === 'admin'
  const canVote = user?.role === 'resident' || user?.role === 'trustee'
  const latestMeeting = dashboard?.latest ?? null
  const upcomingMeeting = dashboard?.upcoming ?? null

  const selectedProxyLabel = useMemo(
    () => proxyCandidates.find(candidate => candidate.user_id === upcomingMeeting?.user_proxy_grantee_id)?.full_name ?? '',
    [proxyCandidates, upcomingMeeting?.user_proxy_grantee_id],
  )

  useEffect(() => {
    async function load() {
      try {
        setLoading(true)
        setDashboard(await getAgmDashboard(schemeId))
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : 'Failed to load AGM dashboard',
          'error',
        )
      } finally {
        setLoading(false)
      }
    }

    load()
  }, [addToast, schemeId])

  useEffect(() => {
    async function loadMembers() {
      try {
        setLoadingMembers(true)
        const members = await listSchemeMembers(schemeId)
        setProxyCandidates(members)
      } catch {
        setProxyCandidates([])
      } finally {
        setLoadingMembers(false)
      }
    }

    loadMembers()
  }, [schemeId])

  useEffect(() => {
    setProxyUserId(upcomingMeeting?.user_proxy_grantee_id ?? '')
  }, [upcomingMeeting?.user_proxy_grantee_id])

  async function refreshDashboard() {
    setDashboard(await getAgmDashboard(schemeId))
  }

  async function handleVote(resolutionId: string, choice: AgmVoteChoice) {
    setVotingResolutionId(resolutionId)
    try {
      await castAgmVote(schemeId, resolutionId, choice)
      await refreshDashboard()
      addToast(choice === 'for' ? 'Vote in favour recorded' : 'Vote against recorded', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to cast vote',
        'error',
      )
    } finally {
      setVotingResolutionId(null)
    }
  }

  async function handleAssignProxy() {
    if (!upcomingMeeting || !proxyUserId) return

    setAssigningProxy(true)
    try {
      await assignAgmProxy(schemeId, upcomingMeeting.id, proxyUserId)
      await refreshDashboard()
      addToast('Proxy assigned for the upcoming AGM', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to assign proxy',
        'error',
      )
    } finally {
      setAssigningProxy(false)
    }
  }

  async function handleSchedule() {
    const resolutions = scheduleForm.resolutions
      .map(resolution => ({
        title: resolution.title.trim(),
        description: resolution.description.trim(),
      }))
      .filter(resolution => resolution.title && resolution.description)

    if (!scheduleForm.date || resolutions.length === 0) return

    setSavingSchedule(true)
    try {
      await scheduleAgmMeeting(schemeId, {
        date: scheduleForm.date,
        quorum_required: Math.max(1, Number(scheduleForm.quorum_required) || 1),
        resolutions,
      })
      setShowScheduleModal(false)
      setScheduleForm({
        date: '',
        quorum_required: '1',
        resolutions: [{ ...EMPTY_RESOLUTION }, { ...EMPTY_RESOLUTION }],
      })
      await refreshDashboard()
      addToast('AGM scheduled', 'success')
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : 'Failed to schedule AGM',
        'error',
      )
    } finally {
      setSavingSchedule(false)
    }
  }

  if (loading) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading AGM and voting…
        </div>
      </div>
    )
  }

  const lastQuorumDenominator = latestMeeting ? Math.max(latestMeeting.quorum_required, 1) : 1
  const lastQuorumPct = latestMeeting
    ? Math.min(100, Math.round((latestMeeting.quorum_present / lastQuorumDenominator) * 100))
    : 0

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › AGM & Voting</p>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-1">
        <h1 className="font-serif text-[28px] font-semibold text-ink">AGM & Voting</h1>
        {isAdmin && (
          <button
            onClick={() => setShowScheduleModal(true)}
            className="text-[12px] font-semibold bg-accent text-white px-3 py-2 rounded hover:opacity-90 transition-colors"
          >
            + Schedule AGM
          </button>
        )}
      </div>
      <p className="text-[14px] text-muted mb-8">Annual general meetings, resolutions, proxy assignments, and member voting.</p>

      <div className="grid grid-cols-3 gap-3 sm:gap-4 mb-8">
        {[
          { label: 'Last AGM', value: formatDate(latestMeeting?.date) },
          { label: 'Next AGM', value: formatDate(upcomingMeeting?.date) },
          { label: 'Quorum', value: latestMeeting ? `${latestMeeting.quorum_present}/${latestMeeting.quorum_required}` : 'No record' },
        ].map(({ label, value }) => (
          <div key={label} className="bg-surface border border-border rounded-lg px-3 sm:px-5 py-4">
            <div className="text-[16px] sm:text-[22px] font-semibold text-ink font-serif mb-1 leading-tight">{value}</div>
            <div className="text-[11px] sm:text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {latestMeeting ? (
        <div className="bg-surface border border-border rounded-lg overflow-hidden mb-6">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">
              AGM — {new Date(latestMeeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
            </span>
            <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${meetingStatusStyle(latestMeeting)}`}>
              {meetingStatusLabel(latestMeeting)}
            </span>
          </div>
          <div className="px-5 py-4 border-b border-border">
            <div className="flex justify-between text-[11px] text-muted mb-2">
              <span className="font-semibold text-ink">Quorum present</span>
              <span>{latestMeeting.quorum_present} of {latestMeeting.quorum_required} members represented</span>
            </div>
            <div className="h-[6px] bg-border rounded-full overflow-hidden">
              <div className="h-full bg-green rounded-full" style={{ width: `${lastQuorumPct}%` }} />
            </div>
          </div>
          <MeetingResolutions
            canVote={false}
            meeting={latestMeeting}
            votingResolutionId={votingResolutionId}
            onVote={handleVote}
          />
        </div>
      ) : (
        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-10 text-center text-muted text-[14px] mb-6">
          No previous AGM has been recorded for this scheme yet.
        </div>
      )}

      {upcomingMeeting ? (
        <div className="bg-surface border border-border rounded-lg overflow-hidden">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">
              AGM — {new Date(upcomingMeeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
            </span>
            <span className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${meetingStatusStyle(upcomingMeeting)}`}>
              {meetingStatusLabel(upcomingMeeting)}
            </span>
          </div>
          {canVote && (
            <div className="px-5 py-3 border-b border-border bg-accent-bg/40">
              <p className="text-[12px] text-accent font-medium">Voting is open on the resolutions below ahead of the AGM.</p>
            </div>
          )}
          {canVote && (
            <div className="px-5 py-4 border-b border-border bg-page/50">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <div className="text-[12px] font-semibold text-ink mb-1">Assign proxy</div>
                  <div className="text-[12px] text-muted">
                    {upcomingMeeting.user_proxy_grantee_id && selectedProxyLabel
                      ? `Current proxy: ${selectedProxyLabel}`
                      : 'Select another member to carry your vote if you cannot attend.'}
                  </div>
                </div>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <select
                    value={proxyUserId}
                    onChange={event => setProxyUserId(event.target.value)}
                    disabled={loadingMembers || assigningProxy || !!upcomingMeeting.user_proxy_grantee_id}
                    className="border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent disabled:opacity-50"
                  >
                    <option value="">Select proxy member</option>
                    {proxyCandidates
                      .filter(candidate => candidate.user_id !== user?.id)
                      .map(candidate => (
                        <option key={candidate.user_id} value={candidate.user_id}>
                          {candidate.full_name} · {candidate.role}
                        </option>
                      ))}
                  </select>
                  <button
                    onClick={handleAssignProxy}
                    disabled={!proxyUserId || assigningProxy || !!upcomingMeeting.user_proxy_grantee_id}
                    className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {assigningProxy ? 'Assigning…' : upcomingMeeting.user_proxy_grantee_id ? 'Proxy assigned' : 'Assign proxy'}
                  </button>
                </div>
              </div>
            </div>
          )}
          <MeetingResolutions
            canVote={canVote}
            meeting={upcomingMeeting}
            votingResolutionId={votingResolutionId}
            onVote={handleVote}
          />
        </div>
      ) : (
        <div className="bg-[#f0efe9] border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No AGM is currently scheduled.
        </div>
      )}

      <Modal open={showScheduleModal} onClose={() => !savingSchedule && setShowScheduleModal(false)} title="Schedule AGM">
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Date</label>
              <input
                type="date"
                value={scheduleForm.date}
                onChange={event => setScheduleForm(current => ({ ...current, date: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">Quorum required</label>
              <input
                type="number"
                min="1"
                value={scheduleForm.quorum_required}
                onChange={event => setScheduleForm(current => ({ ...current, quorum_required: event.target.value }))}
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
            </div>
          </div>

          <div className="flex items-center justify-between">
            <div className="text-[12px] font-semibold text-ink">Resolutions</div>
            <button
              onClick={() => setScheduleForm(current => ({
                ...current,
                resolutions: [...current.resolutions, { ...EMPTY_RESOLUTION }],
              }))}
              className="text-[12px] font-semibold text-accent hover:underline"
            >
              + Add resolution
            </button>
          </div>

          <div className="flex flex-col gap-3">
            {scheduleForm.resolutions.map((resolution, index) => (
              <div key={index} className="border border-border rounded-lg p-3 bg-page/40">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-[12px] font-semibold text-ink">Resolution {index + 1}</span>
                  {scheduleForm.resolutions.length > 1 && (
                    <button
                      onClick={() => setScheduleForm(current => ({
                        ...current,
                        resolutions: current.resolutions.filter((_, itemIndex) => itemIndex !== index),
                      }))}
                      className="text-[11px] text-muted hover:text-red"
                    >
                      Remove
                    </button>
                  )}
                </div>
                <div className="flex flex-col gap-2">
                  <input
                    type="text"
                    value={resolution.title}
                    onChange={event => setScheduleForm(current => ({
                      ...current,
                      resolutions: current.resolutions.map((item, itemIndex) => itemIndex === index ? { ...item, title: event.target.value } : item),
                    }))}
                    placeholder="Resolution title"
                    className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
                  />
                  <textarea
                    value={resolution.description}
                    onChange={event => setScheduleForm(current => ({
                      ...current,
                      resolutions: current.resolutions.map((item, itemIndex) => itemIndex === index ? { ...item, description: event.target.value } : item),
                    }))}
                    placeholder="Resolution description"
                    rows={3}
                    className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent resize-none"
                  />
                </div>
              </div>
            ))}
          </div>

          <div className="flex justify-end gap-3 pt-1">
            <button
              onClick={() => setShowScheduleModal(false)}
              className="text-[12px] text-muted hover:text-ink px-3 py-2"
            >
              Cancel
            </button>
            <button
              onClick={handleSchedule}
              disabled={!scheduleForm.date || savingSchedule}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {savingSchedule ? 'Scheduling…' : 'Schedule'}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

function MeetingResolutions({
  canVote,
  meeting,
  votingResolutionId,
  onVote,
}: {
  canVote: boolean
  meeting: AgmMeetingInfo
  votingResolutionId: string | null
  onVote: (resolutionId: string, choice: AgmVoteChoice) => Promise<void>
}) {
  return (
    <div className="px-5 py-4 flex flex-col gap-4">
      {meeting.resolutions.length === 0 ? (
        <div className="text-[13px] text-muted">No resolutions have been added to this AGM yet.</div>
      ) : meeting.resolutions.map((resolution, index) => {
        const totalEligible = Math.max(resolution.total_eligible, 1)
        const forPct = Math.round((resolution.votes_for / totalEligible) * 100)
        const canVoteOnResolution = canVote && resolution.status === 'open' && !resolution.user_vote

        return (
          <div key={resolution.id} className={index < meeting.resolutions.length - 1 ? 'pb-4 border-b border-border' : ''}>
            <div className="flex items-start justify-between gap-4 mb-2">
              <div>
                <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {index + 1} OF {meeting.resolutions.length}</div>
                <div className="text-[13px] font-semibold text-ink">{resolution.title}</div>
                <div className="text-[12px] text-muted mt-1">{resolution.description}</div>
              </div>
              <span className={`flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full ${resolutionStatusStyle(resolution)}`}>
                {resolutionStatusLabel(resolution)}
              </span>
            </div>
            <div className="flex justify-between text-[11px] text-muted mb-1">
              <span>In favour · {resolution.votes_for} votes ({forPct}%)</span>
              <span>Against · {resolution.votes_against}</span>
            </div>
            <div className="h-[6px] bg-border rounded-full overflow-hidden">
              <div className="h-full bg-accent rounded-full transition-all duration-300" style={{ width: `${forPct}%` }} />
            </div>
            {resolution.user_vote && (
              <div className="text-[12px] text-green font-medium mt-3">✓ Your vote: {resolution.user_vote}</div>
            )}
            {canVoteOnResolution && (
              <div className="flex gap-2 mt-3">
                <button
                  onClick={() => onVote(resolution.id, 'for')}
                  disabled={votingResolutionId === resolution.id}
                  className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {votingResolutionId === resolution.id ? 'Saving…' : 'Vote in favour'}
                </button>
                <button
                  onClick={() => onVote(resolution.id, 'against')}
                  disabled={votingResolutionId === resolution.id}
                  className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Vote against
                </button>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
