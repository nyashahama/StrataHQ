import { PortfolioOverview } from "@/components/agent/PortfolioOverview";
import type { AttentionItem } from "@/lib/attention";
import { fetchBackendJson } from "@/lib/server-api";
import type { SchemeSummary } from "@/lib/scheme-api";
import { requireAdminSession } from "@/lib/server-session";

export default async function AgentPortfolioPage() {
  await requireAdminSession();

  const [schemes, attentionQueue] = await Promise.all([
    fetchBackendJson<SchemeSummary[]>("/api/v1/schemes"),
    fetchBackendJson<{ items: AttentionItem[] }>("/api/v1/levies/attention"),
  ]);

  return (
    <PortfolioOverview
      schemes={schemes}
      attentionItems={attentionQueue.items}
    />
  );
}
