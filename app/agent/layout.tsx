"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import AppShell from "@/components/AppShell";
import Sidebar from "@/components/Sidebar";
import { ToastProvider } from "@/lib/toast";
import Copilot from "@/components/Copilot";
import { isAdminRole, primarySchemeId } from "@/lib/session";

export default function AgentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (user === null) {
      router.replace("/auth/login");
    } else if (!isAdminRole(user.role)) {
      router.replace(`/app/${primarySchemeId(user) ?? ""}`);
    }
  }, [user, loading, router]);

  if (loading || !user || !isAdminRole(user.role)) return null;

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
