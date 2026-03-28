// lib/mock/whatsapp.ts
// Simulates StrataHQ WhatsApp Bot interactions for Sunridge Heights (scheme-001)
// The bot is a virtual WhatsApp number that residents text to check balances,
// submit maintenance requests, and receive scheme notices.

export type MessageSender = 'resident' | 'bot'

export interface WhatsAppMessage {
  from: MessageSender
  text: string
  time: string   // HH:MM
  date: string   // YYYY-MM-DD
}

export interface WhatsAppThread {
  id: string
  unit_identifier: string
  owner_name: string
  connected: boolean
  last_active: string    // ISO timestamp
  unread: number
  messages: WhatsAppMessage[]
}

export interface WhatsAppBroadcast {
  id: string
  sent_at: string        // ISO timestamp
  sent_by: string
  recipient_count: number
  message: string
  type: 'levy' | 'agm' | 'maintenance' | 'general'
}

export const SCHEME_WHATSAPP_NUMBER = '+27 82 787 2848'  // fictional

export const mockWhatsAppThreads: WhatsAppThread[] = [
  {
    id: 'wt-001',
    unit_identifier: '2B',
    owner_name: 'Molefe, S.',
    connected: true,
    last_active: '2025-10-16T09:17:00Z',
    unread: 2,
    messages: [
      {
        from: 'resident',
        text: 'Hi',
        time: '09:12',
        date: '2025-10-16',
      },
      {
        from: 'bot',
        text: 'Hi Molefe! 👋 Welcome to Sunridge Heights on WhatsApp.\n\nReply with:\n1️⃣ Balance — check your levy account\n2️⃣ Request — log a maintenance request\n3️⃣ Notices — see recent scheme notices',
        time: '09:12',
        date: '2025-10-16',
      },
      {
        from: 'resident',
        text: '1',
        time: '09:15',
        date: '2025-10-16',
      },
      {
        from: 'bot',
        text: '💳 *Unit 2B Levy Account — October 2025*\n\nMonthly levy: R 2 450\nAmount paid: R 1 200\nOutstanding: R 1 250\nDue date: 1 Oct 2025 (OVERDUE)\n\nRef: SH-2B-OCT25\n\nPlease pay the outstanding amount to avoid further charges. Reply *2* to log a query.',
        time: '09:15',
        date: '2025-10-16',
      },
      {
        from: 'resident',
        text: 'ok thanks ill pay tomorrow',
        time: '09:17',
        date: '2025-10-16',
      },
      {
        from: 'bot',
        text: "No problem! I've noted that. Use reference *SH-2B-OCT25* when paying. You'll receive a confirmation here once your payment is processed. 🙏",
        time: '09:17',
        date: '2025-10-16',
      },
    ],
  },
  {
    id: 'wt-002',
    unit_identifier: '4B',
    owner_name: 'Naidoo, R.',
    connected: true,
    last_active: '2025-10-15T14:22:00Z',
    unread: 0,
    messages: [
      {
        from: 'resident',
        text: 'The light in the parking garage is not working. Has been out for 2 days now',
        time: '14:20',
        date: '2025-10-15',
      },
      {
        from: 'bot',
        text: "Thanks Naidoo! I've logged a maintenance request on your behalf.\n\n🔧 *Maintenance Request #MR-002*\nCategory: Electrical\nLocation: Parking garage\nStatus: Open\n\nYour managing agent will assign a contractor and follow up. You'll get updates here. Reply *status* anytime to check progress.",
        time: '14:20',
        date: '2025-10-15',
      },
      {
        from: 'resident',
        text: 'thank you',
        time: '14:22',
        date: '2025-10-15',
      },
      {
        from: 'bot',
        text: "You're welcome! We'll keep you posted. 👍",
        time: '14:22',
        date: '2025-10-15',
      },
    ],
  },
  {
    id: 'wt-003',
    unit_identifier: '1A',
    owner_name: 'Henderson, T.',
    connected: true,
    last_active: '2025-10-14T11:05:00Z',
    unread: 0,
    messages: [
      {
        from: 'resident',
        text: 'Did you receive my levy payment?',
        time: '11:03',
        date: '2025-10-14',
      },
      {
        from: 'bot',
        text: '✅ *Payment Confirmed — Unit 1A*\n\nOctober 2025 levy: R 2 450\nStatus: *PAID IN FULL*\nPayment received: 28 Sep 2025\nBank ref: FNB-9283471\n\nThank you Henderson! 🎉',
        time: '11:03',
        date: '2025-10-14',
      },
      {
        from: 'resident',
        text: 'great thanks',
        time: '11:05',
        date: '2025-10-14',
      },
    ],
  },
  {
    id: 'wt-004',
    unit_identifier: '5A',
    owner_name: 'Khumalo, B.',
    connected: true,
    last_active: '2025-10-13T16:40:00Z',
    unread: 0,
    messages: [
      {
        from: 'resident',
        text: 'When is the agm?',
        time: '16:38',
        date: '2025-10-13',
      },
      {
        from: 'bot',
        text: "📅 *Next AGM — Sunridge Heights*\n\nThe next Annual General Meeting is scheduled for:\n*Tuesday, 14 October 2025 at 18:30*\nVenue: Community Room, Ground Floor\n\nA formal notice with the agenda will be sent to all owners. Reply *proxy* if you can't attend and need a proxy form.",
        time: '16:38',
        date: '2025-10-13',
      },
      {
        from: 'resident',
        text: 'ill be there',
        time: '16:40',
        date: '2025-10-13',
      },
    ],
  },
  {
    id: 'wt-005',
    unit_identifier: '3A',
    owner_name: 'van der Berg, L.',
    connected: false,
    last_active: '2025-10-01T00:00:00Z',
    unread: 0,
    messages: [],
  },
]

export const mockWhatsAppBroadcasts: WhatsAppBroadcast[] = [
  {
    id: 'wb-001',
    sent_at: '2025-10-11T09:00:00Z',
    sent_by: 'Agent Portal',
    recipient_count: 7,
    type: 'agm',
    message:
      '📋 *AGM Notice — Sunridge Heights*\n\nDear Owner,\n\nPlease be advised that the Annual General Meeting will be held on *Tuesday, 14 October 2025 at 18:30* in the Community Room.\n\nAgenda and proxy forms are available in the Document Vault on StrataHQ. Reply *proxy* if you require a proxy form.',
  },
  {
    id: 'wb-002',
    sent_at: '2025-10-01T08:00:00Z',
    sent_by: 'Agent Portal',
    recipient_count: 8,
    type: 'levy',
    message:
      '💳 *October 2025 Levy Reminder*\n\nDear Owner,\n\nYour October 2025 levy of *R 2 450* is due today (1 October).\n\nPlease use your unit reference (e.g. SH-4B-OCT25) when making payment. Reply *balance* to check your account status.',
  },
  {
    id: 'wb-003',
    sent_at: '2025-09-25T10:30:00Z',
    sent_by: 'Agent Portal',
    recipient_count: 8,
    type: 'maintenance',
    message:
      '🔧 *Planned Maintenance — Pool Area*\n\nDear Resident,\n\nPlease note the pool will be closed from *Sat 28 Sep to Wed 2 Oct* for pump replacement. We apologise for the inconvenience.',
  },
]

export function getWhatsAppStats(threads: WhatsAppThread[]) {
  return {
    total_residents: threads.length,
    connected: threads.filter(t => t.connected).length,
    unread: threads.reduce((sum, t) => sum + t.unread, 0),
  }
}
