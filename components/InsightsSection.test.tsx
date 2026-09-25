import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import InsightsSection from './InsightsSection'

describe('InsightsSection', () => {
  it('uses actionable links for illustrative attention items', () => {
    render(<InsightsSection />)

    const actions = [
      ['Review arrears →', '/early-access?focus=arrears'],
      ['Send reminders →', '/early-access?focus=collection-reminders'],
      ['Model scenarios →', '/early-access?focus=reserve-risk'],
    ] as const

    for (const [label, href] of actions) {
      expect(screen.getByRole('link', { name: label })).toHaveAttribute('href', href)
      expect(screen.getByRole('link', { name: label }).getAttribute('href')).not.toBe('#')
    }
  })
})
