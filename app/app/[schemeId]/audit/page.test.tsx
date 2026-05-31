import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ user: { role: "trustee" } })),
}));

vi.mock("next/navigation", () => ({
  useParams: vi.fn(() => ({ schemeId: "scheme-1" })),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));

describe("AuditPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", async () => {
    mockUseAuthenticatedQuery.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
      refetch: vi.fn(),
    });

    const { default: AuditPage } = await import("@/app/app/[schemeId]/audit/page");
    render(<AuditPage />);

    expect(screen.getByText("Loading audit events…")).toBeInTheDocument();
  });

  it("renders RetryState on query failure", async () => {
    mockUseAuthenticatedQuery.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("Failed to load audit events"),
      refetch: vi.fn(),
    });

    const { default: AuditPage } = await import("@/app/app/[schemeId]/audit/page");
    render(<AuditPage />);

    expect(screen.getByText("Could not load audit events")).toBeInTheDocument();
  });

  it("renders empty state when no events", async () => {
    mockUseAuthenticatedQuery.mockReturnValue({
      data: { events: [], total: 0, limit: 50 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    const { default: AuditPage } = await import("@/app/app/[schemeId]/audit/page");
    render(<AuditPage />);

    expect(screen.getByText("No audit events yet.")).toBeInTheDocument();
  });

  it("renders populated event list with expandable details", async () => {
    mockUseAuthenticatedQuery.mockReturnValue({
      data: {
        events: [
          {
            id: "evt-1",
            scheme_id: "scheme-1",
            org_id: "org-1",
            actor_user_id: "user-1",
            actor_role: "admin",
            resource_type: "scheme",
            resource_id: "scheme-1",
            action: "scheme.updated",
            before_state: { name: "Old Name" },
            after_state: { name: "New Name" },
            metadata: null,
            occurred_at: "2026-04-29T10:00:00Z",
          },
        ],
        total: 1,
        limit: 50,
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    const { default: AuditPage } = await import("@/app/app/[schemeId]/audit/page");
    render(<AuditPage />);

    expect(screen.getByText("Scheme Updated")).toBeInTheDocument();
    expect(screen.getByText("Details")).toBeInTheDocument();
  });

  it("shows a truncation message and load more control when total exceeds limit", async () => {
    mockUseAuthenticatedQuery.mockReturnValue({
      data: {
        events: Array.from({ length: 50 }, (_, i) => ({
          id: `evt-${i + 1}`,
          scheme_id: "scheme-1",
          org_id: "org-1",
          actor_user_id: "user-1",
          actor_role: "admin",
          resource_type: "scheme",
          resource_id: "scheme-1",
          action: "scheme.updated",
          before_state: null,
          after_state: null,
          metadata: null,
          occurred_at: "2026-04-29T10:00:00Z",
        })),
        total: 120,
        limit: 50,
      },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    const { default: AuditPage } = await import("@/app/app/[schemeId]/audit/page");
    render(<AuditPage />);

    expect(screen.getByText("Showing latest 50 of 120 audit events.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show more events" })).toBeInTheDocument();
  });
});
