import { describe, expect, it, vi } from "vitest";

const mockUseQuery = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: mockUseQuery,
}));

describe("useAuthenticatedQuery", () => {
  it("does not carry previous data across different authenticated query keys by default", async () => {
    const { useAuthenticatedQuery } = await import("./useAuthenticatedQuery");
    const queryFn = vi.fn();

    useAuthenticatedQuery({
      queryKey: ["scheme", "scheme-1", "financials"],
      queryFn,
      staleTime: 30_000,
    });

    expect(mockUseQuery).toHaveBeenCalledWith({
      queryKey: ["scheme", "scheme-1", "financials"],
      queryFn,
      staleTime: 30_000,
    });
  });
});
