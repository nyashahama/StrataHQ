"use client";

import { useEffect, useState } from "react";
import Modal from "@/components/Modal";
import { useToast } from "@/lib/toast";
import { getCollectionReminderDraft, sendCollectionReminder } from "@/lib/attention-api";
import type { ReminderDraft } from "@/lib/attention";

export default function CollectionExecutionModal({
  open,
  schemeId,
  accountId,
  onClose,
  onSent,
}: {
  open: boolean;
  schemeId: string;
  accountId: string;
  onClose: () => void;
  onSent: () => void;
}) {
  const { addToast } = useToast();
  const [draft, setDraft] = useState<ReminderDraft | null>(null);
  const [sending, setSending] = useState(false);

  useEffect(() => {
    if (!open) return;
    getCollectionReminderDraft(schemeId, accountId)
      .then(setDraft)
      .catch((error) => addToast(error instanceof Error ? error.message : "Failed to load reminder draft", "error"));
  }, [open, schemeId, accountId, addToast]);

  async function handleSend() {
    if (!draft) return;
    setSending(true);
    try {
      await sendCollectionReminder(schemeId, accountId, {
        email: {
          enabled: draft.email.enabled,
          subject: draft.email.subject,
          body: draft.email.body,
        },
        whatsapp: {
          enabled: draft.whatsapp.enabled,
          body: draft.whatsapp.body,
        },
      });
      addToast("Reminder processed", "success");
      onSent();
      onClose();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Failed to send reminder", "error");
    } finally {
      setSending(false);
    }
  }

  return (
    <Modal open={open} onClose={() => !sending && onClose()} title="Send reminder">
      {!draft ? (
        <div className="py-8 text-center text-sm text-muted">Loading reminder draft…</div>
      ) : (
        <div className="space-y-4">
          <div className="rounded-lg border border-border bg-page px-4 py-3">
            <p className="text-xs text-muted">{draft.scheme_name}</p>
            <p className="text-sm font-semibold text-ink">{draft.unit_label}</p>
            <p className="text-xs text-muted">{draft.owner_name}</p>
          </div>

          <div className="space-y-2 rounded-lg border border-border px-4 py-3">
            <label className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold text-ink">Email</span>
              <input
                type="checkbox"
                checked={draft.email.enabled}
                disabled={!!draft.email.disabled_reason}
                onChange={(event) =>
                  setDraft((current) =>
                    current
                      ? { ...current, email: { ...current.email, enabled: event.target.checked } }
                      : current,
                  )
                }
              />
            </label>
            {draft.email.disabled_reason ? <p className="text-xs text-muted">{draft.email.disabled_reason}</p> : null}
            <input
              aria-label="Email subject"
              value={draft.email.subject ?? ""}
              onChange={(event) =>
                setDraft((current) =>
                  current
                    ? { ...current, email: { ...current.email, subject: event.target.value } }
                    : current,
                )
              }
              className="w-full rounded-lg border border-border px-3 py-2 text-sm"
            />
            <textarea
              aria-label="Email body"
              value={draft.email.body}
              onChange={(event) =>
                setDraft((current) =>
                  current
                    ? { ...current, email: { ...current.email, body: event.target.value } }
                    : current,
                )
              }
              rows={6}
              className="w-full rounded-lg border border-border px-3 py-2 text-sm"
            />
          </div>

          <div className="space-y-2 rounded-lg border border-border px-4 py-3">
            <label className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold text-ink">WhatsApp</span>
              <input
                type="checkbox"
                checked={draft.whatsapp.enabled}
                disabled={!!draft.whatsapp.disabled_reason}
                onChange={(event) =>
                  setDraft((current) =>
                    current
                      ? { ...current, whatsapp: { ...current.whatsapp, enabled: event.target.checked } }
                      : current,
                  )
                }
              />
            </label>
            {draft.whatsapp.disabled_reason ? <p className="text-xs text-muted">{draft.whatsapp.disabled_reason}</p> : null}
            <textarea
              aria-label="WhatsApp body"
              value={draft.whatsapp.body}
              onChange={(event) =>
                setDraft((current) =>
                  current
                    ? { ...current, whatsapp: { ...current.whatsapp, body: event.target.value } }
                    : current,
                )
              }
              rows={4}
              className="w-full rounded-lg border border-border px-3 py-2 text-sm"
            />
          </div>

          <button
            onClick={handleSend}
            disabled={sending || (!draft.email.enabled && !draft.whatsapp.enabled)}
            className="w-full rounded-lg bg-ink px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
          >
            {sending ? "Sending…" : "Send reminder"}
          </button>
        </div>
      )}
    </Modal>
  );
}