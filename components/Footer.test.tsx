import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import Footer from './Footer'

describe('Footer', () => {
  it('uses concrete link targets for section links', () => {
    render(<Footer />)

    const expected = {
      Features: '/#features',
      Modules: '/#modules',
      Pricing: '/#pricing',
      Changelog: '/#roles',
      'Documentation': '/#features',
      'STSMA guide': '/#problem',
      'Blog': '/early-access',
      'Help centre': '/auth/login',
      About: '/#roles',
      Contact: '/#problem',
      'Privacy policy': '/auth/login',
      Terms: '/auth/login',
    } as const

    for (const [label, href] of Object.entries(expected)) {
      expect(screen.getByRole('link', { name: label })).toHaveAttribute('href', href)
    }

    for (const link of screen.getAllByRole('link')) {
      expect(link.getAttribute('href')).not.toBe('#')
      expect(link.getAttribute('href')).toBeTruthy()
    }
  })
})
