import type { CollectionEvent } from "@/lib/attention";

export function CollectionActivityLog({ events }: { events: CollectionEvent[] }) {
  if (events.length === 0) {
    return <p className="text-xs text-muted">No collection activity yet.</p>;
  }

  return (
    <div className="space-y-2">
      {events.map((event) => (
        <div key={event.id} className="rounded-lg border border-border bg-page px-3 py-2">
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs font-semibold text-ink">{event.event_type}</span>
            <span className="text-[11px] text-muted">{new Date(event.created_at).toLocaleDateString("en-ZA")}</span>
          </div>
          {event.email_status ? <p className="mt-1 text-[11px] text-muted">Email: {event.email_status}</p> : null}
          {event.whatsapp_status ? <p className="text-[11px] text-muted">WhatsApp: {event.whatsapp_status}</p> : null}
          {event.note ? <p className="mt-1 text-[12px] text-ink">{event.note}</p> : null}
        </div>
      ))}
    </div>
  );
}