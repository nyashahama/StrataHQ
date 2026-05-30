import { describe, expect, it } from 'vitest'

import { formatDateOnly, formatShortDate } from './date-format'

describe('date-only formatting', () => {
  it('formats date-only strings without shifting in timezones behind UTC', () => {
    expect(formatShortDate('2026-04-15')).toBe('15 Apr')
    expect(formatDateOnly('2026-04-15', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      timeZone: 'America/Los_Angeles',
    })).toBe('15 Apr 2026')
  })
})
