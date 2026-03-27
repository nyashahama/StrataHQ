// lib/mock/agm.ts

export interface AgmMeeting {
  id: string
  scheme_id: string
  date: string
  quorum_required: number
  quorum_present: number
  status: 'upcoming' | 'in_progress' | 'closed'
}

export interface AgmResolution {
  id: string
  meeting_id: string
  title: string
  description: string
  votes_for: number
  votes_against: number
  total_eligible: number
  status: 'open' | 'passed' | 'failed'
}

export const mockAgmMeeting: AgmMeeting = {
  id: 'agm-2025',
  scheme_id: 'scheme-001',
  date: '2025-11-14',
  quorum_required: 25,
  quorum_present: 38,
  status: 'closed',
}

export const mockAgmResolutions: AgmResolution[] = [
  {
    id: 'res-001',
    meeting_id: 'agm-2025',
    title: 'Approval of 2026 maintenance budget',
    description: 'Proposed total maintenance budget of R485,000 for the financial year 2026, covering all scheduled and reactive maintenance.',
    votes_for: 31,
    votes_against: 7,
    total_eligible: 48,
    status: 'passed',
  },
  {
    id: 'res-002',
    meeting_id: 'agm-2025',
    title: 'Levy increase — 6% from January 2026',
    description: 'Proposed increase of standard levy from R2,450 to R2,597 per month effective 1 January 2026, in line with CPI.',
    votes_for: 26,
    votes_against: 12,
    total_eligible: 48,
    status: 'passed',
  },
  {
    id: 'res-003',
    meeting_id: 'agm-2025',
    title: 'Appointment of trustees for 2025–2026',
    description: 'Re-appointment of Henderson, T. (Chair), Molefe, S., and van der Berg, L. as trustees for the 2025–2026 term.',
    votes_for: 35,
    votes_against: 3,
    total_eligible: 48,
    status: 'passed',
  },
]

export const mockUpcomingAgm: AgmMeeting = {
  id: 'agm-2026',
  scheme_id: 'scheme-001',
  date: '2026-11-20',
  quorum_required: 25,
  quorum_present: 0,
  status: 'upcoming',
}

export const mockUpcomingResolutions: AgmResolution[] = [
  {
    id: 'res-004',
    meeting_id: 'agm-2026',
    title: 'Approval of 2027 maintenance budget',
    description: 'Proposed total maintenance budget of R520,000 for financial year 2027, reflecting a 7% increase over 2026 actuals.',
    votes_for: 0,
    votes_against: 0,
    total_eligible: 48,
    status: 'open',
  },
  {
    id: 'res-005',
    meeting_id: 'agm-2026',
    title: 'Levy increase — 5% from January 2027',
    description: 'Proposed increase of standard levy from R2,597 to R2,727 per month effective 1 January 2027, in line with projected CPI.',
    votes_for: 0,
    votes_against: 0,
    total_eligible: 48,
    status: 'open',
  },
]
