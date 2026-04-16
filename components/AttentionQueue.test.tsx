import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AttentionItem } from "@/lib/attention";
import AttentionQueue from "./AttentionQueue";

function makeItem(overrides: Partial<AttentionItem> = {}): AttentionItem {
  return {
    levy_account_id: "acc-1",
    scheme_id: "scheme-1",
    scheme_name: "Rosewood Estate",
    unit_id: "unit-1",
    unit_identifier: "5C",
    owner_name: "Rose Example",
    outstanding_cents: 930000,
    days_overdue: 97,
    risk_score: 88,
    score_drivers: ["90+ days overdue", "high balance outstanding"],
    recommended_action: "legal_review_flagged",
    ...overrides,
  };
}

describe("AttentionQueue", () => {
  it("renders ranked queue items and score drivers", () => {
    render(
      <AttentionQueue
        items={[makeItem()]}
        scope="portfolio"
        loading={false}
        error={null}
        onRefresh={vi.fn()}
      />
    );

    expect(screen.getByText("Rosewood Estate · Unit 5C")).toBeInTheDocument();
    expect(screen.getByText("90+ days overdue")).toBeInTheDocument();
    expect(screen.getByText("Legal review")).toBeInTheDocument();
  });

  it("shows the empty state when no items exist", () => {
    render(
      <AttentionQueue
        items={[]}
        scope="scheme"
        loading={false}
        error={null}
        onRefresh={vi.fn()}
      />
    );

    expect(screen.getByText(/No collection cases need attention/i)).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(
      <AttentionQueue
        items={[]}
        scope="scheme"
        loading={true}
        error={null}
        onRefresh={vi.fn()}
      />
    );

    expect(screen.getByText(/Loading queue/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    render(
      <AttentionQueue
        items={[]}
        scope="scheme"
        loading={false}
        error="Failed to load"
        onRefresh={vi.fn()}
      />
    );

    expect(screen.getByText("Failed to load")).toBeInTheDocument();
  });
});