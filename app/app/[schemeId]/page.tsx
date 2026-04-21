import { SchemeOverview } from "@/components/scheme/SchemeOverview";
import type { AttentionItem } from "@/lib/attention";
import { fetchBackendJson } from "@/lib/server-api";
import { requireSchemeSession } from "@/lib/server-session";
import type { SchemeDetail } from "@/lib/scheme-api";

export default async function SchemeOverviewPage({
  params,
}: {
  params: Promise<{ schemeId: string }>;
}) {
  const { schemeId } = await params;
  const user = await requireSchemeSession(schemeId);
  const isResident = user.role === "resident";

  const scheme = await fetchBackendJson<SchemeDetail>(`/api/v1/schemes/${schemeId}`);
  const attentionItems = isResident
    ? []
    : (
        await fetchBackendJson<{ items: AttentionItem[] }>(
          `/api/v1/levies/${schemeId}/attention`,
        )
      ).items;

  return (
    <SchemeOverview
      scheme={scheme}
      attentionItems={attentionItems}
      isResident={isResident}
    />
  );
}
