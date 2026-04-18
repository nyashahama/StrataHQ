'use client'

interface RetryStateProps {
  title: string
  message: string
  onRetry: () => void
}

export default function RetryState({ title, message, onRetry }: RetryStateProps) {
  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center">
        <p className="text-[14px] font-semibold text-ink mb-1">{title}</p>
        <p className="text-[13px] text-muted mb-4">{message}</p>
        <button
          onClick={onRetry}
          className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors"
        >
          Try again
        </button>
      </div>
    </div>
  )
}