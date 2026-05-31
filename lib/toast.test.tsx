import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ToastProvider, useToast } from './toast'

function ToastTrigger() {
  const { addToast } = useToast()
  return (
    <button type="button" onClick={() => addToast('Saved', 'success')}>
      Add toast
    </button>
  )
}

describe('ToastProvider', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('clears pending dismissal timers on unmount', () => {
    vi.useFakeTimers()
    const clearTimeoutSpy = vi.spyOn(globalThis, 'clearTimeout')
    const { unmount } = render(
      <ToastProvider>
        <ToastTrigger />
      </ToastProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Add toast' }))
    expect(screen.getByText('Saved')).toBeInTheDocument()

    unmount()

    expect(clearTimeoutSpy).toHaveBeenCalledTimes(1)
  })
})
