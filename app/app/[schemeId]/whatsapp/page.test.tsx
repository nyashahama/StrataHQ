import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

const mockUseAuth = vi.hoisted(() => vi.fn());
const mockGetCached = vi.hoisted(() => vi.fn());
const mockSetCached = vi.hoisted(() => vi.fn());
const mockInvalidateCache = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: mockUseAuth,
}));

vi.mock("next/navigation", () => ({
  useParams: vi.fn(() => ({ schemeId: "scheme-1" })),
}));

vi.mock("@/lib/toast", () => ({
  useToast: vi.fn(() => ({ addToast: vi.fn() })),
  ToastProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("@/lib/data-cache", () => ({
  getCached: mockGetCached,
  setCached: mockSetCached,
  invalidateCache: mockInvalidateCache,
}));

const mockDashboard = {
  resident_thread: null,
  threads: [
    {
      phone_number: "+27128001234",
      messages: [
        {
          maintenance_request_id: "req-12345678",
          media: [
            { id: "m1", url: "https://example.com/photo.jpg", content_type: "image/jpeg" },
          ],
          id: "msg-1",
          from: "resident" as const,
          text: "My tap is leaking in the kitchen.",
          sent_at: new Date().toISOString(),
        },
        {
          id: "msg-2",
          from: "bot" as const,
          media: [],
          text: "We've logged your maintenance request.",
          sent_at: new Date().toISOString(),
        },
      ],
      id: "thread-1",
      unit_id: "unit-1",
      unit_identifier: "1A",
      owner_name: "John Doe",
      connected: true,
      last_active: new Date().toISOString(),
      unread: 1,
    },
  ],
  broadcasts: [],
  maintenance_intakes: [
    {
      id: "intake-1",
      scheme_id: "scheme-1",
      thread_id: "thread-1",
      message_id: "msg-1",
      unit_id: "unit-1",
      unit_identifier: "1A",
      owner_name: "John Doe",
      status: "candidate" as const,
      category: "plumbing" as const,
      title: "Leaking tap in kitchen",
      description: "The kitchen tap has been dripping continuously for 3 days. Water is pooling on the counter.",
      media_count: 1,
      created_at: new Date().toISOString(),
    },
    {
      id: "intake-2",
      scheme_id: "scheme-1",
      thread_id: "thread-2",
      message_id: "msg-3",
      unit_id: "unit-2",
      unit_identifier: "2B",
      owner_name: "Jane Smith",
      status: "ticket_created" as const,
      category: "electrical" as const,
      title: "Broken light switch",
      description: "The light switch in the living room has stopped working.",
      media_count: 0,
      maintenance_request_id: "req-99999999",
      created_at: new Date().toISOString(),
    },
  ],
  role: "trustee",
  phone_number: "+27128001234",
  total_residents: 10,
  connected_count: 5,
  unread_count: 1,
};

describe("WhatsAppPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetCached.mockReturnValue(mockDashboard);
    mockUseAuth.mockReturnValue({ user: { role: "admin" } });
  });

  it("operator sees Maintenance tab with candidate count", async () => {
    const { default: WhatsAppPage } = await import("@/app/app/[schemeId]/whatsapp/page");
    render(<WhatsAppPage />);

    expect(screen.getByText("Maintenance")).toBeInTheDocument();
  });

  it("candidate card shows Create ticket and Dismiss buttons", async () => {
    const { default: WhatsAppPage } = await import("@/app/app/[schemeId]/whatsapp/page");
    render(<WhatsAppPage />);

    fireEvent.click(screen.getByText("Maintenance"));

    expect(screen.getByText("Unit 1A")).toBeInTheDocument();
    expect(screen.getByText("Leaking tap in kitchen")).toBeInTheDocument();
    expect(screen.getByText("Create ticket")).toBeInTheDocument();
    expect(screen.getByText("Dismiss")).toBeInTheDocument();
  });

  it("ticket-created card shows Ticket created status and reference", async () => {
    const { default: WhatsAppPage } = await import("@/app/app/[schemeId]/whatsapp/page");
    render(<WhatsAppPage />);

    fireEvent.click(screen.getByText("Maintenance"));

    expect(screen.getByText("Unit 2B")).toBeInTheDocument();
    expect(screen.getByText("Broken light switch")).toBeInTheDocument();
    expect(screen.getByText("Ticket created")).toBeInTheDocument();
    expect(screen.getByText(/Ref req-9999/)).toBeInTheDocument();
  });

  it("resident view does not show Maintenance tab", async () => {
    mockUseAuth.mockReturnValue({ user: { role: "resident" } });

    const { default: WhatsAppPage } = await import("@/app/app/[schemeId]/whatsapp/page");
    render(<WhatsAppPage />);

    expect(screen.queryByText("Maintenance")).not.toBeInTheDocument();
    expect(screen.getByText("Connect via WhatsApp")).toBeInTheDocument();
  });
});
