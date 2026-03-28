'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import AppShell from '@/components/AppShell'
import Sidebar from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'

export default function AgentLayout({ children }: { children: React.ReactNode }) {
  const { user, loading } = useMockAuth()
  const router = useRouter()

  useEffect(() => {
    if (loading) return
    if (user === null) router.replace('/auth/login')
    else if (user.role !== 'agent') router.replace(`/app/${user.schemeId}`)
  }, [user, loading, router])

  if (loading || !user || user.role !== 'agent') return null

  return (
    <ToastProvider>
      <AppShell
        headerLabel={user.orgName || 'My Organisation'}
        sidebar={
          <Sidebar
            role="agent-portfolio"
            headerLabel={user.orgName || 'My Organisation'}
          />
        }
      >
        {children}
      </AppShell>
      <Copilot />
    </ToastProvider>
  )
}
