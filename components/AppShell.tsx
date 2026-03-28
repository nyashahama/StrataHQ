'use client'
import { useState, useEffect } from 'react'
import LogoIcon from '@/components/LogoIcon'

interface AppShellProps {
  sidebar: React.ReactNode
  children: React.ReactNode
  headerLabel?: string
}

export default function AppShell({ sidebar, children, headerLabel }: AppShellProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false)

  // Close sidebar on route change (escape key)
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setSidebarOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // Prevent body scroll when mobile sidebar is open
  useEffect(() => {
    document.body.style.overflow = sidebarOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [sidebarOpen])

  return (
    <div className="flex h-screen overflow-hidden bg-page">

      {/* Mobile backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-ink/40 backdrop-blur-[2px] md:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar — off-canvas on mobile, static on desktop */}
      <div
        className={[
          'fixed inset-y-0 left-0 z-40 flex-shrink-0',
          'transition-transform duration-300 ease-in-out',
          'md:static md:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        ].join(' ')}
      >
        {sidebar}
      </div>

      {/* Right side: mobile header + scrollable content */}
      <div className="flex flex-1 flex-col min-w-0 overflow-hidden">

        {/* Mobile top bar — hidden on md+ */}
        <header className="flex items-center gap-3 h-14 px-4 border-b border-border bg-surface flex-shrink-0 md:hidden">
          <button
            onClick={() => setSidebarOpen(true)}
            className="text-muted hover:text-ink p-1 -ml-1 rounded transition-colors"
            aria-label="Open sidebar"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
              <line x1="3" y1="5.5" x2="17" y2="5.5" />
              <line x1="3" y1="10"  x2="17" y2="10" />
              <line x1="3" y1="14.5" x2="17" y2="14.5" />
            </svg>
          </button>

          <div className="flex items-center gap-2 flex-1 min-w-0">
            <LogoIcon className="w-4 h-4 fill-ink flex-shrink-0" />
            <span className="font-serif font-semibold text-ink text-[13px] tracking-tight truncate">
              {headerLabel ?? 'StrataHQ'}
            </span>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto">
          {children}
        </main>

      </div>
    </div>
  )
}
