interface AppShellProps {
  sidebar: React.ReactNode
  children: React.ReactNode
}

export default function AppShell({ sidebar, children }: AppShellProps) {
  return (
    <div className="flex h-screen overflow-hidden bg-page">
      {sidebar}
      <main className="flex-1 overflow-y-auto">
        {children}
      </main>
    </div>
  )
}
