import AppShell from "@/components/AppShell";
import Sidebar from "@/components/Sidebar";
import { ToastProvider } from "@/lib/toast";
import Copilot from "@/components/Copilot";
import { requireAdminSession } from "@/lib/server-session";

export default async function AgentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  await requireAdminSession();

  return (
    <ToastProvider>
      <AppShell
        headerLabel="My Organisation"
        sidebar={
          <Sidebar role="agent-portfolio" headerLabel="My Organisation" />
        }
      >
        {children}
      </AppShell>
      <Copilot />
    </ToastProvider>
  );
}
