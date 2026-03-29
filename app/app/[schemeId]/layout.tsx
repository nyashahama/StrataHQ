'use client'
import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'

export default function SchemeLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  const router = useRouter()
  const params = useParams()
  const schemeId = params.schemeId as string

  useEffect(() => {
    if (loading) return
    if (!user) { router.replace('/auth/login'); return }

    // Admins can access any scheme; trustee/resident must be a member of this specific scheme
    if (user.role !== 'admin') {
      const isMember = user.scheme_memberships.some(m => m.scheme_id === schemeId)
      if (!isMember) {
        router.replace(`/app/${user.scheme_memberships[0]?.scheme_id ?? ''}`)
      }
    }
  }, [user, loading, router, schemeId])

  if (loading || !user) return null

  const currentScheme = user.role === 'admin'
    ? user.scheme_memberships.find(m => m.scheme_id === schemeId) ?? user.scheme_memberships[0]
    : user.scheme_memberships.find(m => m.scheme_id === schemeId)

  const sidebarRole: SidebarRole =
    user.role === 'admin' ? 'agent-scheme' :
    user.role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel = currentScheme?.scheme_name ?? schemeId

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
      <Copilot />
    </ToastProvider>
  )
}
