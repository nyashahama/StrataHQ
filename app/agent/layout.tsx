'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useMockAuth } from '@/lib/mock-auth'
import AppShell from '@/components/AppShell'
import Sidebar from '@/components/Sidebar'

export default function AgentLayout({ children }: { children: React.ReactNode }) {
  const { user } = useMockAuth()
  const router = useRouter()

  useEffect(() => {
    if (user === null) router.replace('/auth/login')
    else if (user.role !== 'agent') router.replace(`/app/${user.schemeId}`)
  }, [user, router])

  if (!user || user.role !== 'agent') return null

  return (
    <AppShell
      sidebar={
        <Sidebar
          role="agent-portfolio"
          headerLabel={user.orgName || 'My Organisation'}
        />
      }
    >
      {children}
    </AppShell>
  )
}
