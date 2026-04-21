import AppShell from '@/components/AppShell'
import Sidebar, { type SidebarRole } from '@/components/Sidebar'
import { ToastProvider } from '@/lib/toast'
import Copilot from '@/components/Copilot'
import { isAdminRole, isResidentRole } from '@/lib/session'
import { requireSchemeSession } from '@/lib/server-session'

export default async function SchemeLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ schemeId: string }>
}) {
  const { schemeId } = await params
  const user = await requireSchemeSession(schemeId)

  const currentScheme = isAdminRole(user.role)
    ? user.scheme_memberships.find(m => m.scheme_id === schemeId) ?? user.scheme_memberships[0]
    : user.scheme_memberships.find(m => m.scheme_id === schemeId)

  const sidebarRole: SidebarRole =
    isAdminRole(user.role) ? 'agent-scheme' :
    isResidentRole(user.role) ? 'resident' : 'trustee'

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
            allMemberships={user.scheme_memberships}
          />
        }
      >
        {children}
      </AppShell>
      <Copilot />
    </ToastProvider>
  )
}
