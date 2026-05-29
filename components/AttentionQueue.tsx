"use client";

import { useState } from "react";
import type { AttentionItem } from "@/lib/attention";
import { CollectionActionPanel } from "./CollectionActionPanel";

export default function AttentionQueue({
  items,
  scope,
  loading,
  error,
  onRefresh,
}: {
  items: AttentionItem[];
  scope: "portfolio" | "scheme";
  loading: boolean;
  error: string | null;
  onRefresh?: () => void;
}) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const handleRefresh = onRefresh ?? (() => {});

  if (loading)
    return (
      <div className="rounded-lg border border-border bg-surface px-6 py-10 text-sm text-muted">
        Loading queue…
      </div>
    );
  if (error)
    return (
      <div className="rounded-lg border border-red bg-red-bg px-6 py-4 text-sm text-red">
        {error}
      </div>
    );
  if (items.length === 0)
    return (
      <div className="rounded-lg border border-border bg-surface px-6 py-10 text-sm text-muted">
        No collection cases need attention.
      </div>
    );

  return (
    <div className="rounded-lg border border-border bg-surface overflow-hidden">
      {items.map((item) => {
        const expanded = expandedId === item.levy_account_id;
        return (
          <div key={item.levy_account_id} className="border-b border-border last:border-b-0">
            <button
              className="w-full px-5 py-4 text-left"
              onClick={() =>
                setExpandedId(expanded ? null : item.levy_account_id)
              }
            >
              <div className="flex items-center justify-between gap-4">
                <div>
                  <div className="text-sm font-semibold text-ink">
                    {scope === "portfolio"
                      ? `${item.scheme_name} · Unit ${item.unit_identifier}`
                      : `Unit ${item.unit_identifier}`}
                  </div>
                  <div className="mt-1 flex flex-wrap gap-2">
                    {item.score_drivers.map((driver) => (
                      <span
                        key={driver}
                        className="rounded-full bg-page px-2 py-1 text-[11px] text-muted"
                      >
                        {driver}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-sm font-semibold text-ink">
                    R {(item.outstanding_cents / 100).toLocaleString("en-ZA")}
                  </div>
                  <div className="text-xs text-muted">
                    {actionLabel(item.recommended_action)}
                  </div>
                </div>
              </div>
            </button>
            {expanded ? (
              <div className="border-t border-border px-5 py-4">
                <CollectionActionPanel item={item} onRefresh={handleRefresh} />
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function actionLabel(
  value: AttentionItem["recommended_action"]
) {
  switch (value) {
    case "legal_review_flagged":
      return "Legal review";
    case "promise_to_pay":
      return "Promise to pay";
    case "active_promise":
      return "Awaiting promise";
    case "follow_up_logged":
      return "Follow-up";
    default:
      return "Send reminder";
  }
}
