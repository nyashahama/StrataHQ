"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { useAuth } from "@/lib/auth";
import { listSchemes, type SchemeSummary } from "@/lib/scheme-api";
import { useToast } from "@/lib/toast";

const HEALTH_STYLES: Record<SchemeSummary["health"], string> = {
  good: "bg-green-bg text-green",
  fair: "bg-yellowbg text-amber",
  poor: "bg-red-bg text-red",
};

export default function AgentPortfolioPage() {
  useAuth();
  const { addToast } = useToast();
  const [schemes, setSchemes] = useState<SchemeSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        setSchemes(await listSchemes());
      } catch (error) {
        addToast(
          error instanceof Error ? error.message : "Failed to load schemes",
          "error",
        );
      } finally {
        setLoading(false);
      }
    }

    load();
  }, [addToast]);

  const totalUnits = schemes.reduce((sum, scheme) => sum + scheme.unit_count, 0);
  const totalMaintenance = schemes.reduce(
    (sum, scheme) => sum + scheme.open_maintenance_count,
    0,
  );
  const avgCollection =
    schemes.length > 0
      ? Math.round(
          schemes.reduce((sum, scheme) => sum + scheme.levy_collection_pct, 0) /
            schemes.length,
        )
      : 0;

  return (
    <div className="px-4 py-6 sm:px-8 sm:py-8 max-w-[900px]">
      <h1 className="font-serif text-[28px] font-semibold text-ink mb-1">
        Portfolio overview
      </h1>
      <p className="text-[14px] text-muted mb-8">
        {loading
          ? "Loading schemes under management…"
          : `${schemes.length} schemes under management.`}
      </p>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 mb-8">
        {[
          { label: "Active schemes", value: String(schemes.length) },
          { label: "Units managed", value: String(totalUnits) },
          { label: "Open maintenance", value: String(totalMaintenance) },
          { label: "Avg collection rate", value: `${avgCollection}%` },
        ].map(({ label, value }) => (
          <div
            key={label}
            className="bg-surface border border-border rounded-lg px-5 py-4"
          >
            <div className="text-[24px] font-semibold text-ink font-serif mb-1">
              {loading ? "…" : value}
            </div>
            <div className="text-[12px] text-muted">{label}</div>
          </div>
        ))}
      </div>

      {loading ? (
        <div className="bg-surface border border-border rounded-lg px-6 py-12 text-center text-muted text-[14px]">
          Loading portfolio…
        </div>
      ) : schemes.length === 0 ? (
        <div className="bg-hover-subtle border border-border rounded-lg px-6 py-12 text-center">
          <p className="text-[15px] text-ink font-medium mb-2">
            No schemes found
          </p>
          <p className="text-[13px] text-muted">
            Complete onboarding or create a scheme to start managing your
            portfolio.
          </p>
        </div>
      ) : (
        <div className="bg-surface border border-border rounded-lg">
          <div className="px-5 py-3 border-b border-border">
            <span className="text-[13px] font-semibold text-ink">
              All schemes
            </span>
          </div>
          <div className="overflow-x-auto">
            <div className="px-5 min-w-[620px]">
              <div className="grid grid-cols-[1fr_44px_80px_100px_68px_60px] gap-4 py-2 text-[11px] font-semibold text-muted uppercase tracking-wide border-b border-border whitespace-nowrap">
                <span>Scheme</span>
                <span>Units</span>
                <span>Collection</span>
                <span>Maintenance</span>
                <span>Health</span>
                <span></span>
              </div>
              {schemes.map((scheme, index) => (
                <div
                  key={scheme.id}
                  className={`grid grid-cols-[1fr_44px_80px_100px_68px_60px] gap-4 items-center py-3 text-[13px] ${
                    index < schemes.length - 1 ? "border-b border-border" : ""
                  }`}
                >
                  <div>
                    <div className="font-semibold text-ink">{scheme.name}</div>
                    <div className="text-[12px] text-muted">{scheme.address}</div>
                  </div>
                  <span className="text-muted">{scheme.unit_count}</span>
                  <span className="font-medium text-ink">
                    {scheme.levy_collection_pct}%
                  </span>
                  <span className="text-muted">
                    {scheme.open_maintenance_count} open
                  </span>
                  <span
                    className={`text-[11px] font-semibold px-2 py-[2px] rounded-full ${HEALTH_STYLES[scheme.health]}`}
                  >
                    {scheme.health.charAt(0).toUpperCase() +
                      scheme.health.slice(1)}
                  </span>
                  <Link
                    href={`/app/${scheme.id}`}
                    className="text-[12px] text-accent font-medium"
                  >
                    View →
                  </Link>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
