import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());
const mockAddToast = vi.hoisted(() => vi.fn());

vi.mock("next/navigation", () => ({
  useParams: () => ({ schemeId: "scheme-1" }),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: () => ({
    user: { role: "admin", scheme_memberships: [] },
  }),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));

vi.mock("@/lib/toast", () => ({
  useToast: () => ({ addToast: mockAddToast }),
}));

vi.mock("@/lib/data-cache", () => ({
  invalidateCache: vi.fn(),
}));

vi.mock("@/lib/query-client", () => ({
  queryClient: { invalidateQueries: vi.fn() },
}));

vi.mock("@/lib/maintenance-api", () => ({
  assignMaintenanceRequest: vi.fn(),
  createMaintenanceRequest: vi.fn(),
  getMaintenanceDashboard: vi.fn(),
  resolveMaintenanceRequest: vi.fn(),
}));

vi.mock("@/lib/contractors-api", () => ({
  searchContractorMarketplace: vi.fn(),
}));

import { assignMaintenanceRequest } from "@/lib/maintenance-api";
import MaintenancePage from "./page";

describe("MaintenancePage contractor assignment", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuthenticatedQuery.mockImplementation((options: { queryKey: readonly unknown[] }) => {
      if (options.queryKey.includes("maintenance")) {
        return {
        data: {
          role: "admin",
          open_count: 1,
          sla_breached_count: 0,
          pending_approval_count: 0,
          resolved_this_month: 0,
          requests: [{
            id: "request-1",
            scheme_id: "scheme-1",
            title: "Leaking tap",
            description: "Kitchen tap leaking",
            category: "plumbing",
            status: "open",
            sla_hours: 48,
            created_at: "2026-04-30T00:00:00Z",
            updated_at: "2026-04-30T00:00:00Z",
            sla_breached: false,
          }],
        },
        isLoading: false,
        error: null,
        refetch: vi.fn(),
        };
      }
      return {
        data: [{
          id: "contractor-1",
          org_id: "org-1",
          name: "AquaFix Plumbing",
          trade: "plumbing",
          suburb: "Sea Point",
          city: "Cape Town",
          province: "Western Cape",
          scheme_ids: ["scheme-1"],
          average_rating: 4.8,
          review_count: 3,
          completed_job_count: 5,
          created_at: "2026-04-30T00:00:00Z",
          updated_at: "2026-04-30T00:00:00Z",
          public_profile: true,
          vetted: true,
          active: true,
          preferred: true,
          phone: "+27 21 555 0199",
        }],
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      };
    });
    vi.mocked(assignMaintenanceRequest).mockResolvedValue({
      id: "request-1",
      scheme_id: "scheme-1",
      contractor_id: "contractor-1",
      contractor_name: "AquaFix Plumbing",
      title: "Leaking tap",
      description: "Kitchen tap leaking",
      category: "plumbing",
      status: "in_progress",
      sla_hours: 48,
      created_at: "2026-04-30T00:00:00Z",
      updated_at: "2026-04-30T00:00:00Z",
      sla_breached: false,
    });
  });

  it("assigns a marketplace contractor by contractor_id", async () => {
    render(<MaintenancePage />);

    fireEvent.click(screen.getByRole("button", { name: "Assign" }));
    fireEvent.click(await screen.findByRole("button", { name: /AquaFix Plumbing/i }));
    fireEvent.click(screen.getByRole("button", { name: "Assign contractor" }));

    await waitFor(() => {
      expect(assignMaintenanceRequest).toHaveBeenCalledWith("scheme-1", "request-1", {
        contractor_id: "contractor-1",
      });
    });
  });
});
