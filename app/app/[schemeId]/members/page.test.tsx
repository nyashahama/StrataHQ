import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

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

describe("MembersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders RetryState when members query fails", async () => {
    const membersRefetch = vi.fn();
    const unitsRefetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load members"),
        refetch: membersRefetch,
      })
      .mockReturnValueOnce({
        data: [],
        isLoading: false,
        refetch: unitsRefetch,
      });

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    expect(screen.getByText("Could not load members")).toBeInTheDocument();
  });

  it("renders RetryState when only units query fails", async () => {
    const membersRefetch = vi.fn();
    const unitsRefetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: [{ user_id: "u1", full_name: "John", email: "john@test.com", role: "trustee" as const }],
        isLoading: false,
        refetch: membersRefetch,
      })
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load units"),
        refetch: unitsRefetch,
      });

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    expect(screen.getByText("Could not load members")).toBeInTheDocument();
  });
});