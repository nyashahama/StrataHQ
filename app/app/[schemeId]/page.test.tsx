import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ user: { role: "admin" } })),
}));

vi.mock("next/navigation", () => ({
  useParams: vi.fn(() => ({ schemeId: "scheme-1" })),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));

describe("SchemeOverviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders RetryState when the overview query fails", async () => {
    const refetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load scheme"),
        refetch,
      })
      .mockReturnValueOnce({
        data: { items: [] },
        isLoading: false,
      });

    const { default: SchemeOverviewPage } = await import("@/app/app/[schemeId]/page");
    render(<SchemeOverviewPage />);

    expect(screen.getByText("Could not load scheme overview")).toBeInTheDocument();
  });
});