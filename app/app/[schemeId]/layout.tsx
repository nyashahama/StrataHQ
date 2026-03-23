'use client'
import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'

export default function SchemeLayout({ children }: { children: React.ReactNode }) {
  const { user } = useMockAuth()
  const router = useRouter()
  const params = useParams()
  const schemeId = params.schemeId as string

  useEffect(() => {
    if (!user) { router.replace('/auth/login'); return }
    if (user.schemeId !== schemeId) router.replace(`/app/${user.schemeId}`)
  }, [user, router, schemeId])

  if (!user) return null

  const sidebarRole: SidebarRole =
    user.role === 'agent' ? 'agent-scheme' :
    user.role === 'trustee' ? 'trustee' : 'resident'

  const headerLabel =
    user.role === 'resident'
      ? `Unit ${user.unitIdentifier ?? '?'} · ${user.schemeName}`
      : user.schemeName

  return (
    <AppShell
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
  )
}
