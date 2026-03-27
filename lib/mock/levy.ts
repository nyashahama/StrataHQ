// lib/mock/levy.ts

export interface LevyPeriod {
  id: string
  scheme_id: string
  amount_cents: number
  due_date: string
  label: string
}

export interface LevyAccount {
  id: string
  unit_id: string
  unit_identifier: string
  owner_name: string
  period_id: string
  amount_cents: number
  paid_cents: number
  status: 'paid' | 'partial' | 'overdue' | 'pending'
  due_date: string
  paid_date: string | null
}

export interface LevyPayment {
  id: string
  levy_account_id: string
  amount_cents: number
  date: string
  reference: string
  bank_ref: string | null
}

export const mockLevyPeriod: LevyPeriod = {
  id: 'period-oct-2025',
  scheme_id: 'scheme-001',
  amount_cents: 245000,
  due_date: '2025-10-01',
  label: 'October 2025',
}

export const mockLevyRoll: LevyAccount[] = [
  { id: 'la-001', unit_id: 'unit-1a', unit_identifier: '1A', owner_name: 'Henderson, T.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-28' },
  { id: 'la-002', unit_id: 'unit-2b', unit_identifier: '2B', owner_name: 'Molefe, S.',        period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 120000, status: 'partial', due_date: '2025-10-01', paid_date: '2025-10-03' },
  { id: 'la-003', unit_id: 'unit-3a', unit_identifier: '3A', owner_name: 'van der Berg, L.', period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 0,      status: 'overdue', due_date: '2025-10-01', paid_date: null },
  { id: 'la-004', unit_id: 'unit-4b', unit_identifier: '4B', owner_name: 'Naidoo, R.',       period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-30' },
  { id: 'la-005', unit_id: 'unit-5a', unit_identifier: '5A', owner_name: 'Khumalo, B.',      period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-10-01' },
  { id: 'la-006', unit_id: 'unit-6c', unit_identifier: '6C', owner_name: 'Abrahams, J.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 0,      status: 'overdue', due_date: '2025-10-01', paid_date: null },
  { id: 'la-007', unit_id: 'unit-7b', unit_identifier: '7B', owner_name: 'Petersen, M.',    period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-09-29' },
  { id: 'la-008', unit_id: 'unit-8a', unit_identifier: '8A', owner_name: 'Dlamini, S.',     period_id: 'period-oct-2025', amount_cents: 245000, paid_cents: 245000, status: 'paid',    due_date: '2025-10-01', paid_date: '2025-10-02' },
]

export const mockCollectionTrend: { month: string; pct: number }[] = [
  { month: 'May', pct: 87 },
  { month: 'Jun', pct: 89 },
  { month: 'Jul', pct: 88 },
  { month: 'Aug', pct: 91 },
  { month: 'Sep', pct: 92 },
  { month: 'Oct', pct: 94 },
]

export const mockUnit4BPayments: LevyPayment[] = [
  { id: 'pay-001', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-09-30', reference: 'SH-4B-OCT25', bank_ref: 'FNB-9283471' },
  { id: 'pay-002', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-08-29', reference: 'SH-4B-SEP25', bank_ref: 'FNB-9274312' },
  { id: 'pay-003', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-07-31', reference: 'SH-4B-AUG25', bank_ref: 'FNB-9265193' },
  { id: 'pay-004', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-06-28', reference: 'SH-4B-JUL25', bank_ref: 'FNB-9256044' },
  { id: 'pay-005', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-05-30', reference: 'SH-4B-JUN25', bank_ref: 'FNB-9246875' },
  { id: 'pay-006', levy_account_id: 'la-004', amount_cents: 245000, date: '2025-04-29', reference: 'SH-4B-MAY25', bank_ref: 'FNB-9237706' },
]
