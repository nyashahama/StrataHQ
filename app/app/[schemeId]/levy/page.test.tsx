import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockUseAuth = vi.hoisted(() => vi.fn());
const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());
const mockInvalidateCache = vi.hoisted(() => vi.fn());
const mockInvalidateQueries = vi.hoisted(() => vi.fn());
const mockRefetch = vi.hoisted(() => vi.fn());
const mockAddToast = vi.hoisted(() => vi.fn());
const mockGetDraft = vi.hoisted(() => vi.fn());
const mockSendReminder = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({ useAuth: mockUseAuth }));
vi.mock("next/navigation", () => ({ useParams: vi.fn(() => ({ schemeId: "scheme-1" })) }));
vi.mock("@/hooks/useAuthenticatedQuery", () => ({ useAuthenticatedQuery: mockUseAuthenticatedQuery }));
vi.mock("@/lib/toast", () => ({ useToast: vi.fn(() => ({ addToast: mockAddToast })) }));
vi.mock("@/lib/data-cache", () => ({ invalidateCache: mockInvalidateCache }));
vi.mock("@/lib/query-client", () => ({ queryClient: { invalidateQueries: mockInvalidateQueries } }));
vi.mock("@/lib/levy-api", () => ({
  createLevyPeriod: vi.fn(),
  getLevyDashboard: vi.fn(),
  reconcileLevyPayments: vi.fn(),
}));
vi.mock("@/lib/attention-api", () => ({
  getCollectionReminderDraft: mockGetDraft,
  sendCollectionReminder: mockSendReminder,
}));

const dashboard = {
  current_period: {
    id: "period-1",
    scheme_id: "scheme-1",
    label: "May 2026",
    due_date: "2026-05-01",
    amount_cents: 250000,
    created_at: "2026-04-01T00:00:00Z",
  },
  my_account: null,
  collection_trend: [],
  levy_roll: [
    {
      id: "account-1",
      unit_id: "unit-1",
      unit_identifier: "1A",
      owner_name: "Rose Example",
      period_id: "period-1",
      amount_cents: 250000,
      paid_cents: 0,
      status: "overdue" as const,
      due_date: "2026-05-01",
      paid_date: null,
    },
  ],
  my_payments: [],
  role: "admin",
  collection_rate_pct: 0,
  overdue_count: 1,
  total_collected_cents: 0,
};

describe("LevyPaymentsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAuth.mockReturnValue({ user: { role: "admin", scheme_memberships: [] } });
    mockUseAuthenticatedQuery.mockReturnValue({
      data: dashboard,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });
    mockGetDraft.mockResolvedValue({
      account_id: "account-1",
      scheme_id: "scheme-1",
      scheme_name: "Rosewood Estate",
      unit_label: "Unit 1A",
      owner_name: "Rose Example",
      email: { enabled: true, to: "rose@example.com", subject: "Reminder", body: "Email body" },
      whatsapp: { enabled: false, to: "", body: "", disabled_reason: "No WhatsApp connection" },
    });
    mockSendReminder.mockResolvedValue({ event_type: "reminder_sent" });
  });

  it("opens the collection reminder workflow from an overdue levy account", async () => {
    const { default: LevyPaymentsPage } = await import("@/app/app/[schemeId]/levy/page");

    render(<LevyPaymentsPage />);

    fireEvent.click(screen.getByRole("button", { name: "Remind" }));

    expect(await screen.findByText("Send reminder")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("Email body")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Send reminder" }));

    await waitFor(() => {
      expect(mockSendReminder).toHaveBeenCalledWith("scheme-1", "account-1", {
        email: { enabled: true, subject: "Reminder", body: "Email body" },
        whatsapp: { enabled: false, body: "" },
      });
    });
  });
});
