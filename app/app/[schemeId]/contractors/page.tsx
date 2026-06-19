"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";

import RetryState from "@/components/RetryState";
import { useAuth } from "@/lib/auth";
import { listContractors, searchContractorMarketplace } from "@/lib/contractors-api";
import type { ContractorInfo } from "@/lib/contractors";
import { schemeKeys } from "@/lib/query-keys";
import { useAuthenticatedQuery } from "@/hooks/useAuthenticatedQuery";

const TRADES = ["all", "plumbing", "electrical", "structural", "garden", "pool", "other"] as const;

export default function ContractorsPage() {
  const { user } = useAuth();
  const params = useParams();
  const schemeId = params.schemeId as string;
  const [mode, setMode] = useState<"directory" | "marketplace" | "all">("directory");
  const [trade, setTrade] = useState("all");
  const [query, setQuery] = useState("");

  const isResident = user?.role === "resident";
  const isAdmin = user?.role === "admin";

  useEffect(() => {
    if (mode === "all" && !isAdmin) {
      setMode("directory");
    }
  }, [mode, isAdmin]);

  const { data, isLoading, error, refetch } = useAuthenticatedQuery<ContractorInfo[]>({
    queryKey: [...schemeKeys.contractors(schemeId), mode, trade, query] as const,
    enabled: !isResident,
    queryFn: () => {
      const selectedTrade = trade === "all" ? undefined : trade;
      if (mode === "marketplace") {
        return searchContractorMarketplace({ scheme_id: schemeId, trade: selectedTrade });
      }
      return listContractors({
        scheme_id: mode === "all" ? undefined : schemeId,
        trade: selectedTrade,
        q: query.trim() || undefined,
        active: true,
      });
    },
    staleTime: 30_000,
  });

  const contractors = useMemo(() => data ?? [], [data]);

  if (isResident) {
    return (
      <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Contractor directory is not available for resident accounts.
        </div>
      </div>
    );
  }

  if (error) {
    return <RetryState title="Could not load contractors" message="Temporary service issue. Try again." onRetry={refetch} />;
  }

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[1040px]">
      <p className="text-[12px] text-muted mb-4">Scheme › Contractors</p>
      <div className="flex flex-col md:flex-row md:items-end md:justify-between gap-4 mb-6">
        <div>
          <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">Contractors</h1>
          <p className="text-[14px] text-muted">Browse vetted contractors and manage the scheme directory.</p>
        </div>
      </div>

      <div className="bg-surface border border-border rounded-lg overflow-hidden">
        <div className="px-5 py-4 border-b border-border flex flex-col md:flex-row gap-3 md:items-center md:justify-between">
          <div className="flex gap-2">
            {(["directory", "marketplace", "all"] as const)
              .filter(item => item !== "all" || isAdmin)
              .map(item => (
              <button
                key={item}
                onClick={() => { setMode(item); if (item === "marketplace") setQuery(""); }}
                className={`text-[12px] font-semibold px-3 py-1.5 rounded border ${mode === item ? "bg-accent text-white border-accent" : "border-border text-muted"}`}
              >
                {item === "directory" ? "Scheme directory" : item === "marketplace" ? "Marketplace" : "All"}
              </button>
            ))}
          </div>
          <div className="flex gap-2">
            <input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search contractors" className="border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent" />
            <select value={trade} onChange={e => setTrade(e.target.value)} className="border border-border rounded px-3 py-2 text-[13px] text-ink bg-surface focus:outline-none focus:border-accent">
              {TRADES.map(item => <option key={item} value={item}>{item === "all" ? "All trades" : item}</option>)}
            </select>
          </div>
        </div>

        {isLoading ? (
          <div className="px-5 py-12 text-center text-[13px] text-muted">Loading contractors...</div>
        ) : contractors.length === 0 ? (
          <div className="px-5 py-12 text-center text-[13px] text-muted">No contractors found.</div>
        ) : (
          <div className="divide-y divide-border">
            {contractors.map(contractor => (
              <div key={contractor.id} className="px-5 py-4 flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[13px] font-semibold text-ink">{contractor.name}</span>
                    {contractor.vetted && <span className="text-[10px] font-semibold px-2 py-[2px] rounded-full bg-green-bg text-green">Vetted</span>}
                    {contractor.preferred && <span className="text-[10px] font-semibold px-2 py-[2px] rounded-full bg-accent-bg text-accent">Preferred</span>}
                  </div>
                  <div className="text-[12px] text-muted">{contractor.trade} · {contractor.suburb}, {contractor.city}</div>
                  <div className="text-[11px] text-muted mt-1">{contractor.completed_job_count} completed jobs · {contractor.review_count} reviews</div>
                </div>
                <div className="text-right text-[13px] font-semibold text-ink">
                  {contractor.average_rating > 0 ? contractor.average_rating.toFixed(1) : "—"}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
