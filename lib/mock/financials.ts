// lib/mock/financials.ts

export interface BudgetLine {
  id: string
  scheme_id: string
  category: string
  budgeted_cents: number
  actual_cents: number
  period_label: string
}

export interface ReserveFund {
  scheme_id: string
  balance_cents: number
  target_cents: number
  last_updated: string
}

export const mockBudgetLines: BudgetLine[] = [
  { id: 'bl-001', scheme_id: 'scheme-001', category: 'Administration',       budgeted_cents: 4800000,  actual_cents: 4620000,  period_label: '2025' },
  { id: 'bl-002', scheme_id: 'scheme-001', category: 'Cleaning',             budgeted_cents: 3600000,  actual_cents: 3600000,  period_label: '2025' },
  { id: 'bl-003', scheme_id: 'scheme-001', category: 'Maintenance',          budgeted_cents: 18500000, actual_cents: 21340000, period_label: '2025' },
  { id: 'bl-004', scheme_id: 'scheme-001', category: 'Insurance',            budgeted_cents: 9600000,  actual_cents: 9600000,  period_label: '2025' },
  { id: 'bl-005', scheme_id: 'scheme-001', category: 'Electricity (common)', budgeted_cents: 7200000,  actual_cents: 6890000,  period_label: '2025' },
  { id: 'bl-006', scheme_id: 'scheme-001', category: 'Landscaping',          budgeted_cents: 2400000,  actual_cents: 2280000,  period_label: '2025' },
  { id: 'bl-007', scheme_id: 'scheme-001', category: 'Pool maintenance',     budgeted_cents: 1800000,  actual_cents: 3960000,  period_label: '2025' },
  { id: 'bl-008', scheme_id: 'scheme-001', category: 'Reserve levy',         budgeted_cents: 600000,   actual_cents: 546000,   period_label: '2025' },
]

export const mockReserveFund: ReserveFund = {
  scheme_id: 'scheme-001',
  balance_cents: 18450000,
  target_cents: 36000000,
  last_updated: '2025-10-01T00:00:00Z',
}
