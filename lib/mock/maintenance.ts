// lib/mock/maintenance.ts

export interface MaintenanceRequest {
  id: string
  scheme_id: string
  unit_id: string | null
  title: string
  description: string
  category: 'plumbing' | 'electrical' | 'structural' | 'garden' | 'pool' | 'other'
  status: 'open' | 'in_progress' | 'pending_approval' | 'resolved'
  contractor_name: string | null
  contractor_phone: string | null
  sla_hours: number
  created_at: string
  resolved_at: string | null
  submitted_by_unit: string | null
}

export const mockMaintenanceRequests: MaintenanceRequest[] = [
  {
    id: 'mr-001',
    scheme_id: 'scheme-001',
    unit_id: 'unit-2b',
    title: 'Shower drain blocked — Unit 2B',
    description: 'Resident reports shower draining very slowly, likely hair blockage.',
    category: 'plumbing',
    status: 'in_progress',
    contractor_name: 'Rapid Plumbing Co.',
    contractor_phone: '021 555 0123',
    sla_hours: 48,
    created_at: '2025-10-14T09:00:00Z',
    resolved_at: null,
    submitted_by_unit: '2B',
  },
  {
    id: 'mr-002',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Parking bay lights not working',
    description: 'Three overhead lights in basement parking bay B not functioning.',
    category: 'electrical',
    status: 'open',
    contractor_name: null,
    contractor_phone: null,
    sla_hours: 24,
    created_at: '2025-10-15T14:30:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-003',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Pool pump replacement',
    description: 'Pool pump has failed and requires full replacement. Quote obtained.',
    category: 'pool',
    status: 'pending_approval',
    contractor_name: 'AquaFix Pool Services',
    contractor_phone: '021 555 0456',
    sla_hours: 72,
    created_at: '2025-10-12T11:00:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-004',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Garden service — monthly',
    description: 'Scheduled monthly garden maintenance and lawn cutting.',
    category: 'garden',
    status: 'resolved',
    contractor_name: 'GreenThumb Gardens',
    contractor_phone: '021 555 0789',
    sla_hours: 8,
    created_at: '2025-10-10T07:00:00Z',
    resolved_at: '2025-10-10T11:00:00Z',
    submitted_by_unit: null,
  },
  {
    id: 'mr-005',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Lift service certificate renewal',
    description: 'Annual lift inspection due. Booking with certified inspector.',
    category: 'structural',
    status: 'in_progress',
    contractor_name: 'Cape Lift Services',
    contractor_phone: '021 555 0321',
    sla_hours: 96,
    created_at: '2025-10-08T10:00:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
  {
    id: 'mr-006',
    scheme_id: 'scheme-001',
    unit_id: 'unit-4b',
    title: 'Leaking tap in kitchen — Unit 4B',
    description: 'Persistent drip from kitchen mixer tap.',
    category: 'plumbing',
    status: 'resolved',
    contractor_name: 'Rapid Plumbing Co.',
    contractor_phone: '021 555 0123',
    sla_hours: 48,
    created_at: '2025-09-20T13:00:00Z',
    resolved_at: '2025-09-22T10:00:00Z',
    submitted_by_unit: '4B',
  },
  {
    id: 'mr-007',
    scheme_id: 'scheme-001',
    unit_id: null,
    title: 'Intercom system fault — Block A',
    description: 'Intercom for units 1A–4B not ringing. Possible wiring fault.',
    category: 'electrical',
    status: 'open',
    contractor_name: null,
    contractor_phone: null,
    sla_hours: 24,
    created_at: '2025-10-16T08:45:00Z',
    resolved_at: null,
    submitted_by_unit: null,
  },
]
