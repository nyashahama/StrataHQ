'use client'
import { useTheme } from '@/components/ThemeProvider'

export default function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const isDark = theme === 'dark'

  return (
    <button
      onClick={toggle}
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      className="flex items-center gap-2 px-3 py-[7px] text-[12px] text-muted hover:text-ink hover:bg-hover-subtle w-full transition-colors"
    >
      {isDark ? (
        <>
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
            <circle cx="7" cy="7" r="2.5" />
            <path d="M7 1v1.5M7 11.5V13M1 7h1.5M11.5 7H13M3.05 3.05l1.06 1.06M9.9 9.9l1.06 1.06M3.05 10.95l1.06-1.06M9.9 4.1l1.06-1.06" />
          </svg>
          Light mode
        </>
      ) : (
        <>
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
            <path d="M11.5 8.5A5.5 5.5 0 0 1 5.5 2.5a5.5 5.5 0 1 0 6 6z" />
          </svg>
          Dark mode
        </>
      )}
    </button>
  )
}
