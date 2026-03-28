import { describe, it, expect } from 'vitest'
import {
  parseBankStatementCSV,
  matchTransactions,
  summariseMatches,
  type ParsedTransaction,
} from './reconcile'
import type { LevyAccount } from './mock/levy'

// ── Helpers ────────────────────────────────────────────────────────────────

function makeAccount(overrides: Partial<LevyAccount> = {}): LevyAccount {
  return {
    id: 'acc-1',
    unit_id: 'unit-1',
    unit_identifier: '4B',
    owner_name: 'Smith, J.',
    period_id: 'period-1',
    amount_cents: 245000,
    paid_cents: 0,
    status: 'pending',
    due_date: '2025-10-01',
    paid_date: null,
    ...overrides,
  }
}

// ── parseBankStatementCSV ──────────────────────────────────────────────────

describe('parseBankStatementCSV', () => {
  it('returns empty array for empty string', () => {
    expect(parseBankStatementCSV('')).toEqual([])
  })

  it('returns empty array for header-only CSV', () => {
    expect(parseBankStatementCSV('Date,Description,Amount,Balance')).toEqual([])
  })

  it('throws when required columns are missing', () => {
    const csv = 'Notes,Ref\n2025-10-01,foo'
    expect(() => parseBankStatementCSV(csv)).toThrow(/Could not detect required columns/)
  })

  it('parses a basic CSV with standard headers', () => {
    const csv = [
      'Date,Description,Amount,Balance',
      '2025-10-01,UNIT 4B LEVY,2450.00,10000.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(1)
    expect(result[0].amount_cents).toBe(245000)
    expect(result[0].description).toBe('UNIT 4B LEVY')
    expect(result[0].date).toBe('2025-10-01')
    expect(result[0].running_balance_cents).toBe(1000000)
  })

  it('skips rows with zero or negative amounts (debits)', () => {
    const csv = [
      'Date,Description,Amount,Balance',
      '2025-10-01,DEBIT CHARGE,-50.00,9950.00',
      '2025-10-02,UNIT 4B LEVY,2450.00,12400.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(1)
    expect(result[0].description).toBe('UNIT 4B LEVY')
  })

  it('handles quoted fields containing commas', () => {
    const csv = [
      'Date,Description,Amount,Balance',
      '2025-10-01,"Smith, J. UNIT 5A",1500.00,8000.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(1)
    expect(result[0].description).toBe('Smith, J. UNIT 5A')
  })

  it('handles alternative column names (Narration, Credit)', () => {
    const csv = [
      'Date,Narration,Credit,Balance',
      '2025-10-01,UNIT 3C LEVY,1800.00,5000.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(1)
    expect(result[0].amount_cents).toBe(180000)
  })

  it('handles missing balance column gracefully', () => {
    const csv = [
      'Date,Description,Amount',
      '2025-10-01,UNIT 4B LEVY,2450.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(1)
    expect(result[0].running_balance_cents).toBe(0)
  })

  it('parses multiple rows correctly', () => {
    const csv = [
      'Date,Description,Amount,Balance',
      '2025-10-01,UNIT 1A LEVY,2450.00,10000.00',
      '2025-10-02,UNIT 2B LEVY,2450.00,7550.00',
      '2025-10-03,UNIT 3C LEVY,2450.00,5100.00',
    ].join('\n')

    const result = parseBankStatementCSV(csv)
    expect(result).toHaveLength(3)
  })
})

// ── matchTransactions ──────────────────────────────────────────────────────

describe('matchTransactions', () => {
  const tx = (description: string, amount_cents = 245000): ParsedTransaction => ({
    date: '2025-10-01',
    description,
    amount_cents,
    running_balance_cents: 0,
    raw: '',
  })

  describe('HIGH confidence — unit identifier', () => {
    it('matches "UNIT 4B" pattern', () => {
      const result = matchTransactions([tx('UNIT 4B LEVY PAYMENT')], [makeAccount()])
      expect(result[0].confidence).toBe('high')
      expect(result[0].account?.id).toBe('acc-1')
    })

    it('matches "UNIT4B" (no space) pattern', () => {
      const result = matchTransactions([tx('UNIT4B OCT LEVY')], [makeAccount()])
      expect(result[0].confidence).toBe('high')
    })

    it('matches "4B LEVY" pattern', () => {
      const result = matchTransactions([tx('4B LEVY PAYMENT')], [makeAccount()])
      expect(result[0].confidence).toBe('high')
    })

    it('matches "/4B/" pattern', () => {
      const result = matchTransactions([tx('PAYMENT/4B/OCT2025')], [makeAccount()])
      expect(result[0].confidence).toBe('high')
    })
  })

  describe('MEDIUM confidence — surname + amount', () => {
    it('matches surname in description with matching amount', () => {
      const result = matchTransactions(
        [tx('SMITH OCT LEVY', 245000)],
        [makeAccount({ owner_name: 'Smith, J.' })]
      )
      expect(result[0].confidence).toBe('medium')
    })

    it('matches partial payment (less than full amount) with surname', () => {
      const result = matchTransactions(
        [tx('SMITH PART PAYMENT', 100000)],
        [makeAccount({ owner_name: 'Smith, J.', amount_cents: 245000 })]
      )
      expect(result[0].confidence).toBe('medium')
    })

    it('does not match if surname is too short (< 3 chars)', () => {
      const result = matchTransactions(
        [tx('LI OCT LEVY', 245000)],
        [makeAccount({ owner_name: 'Li, J.', unit_identifier: 'X99' })]
      )
      // Short surname should not trigger medium match
      expect(result[0].confidence).not.toBe('medium')
    })
  })

  describe('LOW confidence — exact amount only', () => {
    it('matches when only one unpaid account has the exact amount', () => {
      const accounts = [
        makeAccount({ id: 'acc-1', unit_identifier: 'Z1', owner_name: 'Jones, A.', amount_cents: 245000 }),
      ]
      const result = matchTransactions([tx('UNKNOWN PAYMENT', 245000)], accounts)
      expect(result[0].confidence).toBe('low')
    })

    it('does not low-match when multiple accounts share the same amount', () => {
      const accounts = [
        makeAccount({ id: 'acc-1', unit_identifier: 'Z1', owner_name: 'Jones, A.', amount_cents: 245000 }),
        makeAccount({ id: 'acc-2', unit_identifier: 'Z2', owner_name: 'Brown, B.', amount_cents: 245000 }),
      ]
      const result = matchTransactions([tx('UNKNOWN PAYMENT', 245000)], accounts)
      expect(result[0].confidence).toBe('unmatched')
    })

    it('does not low-match against already-paid accounts', () => {
      const accounts = [
        makeAccount({ id: 'acc-1', unit_identifier: 'Z1', owner_name: 'Jones, A.', status: 'paid' }),
      ]
      const result = matchTransactions([tx('UNKNOWN PAYMENT', 245000)], accounts)
      expect(result[0].confidence).toBe('unmatched')
    })
  })

  describe('UNMATCHED', () => {
    it('returns unmatched when no patterns fit', () => {
      const result = matchTransactions(
        [tx('RANDOM GIBBERISH PAYMENT', 999)],
        [makeAccount()]
      )
      expect(result[0].confidence).toBe('unmatched')
      expect(result[0].account).toBeNull()
    })
  })

  describe('deduplication — each account matched at most once', () => {
    it('does not reuse an account across multiple transactions', () => {
      const accounts = [makeAccount()]
      const txs = [tx('UNIT 4B PAYMENT'), tx('UNIT 4B PAYMENT')]
      const results = matchTransactions(txs, accounts)
      // First should match, second should be unmatched (account already used)
      expect(results[0].confidence).toBe('high')
      expect(results[1].confidence).toBe('unmatched')
    })
  })

  describe('new_status calculation', () => {
    it('sets status to "paid" when payment covers full amount', () => {
      const result = matchTransactions(
        [tx('UNIT 4B', 245000)],
        [makeAccount({ paid_cents: 0, amount_cents: 245000 })]
      )
      expect(result[0].new_status).toBe('paid')
      expect(result[0].new_paid_cents).toBe(245000)
    })

    it('sets status to "partial" when payment is less than full amount', () => {
      const result = matchTransactions(
        [tx('UNIT 4B', 100000)],
        [makeAccount({ paid_cents: 0, amount_cents: 245000 })]
      )
      expect(result[0].new_status).toBe('partial')
    })

    it('sets status to "paid" when partial + new payment covers full amount', () => {
      const result = matchTransactions(
        [tx('UNIT 4B', 145000)],
        [makeAccount({ paid_cents: 100000, amount_cents: 245000 })]
      )
      expect(result[0].new_status).toBe('paid')
      expect(result[0].new_paid_cents).toBe(245000)
    })
  })
})

// ── summariseMatches ──────────────────────────────────────────────────────

describe('summariseMatches', () => {
  it('returns zeros for empty array', () => {
    const summary = summariseMatches([])
    expect(summary.total_transactions).toBe(0)
    expect(summary.matched_high).toBe(0)
    expect(summary.unmatched).toBe(0)
    expect(summary.total_amount_cents).toBe(0)
  })

  it('counts confidence levels correctly', () => {
    const accounts = [
      makeAccount({ id: 'acc-1', unit_identifier: '4B' }),
      makeAccount({ id: 'acc-2', unit_identifier: 'Z9', owner_name: 'Brown, B.' }),
      makeAccount({ id: 'acc-3', unit_identifier: 'X1', owner_name: 'Xyz, A.', amount_cents: 99900 }),
    ]
    const txs: ParsedTransaction[] = [
      { date: '2025-10-01', description: 'UNIT 4B LEVY', amount_cents: 245000, running_balance_cents: 0, raw: '' },
      { date: '2025-10-01', description: 'BROWN OCT LEVY', amount_cents: 245000, running_balance_cents: 0, raw: '' },
      { date: '2025-10-01', description: 'COMPLETELY UNKNOWN', amount_cents: 1, running_balance_cents: 0, raw: '' },
    ]
    const matches = matchTransactions(txs, accounts)
    const summary = summariseMatches(matches)

    expect(summary.total_transactions).toBe(3)
    expect(summary.matched_high).toBe(1)
    expect(summary.matched_medium).toBe(1)
    expect(summary.unmatched).toBe(1)
    expect(summary.total_amount_cents).toBe(245000 + 245000 + 1)
    expect(summary.matched_amount_cents).toBe(245000 + 245000)
  })
})
