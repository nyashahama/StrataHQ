import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import AppShell from './AppShell'
import { usePathname } from 'next/navigation'

vi.mock('next/navigation', () => ({
  usePathname: vi.fn(),
}))

describe('AppShell', () => {
  it('closes sidebar when the path changes', () => {
    const pathname = vi.mocked(usePathname)
    pathname.mockReturnValue('/app/first')

    const { rerender } = render(
      <AppShell sidebar={<div>Sidebar content</div>}>
        <div>Shell content</div>
      </AppShell>,
    )

    fireEvent.click(screen.getByLabelText('Open sidebar'))
    expect(document.body.style.overflow).toBe('hidden')

    pathname.mockReturnValue('/app/second')
    rerender(
      <AppShell sidebar={<div>Sidebar content</div>}>
        <div>Shell content</div>
      </AppShell>,
    )

    expect(document.body.style.overflow).toBe('')
    document.body.style.overflow = ''
  })
})
