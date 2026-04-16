import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import CollectionExecutionModal from "./CollectionExecutionModal";

vi.mock("@/lib/attention-api", () => ({
  getCollectionReminderDraft: vi.fn().mockResolvedValue({
    account_id: "acc-1",
    scheme_id: "scheme-1",
    scheme_name: "Rosewood Estate",
    unit_label: "Unit 5C",
    owner_name: "Rose Example",
    email: { enabled: true, to: "rose@example.com", subject: "Levy arrears reminder for Rosewood Estate", body: "Email body" },
    whatsapp: { enabled: true, to: "+27715550101", body: "WhatsApp body" },
  }),
  sendCollectionReminder: vi.fn().mockResolvedValue({ event_type: "reminder_sent" }),
}));

describe("CollectionExecutionModal", () => {
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
});