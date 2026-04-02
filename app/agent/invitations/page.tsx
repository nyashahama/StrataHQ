"use client";

import { useEffect, useState } from "react";

import Modal from "@/components/Modal";
import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import { useAuth } from "@/lib/auth";
import { useToast } from "@/lib/toast";

interface Invitation {
  id: string;
  full_name: string;
  email: string;
  role: "trustee" | "resident";
  scheme_id: string;
  scheme_name?: string;
  unit_id?: string | null;
  status: string;
  expires_at: string;
}

interface InviteForm {
  full_name: string;
  email: string;
  role: "trustee" | "resident";
  scheme_id: string;
  unit_id: string;
}

const ROLE_STYLES: Record<Invitation["role"], string> = {
  trustee: "bg-accent-bg text-accent",
  resident: "bg-green-bg text-green",
};

const INITIAL_FORM: InviteForm = {
  full_name: "",
  email: "",
  role: "trustee",
  scheme_id: "",
  unit_id: "",
};

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString("en-ZA", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function daysUntil(value: string): number {
  const diff = new Date(value).getTime() - Date.now();
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
}

export default function InvitationsPage() {
  const { addToast } = useToast();
  const { user } = useAuth();
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [activeAction, setActiveAction] = useState<{
    id: string;
    type: "resend" | "revoke";
  } | null>(null);
  const [showCompose, setShowCompose] = useState(false);
  const [formError, setFormError] = useState("");
  const [form, setForm] = useState<InviteForm>(INITIAL_FORM);

  const schemes = user?.scheme_memberships ?? [];
  const schemeNameById = Object.fromEntries(
    schemes.map((scheme) => [scheme.scheme_id, scheme.scheme_name]),
  );

  const trusteeCount = invitations.filter((inv) => inv.role === "trustee").length;
  const residentCount = invitations.length - trusteeCount;
  const nextExpiry = [...invitations].sort(
    (a, b) =>
      new Date(a.expires_at).getTime() - new Date(b.expires_at).getTime(),
  )[0];

  function enrichInvitation(invitation: Invitation): Invitation {
    return {
      ...invitation,
      scheme_name:
        invitation.scheme_name ??
        schemeNameById[invitation.scheme_id] ??
        invitation.scheme_id,
    };
  }

  async function loadInvitations() {
    setLoading(true);
    try {
      const res = await apiFetch("/api/v1/invitations");
      if (!res.ok) {
        throw new Error(await readApiError(res, "Failed to load invitations"));
      }
      const data = await readApiData<Invitation[]>(res);
      setInvitations(data.map(enrichInvitation));
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : "Failed to load invitations",
        "error" as never,
      );
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadInvitations();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (showCompose && schemes.length > 0 && !form.scheme_id) {
      setForm((current) => ({ ...current, scheme_id: schemes[0].scheme_id }));
    }
  }, [showCompose, schemes, form.scheme_id]);

  async function handleAction(id: string, type: "resend" | "revoke") {
    setActiveAction({ id, type });

    try {
      if (type === "revoke") {
        const res = await apiFetch(`/api/v1/invitations/${id}`, {
          method: "DELETE",
        });
        if (!res.ok) {
          throw new Error(
            await readApiError(res, "Failed to revoke invitation"),
          );
        }
        setInvitations((prev) => prev.filter((inv) => inv.id !== id));
        addToast("Invitation revoked", "info" as never);
        return;
      }

      const res = await apiFetch(`/api/v1/invitations/${id}/resend`, {
        method: "POST",
      });
      if (!res.ok) {
        throw new Error(
          await readApiError(res, "Failed to resend invitation"),
        );
      }
      const updated = enrichInvitation(await readApiData<Invitation>(res));
      setInvitations((prev) =>
        prev.map((inv) => (inv.id === id ? updated : inv)),
      );
      addToast("Invitation resent", "success" as never);
    } catch (error) {
      addToast(
        error instanceof Error ? error.message : `Failed to ${type} invitation`,
        "error" as never,
      );
    } finally {
      setActiveAction(null);
    }
  }

  async function handleCreateInvitation() {
    setFormError("");

    if (
      !form.full_name.trim() ||
      !form.email.trim() ||
      !form.role ||
      !form.scheme_id
    ) {
      setFormError("Full name, email, role, and scheme are required.");
      return;
    }

    if (form.role === "resident" && !form.unit_id.trim()) {
      setFormError("Resident invitations require a unit ID.");
      return;
    }

    setSubmitting(true);

    try {
      const res = await apiFetch("/api/v1/invitations", {
        method: "POST",
        body: JSON.stringify({
          full_name: form.full_name.trim(),
          email: form.email.trim(),
          role: form.role,
          scheme_id: form.scheme_id,
          unit_id: form.role === "resident" ? form.unit_id.trim() : "",
        }),
      });

      if (!res.ok) {
        throw new Error(
          await readApiError(res, "Failed to create invitation"),
        );
      }

      const created = enrichInvitation(await readApiData<Invitation>(res));
      setInvitations((prev) => [created, ...prev]);
      setShowCompose(false);
      setForm({
        ...INITIAL_FORM,
        scheme_id: form.scheme_id,
      });
      addToast("Invitation sent", "success" as never);
    } catch (error) {
      setFormError(
        error instanceof Error ? error.message : "Failed to create invitation",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[980px]">
      <div className="flex flex-wrap items-start justify-between gap-4 mb-8">
        <div>
          <p className="text-[12px] text-muted mb-2">Portfolio › Invitations</p>
          <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
            Invitations
          </h1>
          <p className="text-[14px] text-muted max-w-[620px]">
            Invite trustees and residents into the right scheme, then track
            pending access from one place.
          </p>
        </div>

        <button
          onClick={() => {
            setFormError("");
            setForm((current) => ({
              ...INITIAL_FORM,
              scheme_id: current.scheme_id || schemes[0]?.scheme_id || "",
            }));
            setShowCompose(true);
          }}
          disabled={schemes.length === 0}
          className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          + Send invitation
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        {[
          { label: "Pending invites", value: String(invitations.length) },
          { label: "Trustees", value: String(trusteeCount) },
          {
            label: "Next expiry",
            value: nextExpiry
              ? `${Math.max(daysUntil(nextExpiry.expires_at), 0)}d`
              : "—",
            detail: nextExpiry ? formatDate(nextExpiry.expires_at) : "No invites",
          },
        ].map((item) => (
          <div
            key={item.label}
            className="bg-surface border border-border rounded-lg px-5 py-4"
          >
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">
              {item.value}
            </div>
            <div className="text-[12px] text-muted">{item.label}</div>
            {"detail" in item && item.detail ? (
              <div className="text-[11px] text-muted mt-1">{item.detail}</div>
            ) : null}
          </div>
        ))}
      </div>

      <div className="bg-page border border-border rounded-lg px-4 py-3 text-[12px] text-muted mb-6">
        Resident invitations currently require the target unit ID from the
        backend data model. Trustee invitations only need a scheme.
      </div>

      {loading ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading invitations…
        </div>
      ) : invitations.length === 0 ? (
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center">
          <p className="text-[15px] text-ink font-medium mb-2">
            No pending invitations
          </p>
          <p className="text-[13px] text-muted mb-5">
            Send your first invite to bring trustees or residents into a scheme.
          </p>
          <button
            onClick={() => setShowCompose(true)}
            disabled={schemes.length === 0}
            className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Send invitation
          </button>
        </div>
      ) : (
        <div className="bg-surface border border-border rounded-lg overflow-hidden">
          <div className="px-5 py-3 border-b border-border flex items-center justify-between">
            <span className="text-[13px] font-semibold text-ink">
              Pending invitations
            </span>
            <span className="text-[11px] font-semibold px-2 py-[2px] rounded-full bg-yellowbg text-amber">
              {invitations.length} pending
            </span>
          </div>

          <div className="overflow-x-auto">
            <div className="px-5 min-w-[760px]">
              <div className="grid grid-cols-[1.3fr_1fr_auto_auto_auto] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border">
                <span>Invitee</span>
                <span>Scheme</span>
                <span>Role</span>
                <span>Expires</span>
                <span>Actions</span>
              </div>

              {invitations.map((inv, index) => {
                const pendingResend =
                  activeAction?.id === inv.id && activeAction.type === "resend";
                const pendingRevoke =
                  activeAction?.id === inv.id && activeAction.type === "revoke";

                return (
                  <div
                    key={inv.id}
                    className={`grid grid-cols-[1.3fr_1fr_auto_auto_auto] gap-4 items-center py-4 text-[13px] ${
                      index < invitations.length - 1
                        ? "border-b border-border"
                        : ""
                    }`}
                  >
                    <div className="min-w-0">
                      <div className="font-medium text-ink">{inv.full_name}</div>
                      <div className="text-[12px] text-muted truncate">
                        {inv.email}
                      </div>
                      {inv.unit_id ? (
                        <div className="text-[11px] text-muted mt-1">
                          Unit ID: {inv.unit_id}
                        </div>
                      ) : null}
                    </div>

                    <div className="min-w-0">
                      <div className="text-ink font-medium truncate">
                        {inv.scheme_name ?? inv.scheme_id}
                      </div>
                      <div className="text-[11px] text-muted truncate">
                        {inv.scheme_id}
                      </div>
                    </div>

                    <span
                      className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${ROLE_STYLES[inv.role]}`}
                    >
                      {inv.role.charAt(0).toUpperCase() + inv.role.slice(1)}
                    </span>

                    <div className="text-[12px] text-muted">
                      <div>{formatDate(inv.expires_at)}</div>
                      <div className="text-[11px] mt-1">
                        {Math.max(daysUntil(inv.expires_at), 0)} days left
                      </div>
                    </div>

                    <div className="flex items-center gap-3 justify-end">
                      <button
                        onClick={() => handleAction(inv.id, "resend")}
                        disabled={pendingResend || pendingRevoke}
                        className="text-[11px] text-accent font-medium hover:underline disabled:opacity-40 disabled:no-underline"
                      >
                        {pendingResend ? "Sending…" : "Resend"}
                      </button>
                      <button
                        onClick={() => handleAction(inv.id, "revoke")}
                        disabled={pendingResend || pendingRevoke}
                        className="text-[11px] text-red font-medium hover:underline disabled:opacity-40 disabled:no-underline"
                      >
                        {pendingRevoke ? "Revoking…" : "Revoke"}
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      <Modal
        open={showCompose}
        onClose={() => {
          if (submitting) return;
          setShowCompose(false);
          setFormError("");
        }}
        title="Send invitation"
      >
        <div className="flex flex-col gap-4">
          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">
              Full name *
            </label>
            <input
              type="text"
              value={form.full_name}
              onChange={(e) =>
                setForm((current) => ({
                  ...current,
                  full_name: e.target.value,
                }))
              }
              placeholder="e.g. Jane Trustee"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>

          <div>
            <label className="text-[12px] font-semibold text-ink block mb-1">
              Email *
            </label>
            <input
              type="email"
              value={form.email}
              onChange={(e) =>
                setForm((current) => ({
                  ...current,
                  email: e.target.value,
                }))
              }
              placeholder="jane@example.com"
              className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">
                Role *
              </label>
              <select
                value={form.role}
                onChange={(e) =>
                  setForm((current) => ({
                    ...current,
                    role: e.target.value as InviteForm["role"],
                    unit_id:
                      e.target.value === "trustee" ? "" : current.unit_id,
                  }))
                }
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                <option value="trustee">Trustee</option>
                <option value="resident">Resident</option>
              </select>
            </div>

            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">
                Scheme *
              </label>
              <select
                value={form.scheme_id}
                onChange={(e) =>
                  setForm((current) => ({
                    ...current,
                    scheme_id: e.target.value,
                  }))
                }
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              >
                {schemes.length === 0 ? (
                  <option value="">No schemes available</option>
                ) : (
                  schemes.map((scheme) => (
                    <option key={scheme.scheme_id} value={scheme.scheme_id}>
                      {scheme.scheme_name}
                    </option>
                  ))
                )}
              </select>
            </div>
          </div>

          {form.role === "resident" ? (
            <div>
              <label className="text-[12px] font-semibold text-ink block mb-1">
                Unit ID *
              </label>
              <input
                type="text"
                value={form.unit_id}
                onChange={(e) =>
                  setForm((current) => ({
                    ...current,
                    unit_id: e.target.value,
                  }))
                }
                placeholder="Paste the resident unit UUID"
                className="w-full border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent"
              />
              <p className="text-[11px] text-muted mt-1">
                The current backend contract requires the unit UUID for resident
                invitations.
              </p>
            </div>
          ) : null}

          {formError ? (
            <div className="text-[12px] text-red bg-red-bg border border-red/20 rounded px-3 py-2">
              {formError}
            </div>
          ) : null}

          <div className="flex justify-end gap-3 pt-1">
            <button
              onClick={() => {
                setShowCompose(false);
                setFormError("");
              }}
              disabled={submitting}
              className="text-[12px] text-muted hover:text-ink px-3 py-2 disabled:opacity-40"
            >
              Cancel
            </button>
            <button
              onClick={handleCreateInvitation}
              disabled={submitting || schemes.length === 0}
              className="text-[12px] font-semibold bg-accent text-white px-4 py-2 rounded hover:opacity-90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {submitting ? "Sending…" : "Send invitation"}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
