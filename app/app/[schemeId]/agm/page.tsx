'use client'
import { useMockAuth } from '@/lib/mock-auth'
import { mockAgmMeeting, mockAgmResolutions, mockUpcomingAgm } from '@/lib/mock/agm'

export default function AgmVotingPage() {
  const { user } = useMockAuth()

  const meeting = mockAgmMeeting
  const upcoming = mockUpcomingAgm

  const quorumPct = Math.round((meeting.quorum_present / (meeting.quorum_required * 2)) * 100)

  return (
    <div className="px-8 py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Scheme › AGM & Voting</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">AGM & Voting</h1>
      <p className="text-[14px] text-muted mb-8">Annual general meetings and trustee resolutions.</p>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4 mb-8">
        {[
          { label: 'Last AGM', value: new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Next AGM', value: new Date(upcoming.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'short', year: 'numeric' }) },
          { label: 'Quorum', value: `${meeting.quorum_present}/${meeting.quorum_required * 2}` },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-border rounded-lg px-5 py-4">
            <div className="text-[22px] font-semibold text-ink font-serif mb-1">{value}</div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {/* Meeting summary */}
      <div className="bg-white border border-border rounded-lg overflow-hidden mb-6">
        <div className="px-5 py-3 border-b border-border flex items-center justify-between">
          <span className="text-[13px] font-semibold text-ink">
            AGM — {new Date(meeting.date).toLocaleDateString('en-ZA', { day: 'numeric', month: 'long', year: 'numeric' })}
          </span>
          <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">Closed</span>
        </div>
        {/* Quorum bar */}
        <div className="px-5 py-4 border-b border-border">
          <div className="flex justify-between text-[11px] text-muted mb-2">
            <span className="font-semibold text-ink">Quorum reached ✓</span>
            <span>{meeting.quorum_present} of {meeting.quorum_required * 2} owners present</span>
          </div>
          <div className="h-[6px] bg-border rounded-full overflow-hidden">
            <div className="h-full bg-green rounded-full" style={{ width: `${Math.min(quorumPct, 100)}%` }} />
          </div>
        </div>

        {/* Resolutions */}
        <div className="px-5 py-4 flex flex-col gap-4">
          {mockAgmResolutions.map((res, i) => {
            const forPct = Math.round((res.votes_for / res.total_eligible) * 100)
            return (
              <div key={res.id} className={`${i < mockAgmResolutions.length - 1 ? 'pb-4 border-b border-border' : ''}`}>
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div>
                    <div className="text-[11px] font-semibold text-accent mb-1">RESOLUTION {i + 1} OF {mockAgmResolutions.length}</div>
                    <div className="text-[13px] font-semibold text-ink">{res.title}</div>
                    <div className="text-[12px] text-muted mt-1">{res.description}</div>
                  </div>
                  <span className={`flex-shrink-0 text-[11px] font-semibold px-2 py-[2px] rounded-full ${res.status === 'passed' ? 'bg-green-bg text-green' : 'bg-red-bg text-red'}`}>
                    {res.status.charAt(0).toUpperCase() + res.status.slice(1)}
                  </span>
                </div>
                <div className="flex justify-between text-[11px] text-muted mb-1">
                  <span>In favour · {res.votes_for} votes</span>
                  <span>Against · {res.votes_against}</span>
                </div>
                <div className="h-[6px] bg-border rounded-full overflow-hidden">
                  <div className="h-full bg-accent rounded-full" style={{ width: `${forPct}%` }} />
                </div>
                {user?.role === 'resident' && res.status === 'open' && (
                  <div className="flex gap-2 mt-3">
                    <button className="text-[12px] font-semibold bg-accent text-white px-4 py-1.5 rounded hover:bg-[#245a96] transition-colors">Vote in favour</button>
                    <button className="text-[12px] font-semibold border border-border text-ink px-4 py-1.5 rounded hover:bg-page transition-colors">Vote against</button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
