'use client'
import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'

export default function SchemeLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useMockAuth()
  const router = useRouter()
  const params = useParams()
  const schemeId = params.schemeId as string

  useEffect(() => {
    if (loading) return
    if (!user) { router.replace('/auth/login'); return }
    if (user.schemeId !== schemeId) router.replace(`/app/${user.schemeId}`)
  }, [user, loading, router, schemeId])

  if (loading || !user) return null

  const sidebarRole: SidebarRole =
    user.role === 'agent' ? 'agent-scheme' :
    user.role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel =
    user.role === 'resident'
      ? `Unit ${user.unitIdentifier ?? '?'} · ${user.schemeName}`
      : user.schemeName

  return (
    <ToastProvider>
      <AppShell
        headerLabel={headerLabel}
        sidebar={
          <Sidebar
            role={sidebarRole}
            headerLabel={headerLabel}
            schemeId={schemeId}
          />
        }
      >
        {children}
      </AppShell>
    </ToastProvider>
  )
}
