// lib/mock/members.ts

export interface Member {
  id: string
  scheme_id: string
  unit_id: string
  unit_identifier: string
  name: string
  role: 'owner' | 'trustee' | 'resident'
  email: string
  phone: string | null
  is_trustee_committee: boolean
}

export const mockMembers: Member[] = [
  { id: 'mem-001', scheme_id: 'scheme-001', unit_id: 'unit-1a', unit_identifier: '1A', name: 'Henderson, T.',    role: 'trustee',  email: 'thenderson@email.co.za',  phone: '082 555 0101', is_trustee_committee: true },
  { id: 'mem-002', scheme_id: 'scheme-001', unit_id: 'unit-2b', unit_identifier: '2B', name: 'Molefe, S.',        role: 'trustee',  email: 'smolefe@email.co.za',      phone: '083 555 0202', is_trustee_committee: true },
  { id: 'mem-003', scheme_id: 'scheme-001', unit_id: 'unit-3a', unit_identifier: '3A', name: 'van der Berg, L.', role: 'trustee',  email: 'lvanderberg@email.co.za',  phone: '084 555 0303', is_trustee_committee: true },
  { id: 'mem-004', scheme_id: 'scheme-001', unit_id: 'unit-4b', unit_identifier: '4B', name: 'Naidoo, R.',       role: 'owner',    email: 'rnaidoo@email.co.za',      phone: '071 555 0404', is_trustee_committee: false },
  { id: 'mem-005', scheme_id: 'scheme-001', unit_id: 'unit-5a', unit_identifier: '5A', name: 'Khumalo, B.',      role: 'owner',    email: 'bkhumalo@email.co.za',     phone: '072 555 0505', is_trustee_committee: false },
  { id: 'mem-006', scheme_id: 'scheme-001', unit_id: 'unit-6c', unit_identifier: '6C', name: 'Abrahams, J.',    role: 'owner',    email: 'jabrahams@email.co.za',    phone: null,           is_trustee_committee: false },
  { id: 'mem-007', scheme_id: 'scheme-001', unit_id: 'unit-7b', unit_identifier: '7B', name: 'Petersen, M.',    role: 'resident', email: 'mpetersen@email.co.za',    phone: '073 555 0707', is_trustee_committee: false },
  { id: 'mem-008', scheme_id: 'scheme-001', unit_id: 'unit-8a', unit_identifier: '8A', name: 'Dlamini, S.',     role: 'owner',    email: 'sdlamini@email.co.za',     phone: '074 555 0808', is_trustee_committee: false },
]
