import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { schemeKeys } from "@/lib/query-keys";

const mockUseAuth = vi.hoisted(() => vi.fn());
const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());
const mockCreateNotice = vi.hoisted(() => vi.fn());
const mockInvalidateCache = vi.hoisted(() => vi.fn());
const mockInvalidateQueries = vi.hoisted(() => vi.fn());
const mockAddToast = vi.hoisted(() => vi.fn());
const mockRefetch = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({ useAuth: mockUseAuth }));
vi.mock("next/navigation", () => ({ useParams: vi.fn(() => ({ schemeId: "scheme-1" })) }));
vi.mock("@/hooks/useAuthenticatedQuery", () => ({ useAuthenticatedQuery: mockUseAuthenticatedQuery }));
vi.mock("@/lib/toast", () => ({ useToast: vi.fn(() => ({ addToast: mockAddToast })) }));
vi.mock("@/lib/data-cache", () => ({ invalidateCache: mockInvalidateCache }));
vi.mock("@/lib/query-client", () => ({ queryClient: { invalidateQueries: mockInvalidateQueries } }));
vi.mock("@/lib/communications-api", () => ({
  getCommunicationsDashboard: vi.fn(),
  createNotice: mockCreateNotice,
}));

describe("CommunicationsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ user: { role: "admin" } });
    mockUseAuthenticatedQuery.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });
  });

  it("shows in-app wording when notice is successfully created", async () => {
    mockCreateNotice.mockResolvedValue({
      id: "notice-1",
      scheme_id: "scheme-1",
      title: "Water outage",
      body: "Please stay away from stairwells.",
      type: "general",
      sent_by_name: "Admin",
      sent_at: "2026-06-01T09:00:00Z",
      created_at: "2026-06-01T09:00:00Z",
    });

    const { default: CommunicationsPage } = await import("@/app/app/[schemeId]/communications/page");
    render(<CommunicationsPage />);

    fireEvent.click(screen.getByRole("button", { name: "+ Compose notice" }));
    fireEvent.change(screen.getByPlaceholderText("Notice subject"), { target: { value: "Water outage" } });
    fireEvent.change(screen.getByPlaceholderText("Write your notice here…"), {
      target: { value: "Please stay away from stairwells." },
    });

    fireEvent.click(screen.getByRole("button", { name: "Publish to scheme" }));

    await waitFor(() => {
      expect(mockCreateNotice).toHaveBeenCalledWith("scheme-1", {
        title: "Water outage",
        body: "Please stay away from stairwells.",
        type: "general",
      });
      expect(mockAddToast).toHaveBeenCalledWith("Notice saved to scheme communications", "success");
    });
    expect(mockInvalidateCache).toHaveBeenCalledWith("scheme:scheme-1:communications");
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: schemeKeys.communicationsBase("scheme-1"),
    });
  });
});
