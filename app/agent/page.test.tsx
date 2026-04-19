import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ user: { role: "admin" } })),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));

describe("AgentPortfolioPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("passes the attention queue error to AttentionQueue when query fails", async () => {
    const attentionRefetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: [
          {
            id: "scheme-1",
            name: "Scheme 1",
            address: "Addr",
            unit_count: 1,
            levy_collection_pct: 100,
            open_maintenance_count: 0,
            health: "good" as const,
          },
        ],
        isLoading: false,
      })
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load attention queue"),
        refetch: attentionRefetch,
      });

    const { default: AgentPortfolioPage } = await import("@/app/agent/page");
    render(<AgentPortfolioPage />);

    expect(screen.getByText("Failed to load attention queue")).toBeInTheDocument();
  });
});