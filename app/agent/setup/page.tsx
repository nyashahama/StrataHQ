"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import SetupWizard from "@/components/wizard/SetupWizard";

export default function SetupPage() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (!user) router.replace("/auth/login");
    else if (user.wizard_complete) router.replace("/agent");
  }, [user, loading, router]);

  if (loading || !user || user.wizard_complete) return null;

  return <SetupWizard />;
}
