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

function DoubleToastTrigger() {
  const { addToast } = useToast()
  return (
    <button type="button" onClick={() => {
      addToast('Saved', 'success')
      addToast('Saved again', 'info')
    }}>
      Add two toasts
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

  it('assigns unique ids for rapid toasts', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    render(
      <ToastProvider>
        <DoubleToastTrigger />
      </ToastProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Add two toasts' }))

    expect(screen.getByText('Saved')).toBeInTheDocument()
    expect(screen.getByText('Saved again')).toBeInTheDocument()
    expect(errorSpy).not.toHaveBeenCalledWith(
      expect.stringContaining('Encountered two children with the same key'),
    )
    
    errorSpy.mockRestore()
  })
})
