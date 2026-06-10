import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CollectionExecutionModal from "./CollectionExecutionModal";

const mockGetCollectionReminderDraft = vi.hoisted(() => vi.fn());
const mockSendCollectionReminder = vi.hoisted(() => vi.fn());
const mockAddToast = vi.hoisted(() => vi.fn());

vi.mock("@/lib/attention-api", () => ({
  getCollectionReminderDraft: mockGetCollectionReminderDraft,
  sendCollectionReminder: mockSendCollectionReminder,
}));

vi.mock("@/lib/toast", () => ({
  useToast: vi.fn(() => ({ addToast: mockAddToast })),
}));

describe("CollectionExecutionModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetCollectionReminderDraft.mockResolvedValue({
      account_id: "acc-1",
      scheme_id: "scheme-1",
      scheme_name: "Rosewood Estate",
      unit_label: "Unit 5C",
      owner_name: "Rose Example",
      email: { enabled: true, to: "rose@example.com", subject: "Levy arrears reminder for Rosewood Estate", body: "Email body" },
      whatsapp: { enabled: true, to: "+27715550101", body: "WhatsApp body" },
    });
    mockSendCollectionReminder.mockResolvedValue({ event_type: "reminder_sent" });
  });

  it("loads the reminder draft and submits edited email and WhatsApp bodies", async () => {
    const onSent = vi.fn();

    render(
      <CollectionExecutionModal
        open={true}
        schemeId="scheme-1"
        accountId="acc-1"
        onClose={vi.fn()}
        onSent={onSent}
      />
    );

    expect(await screen.findByDisplayValue("Email body")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/email body/i), { target: { value: "Updated email body" } });
    fireEvent.click(screen.getByRole("button", { name: /send reminder/i }));

    await waitFor(() => expect(onSent).toHaveBeenCalled());
  });

  it("shows a retryable error when the reminder draft cannot load", async () => {
    mockGetCollectionReminderDraft.mockRejectedValueOnce(new Error("draft unavailable"));

    render(
      <CollectionExecutionModal
        open={true}
        schemeId="scheme-1"
        accountId="acc-1"
        onClose={vi.fn()}
        onSent={vi.fn()}
      />
    );

    expect(await screen.findByText("Could not load reminder draft")).toBeInTheDocument();
    expect(screen.queryByText("Loading reminder draft…")).not.toBeInTheDocument();
  });
});
