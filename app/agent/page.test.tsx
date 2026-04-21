import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PortfolioOverview } from "@/components/agent/PortfolioOverview";

describe("PortfolioOverview", () => {
  it("renders the empty state when no schemes are returned", () => {
    render(<PortfolioOverview schemes={[]} attentionItems={[]} />);

    expect(screen.getByText("No schemes found")).toBeInTheDocument();
    expect(screen.getByText("0 schemes under management.")).toBeInTheDocument();
  });

  it("renders scheme rows and queue labels for populated data", () => {
    render(
      <PortfolioOverview
        schemes={[
          {
            id: "scheme-1",
            name: "Scheme 1",
            address: "Address 1",
            role: "admin",
            health: "good",
            unit_count: 12,
            total_members: 14,
            trustee_count: 2,
            resident_count: 12,
            levy_collection_pct: 92,
            open_maintenance_count: 3,
            notice_count: 1,
          },
        ]}
        attentionItems={[
          {
            scheme_id: "scheme-1",
            scheme_name: "Scheme 1",
            unit_id: "unit-1",
            unit_identifier: "1A",
            levy_account_id: "acct-1",
            owner_name: "Owner One",
            outstanding_cents: 50000,
            days_overdue: 21,
            risk_score: 78,
            score_drivers: ["overdue > 14d"],
            recommended_action: "follow_up_logged",
          },
        ]}
      />,
    );

    expect(screen.getByText("Scheme 1")).toBeInTheDocument();
    expect(screen.getByText("Collections attention queue")).toBeInTheDocument();
    expect(screen.getByText("overdue > 14d")).toBeInTheDocument();
  });
});
