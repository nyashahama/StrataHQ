"use client";
import { useState, useEffect } from "react";
import { apiFetch } from "@/lib/api";
import { useToast } from "@/lib/toast";

interface Invitation {
  id: string;
  full_name: string;
  email: string;
  role: "trustee" | "resident";
  scheme_name?: string;
  unit_id?: string | null;
  status: string;
  expires_at: string;
  created_at: string;
}

const ROLE_STYLES: Record<string, string> = {
  trustee: "bg-accent-bg text-accent",
  resident: "bg-green-bg text-green",
};

export default function InvitationsPage() {
  const { addToast } = useToast();
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch("/api/v1/invitations")
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to load");
        const data = await res.json();
        setInvitations(data ?? []);
      })
      .catch(() => addToast("Failed to load invitations", "error" as never))
      .finally(() => setLoading(false));
  }, [addToast]);

  async function handleAction(id: string, action: "resend" | "revoke") {
    if (action === "revoke") {
      const res = await apiFetch(`/api/v1/invitations/${id}`, {
        method: "DELETE",
      });
      if (res.ok) {
        setInvitations((prev) => prev.filter((i) => i.id !== id));
        addToast("Invitation revoked", "info" as never);
      } else {
        addToast("Failed to revoke invitation", "error" as never);
      }
    } else {
      const res = await apiFetch(`/api/v1/invitations/${id}/resend`, {
        method: "POST",
      });
      if (res.ok) {
        addToast("Invitation resent", "success" as never);
      } else {
        addToast("Failed to resend invitation", "error" as never);
      }
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <p className="text-[12px] text-muted mb-4">Portfolio › Invitations</p>
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        Invitations
      </h1>
      <p className="text-[14px] text-muted mb-8">
        Pending trustee and resident invitations.
      </p>

      {loading ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading…
        </div>
      ) : invitations.length === 0 ? (
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          No pending invitations
        </div>
      ) : (
        <div className="bg-surface border border-border rounded-lg">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">Pending</span>
            <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">
              {invitations.length} pending
            </span>
          </div>
          <div className="overflow-x-auto">
            <div className="px-5 min-w-[480px]">
              <div className="grid grid-cols-[1fr_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Invitee</span>
                <span>Role</span>
                <span>Actions</span>
              </div>
              {invitations.map((inv, i) => (
                <div
                  key={inv.id}
                  className={`grid grid-cols-[1fr_auto_auto] gap-4 items-center py-3 text-[13px] ${i < invitations.length - 1 ? "border-b border-border" : ""}`}
                >
                  <div>
                    <div className="font-medium text-ink">{inv.full_name}</div>
                    <div className="text-[12px] text-muted">{inv.email}</div>
                  </div>
                  <span
                    className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[inv.role]}`}
                  >
                    {inv.role.charAt(0).toUpperCase() + inv.role.slice(1)}
                  </span>
                  <div className="flex items-center gap-3">
                    <button
                      onClick={() => handleAction(inv.id, "resend")}
                      className="text-[11px] text-accent font-medium hover:underline"
                    >
                      Resend
                    </button>
                    <button
                      onClick={() => handleAction(inv.id, "revoke")}
                      className="text-[11px] text-red font-medium hover:underline"
                    >
                      Revoke
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
