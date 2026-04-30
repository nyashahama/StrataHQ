import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockUseAuth = vi.hoisted(() => vi.fn());
const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({ useAuth: mockUseAuth }));
vi.mock("next/navigation", () => ({ useParams: vi.fn(() => ({ schemeId: "scheme-1" })) }));
vi.mock("@/hooks/useAuthenticatedQuery", () => ({ useAuthenticatedQuery: mockUseAuthenticatedQuery }));
vi.mock("@/lib/toast", () => ({ useToast: vi.fn(() => ({ addToast: vi.fn() })) }));

const dashboard = {
  reserve_fund: { scheme_id: "scheme-1", balance_cents: 1000000, target_cents: 2500000, last_updated: new Date().toISOString() },
  levy_summary: { period_label: "March 2026", total_billed_cents: 2500000, total_collected_cents: 2000000, collection_rate_pct: 80, overdue_count: 2 },
  budget_lines: [],
  available_periods: ["2026"],
  role: "admin",
  selected_period: "2026",
  total_budgeted_cents: 0,
  total_actual_cents: 0,
  surplus_cents: 0,
  levy_forecast: {
    status: "shortfall_risk",
    confidence: "medium",
    months_projected: 12,
    current_monthly_levy_cents: 250000,
    average_collection_rate_pct: 81,
    average_monthly_income_cents: 2033333,
    average_monthly_expense_cents: 2266667,
    projected_reserve_balance_cents: -1800000,
    projected_shortfall_cents: 4300000,
    recommended_monthly_increase_cents: 36000,
    recommended_increase_pct: 14,
    data_points: [{ period_label: "March 2026", billed_cents: 2500000, collected_cents: 2000000, collection_rate_pct: 80, expense_cents: 2300000 }],
    notes: ["Projection uses 3 historical levy periods."],
  },
};

describe("FinancialsPage predictive levy analytics", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuthenticatedQuery.mockReturnValue({ data: dashboard, isLoading: false });
  });

  it("shows predictive levy outlook to admins", async () => {
    mockUseAuth.mockReturnValue({ user: { role: "admin" } });
    const { default: FinancialsPage } = await import("@/app/app/[schemeId]/financials/page");

    render(<FinancialsPage />);

    expect(screen.getByText("Predictive levy outlook")).toBeInTheDocument();
    expect(screen.getByText("Shortfall risk")).toBeInTheDocument();
    expect(screen.getByText(/recommended levy increase/i)).toBeInTheDocument();
  });

  it("does not show predictive levy outlook to residents", async () => {
    mockUseAuth.mockReturnValue({ user: { role: "resident" } });
    const { default: FinancialsPage } = await import("@/app/app/[schemeId]/financials/page");

    render(<FinancialsPage />);

    expect(screen.queryByText("Predictive levy outlook")).not.toBeInTheDocument();
  });
});
