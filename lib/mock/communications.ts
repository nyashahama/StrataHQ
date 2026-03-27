// lib/mock/communications.ts

export interface Notice {
  id: string
  scheme_id: string
  title: string
  body: string
  sent_at: string
  sent_by_name: string
  type: 'general' | 'urgent' | 'agm' | 'levy'
}

export const mockNotices: Notice[] = [
  {
    id: 'notice-001',
    scheme_id: 'scheme-001',
    title: 'Annual General Meeting — 14 November 2025',
    body: 'Dear Owners,\n\nYou are hereby notified that the Annual General Meeting of Sunridge Heights Body Corporate will be held on Thursday, 14 November 2025 at 18:30 in the complex communal room.\n\nAgenda: (1) Approval of 2026 maintenance budget, (2) Levy increase, (3) Trustee elections.\n\nProxy forms must be submitted by 12 November 2025.',
    sent_at: '2025-10-24T10:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'agm',
  },
  {
    id: 'notice-002',
    scheme_id: 'scheme-001',
    title: 'Urgent: Water supply interruption — 18 October 2025',
    body: 'Dear Residents,\n\nPlease be advised that the City of Cape Town will be conducting maintenance on the main water supply line on Saturday 18 October 2025 from 08:00–14:00. All units will experience no water supply during this period.\n\nPlease make alternative arrangements.',
    sent_at: '2025-10-16T14:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'urgent',
  },
  {
    id: 'notice-003',
    scheme_id: 'scheme-001',
    title: 'October levy reminder',
    body: 'Dear Owners,\n\nThis is a reminder that October levies of R2,450 were due on 1 October 2025. If you have not yet made payment, please do so immediately to avoid additional administration fees.\n\nPayment reference: SH-[UNIT]-OCT25',
    sent_at: '2025-10-07T09:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'levy',
  },
  {
    id: 'notice-004',
    scheme_id: 'scheme-001',
    title: 'Pool pump replacement — update',
    body: 'Dear Residents,\n\nThe pool pump has been confirmed as beyond repair. A replacement quote of R8,400 (inclusive) has been received from AquaFix Pool Services. The trustees are reviewing the quote for approval.\n\nThe pool will remain closed until the replacement is complete. We apologise for the inconvenience.',
    sent_at: '2025-10-13T11:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'general',
  },
  {
    id: 'notice-005',
    scheme_id: 'scheme-001',
    title: 'Year-end building inspection — 5 December 2025',
    body: 'Dear Residents,\n\nPlease be advised that our annual building inspection will take place on Friday, 5 December 2025. The inspector will require access to all units between 09:00 and 15:00. If you are unable to be present, please make arrangements with the managing agent by 28 November 2025.',
    sent_at: '2025-10-01T08:00:00Z',
    sent_by_name: 'Acme Property Management',
    type: 'general',
  },
]
