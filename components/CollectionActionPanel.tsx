"use client";

import { useEffect, useState } from "react";
import type { AttentionItem, CollectionEvent } from "@/lib/attention";
import { createCollectionEvent, listCollectionEvents } from "@/lib/attention-api";
import { useToast } from "@/lib/toast";
import CollectionExecutionModal from "./CollectionExecutionModal";
import { CollectionActivityLog } from "./CollectionActivityLog";

export function CollectionActionPanel({
  item,
  onRefresh,
}: {
  item: AttentionItem;
  onRefresh: () => void;
}) {
  const { addToast } = useToast();
  const [events, setEvents] = useState<CollectionEvent[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [note, setNote] = useState("");
  const [promiseAmount, setPromiseAmount] = useState("");
  const [promiseDate, setPromiseDate] = useState("");
  const [saving, setSaving] = useState(false);

  async function loadEvents() {
    try {
      setEvents(await listCollectionEvents(item.scheme_id, item.levy_account_id));
    } catch {
      setEvents([]);
    }
  }

  useEffect(() => {
    loadEvents();
  }, [item.scheme_id, item.levy_account_id]);

  async function submitEvent(input: {
    event_type: "follow_up_logged" | "promise_to_pay" | "legal_review_flagged";
    note?: string;
    promise_amount_cents?: number;
    promise_date?: string;
  }) {
    try {
      setSaving(true);
      await createCollectionEvent(item.scheme_id, item.levy_account_id, input);
      setNote("");
      setPromiseAmount("");
      setPromiseDate("");
      await loadEvents();
      onRefresh();
      addToast("Collection action saved", "success");
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Failed to save collection action", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        <button className="rounded-lg bg-ink px-3 py-2 text-xs font-semibold text-white" onClick={() => setModalOpen(true)}>
          Send reminder
        </button>
        <button className="rounded-lg border border-border px-3 py-2 text-xs text-ink" onClick={() => submitEvent({ event_type: "legal_review_flagged", note: "Flagged from attention queue" })}>
          Flag legal review
        </button>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="rounded-lg border border-border px-4 py-3">
          <p className="mb-2 text-xs font-semibold text-ink">Log follow-up</p>
          <textarea
            value={note}
            onChange={(event) => setNote(event.target.value)}
            rows={3}
            className="w-full rounded-lg border border-border px-3 py-2 text-sm"
          />
          <button
            onClick={() => submitEvent({ event_type: "follow_up_logged", note })}
            disabled={saving || !note.trim()}
            className="mt-2 rounded-lg border border-border px-3 py-2 text-xs text-ink disabled:opacity-40"
          >
            Save follow-up
          </button>
        </div>

        <div className="rounded-lg border border-border px-4 py-3">
          <p className="mb-2 text-xs font-semibold text-ink">Promise to pay</p>
          <input
            placeholder="Amount in cents"
            value={promiseAmount}
            onChange={(event) => setPromiseAmount(event.target.value)}
            className="mb-2 w-full rounded-lg border border-border px-3 py-2 text-sm"
          />
          <input
            type="date"
            value={promiseDate}
            onChange={(event) => setPromiseDate(event.target.value)}
            className="w-full rounded-lg border border-border px-3 py-2 text-sm"
          />
          <button
            onClick={() =>
              submitEvent({
                event_type: "promise_to_pay",
                promise_amount_cents: Number(promiseAmount),
                promise_date: promiseDate,
              })
            }
            disabled={saving || !promiseAmount || !promiseDate}
            className="mt-2 rounded-lg border border-border px-3 py-2 text-xs text-ink disabled:opacity-40"
          >
            Save promise
          </button>
        </div>
      </div>
      <CollectionActivityLog events={events} />
      <CollectionExecutionModal
        open={modalOpen}
        schemeId={item.scheme_id}
        accountId={item.levy_account_id}
        onClose={() => setModalOpen(false)}
        onSent={async () => {
          await loadEvents();
          onRefresh();
        }}
      />
    </div>
  );
}