import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import PendingPage from "./page";

const replace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

describe("PendingPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("redirects to dashboard when refresh reveals active session", async () => {
    document.cookie = "sh_csrf=pending-csrf-token";

    const session = {
      id: "00000000-0000-4000-8000-000000000001",
      email: "admin@example.com",
      full_name: "Admin One",
      role: "admin",
      wizard_complete: true,
      scheme_memberships: [
        {
          scheme_id: "123e4567-e89b-12d3-a456-426614174000",
          scheme_name: "Sunridge",
          unit_id: null,
          role: "admin",
        },
      ],
    };

    vi.spyOn(global, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(session), { status: 200 }),
    );

    render(<PendingPage />);

    fireEvent.click(screen.getByRole("button", { name: /check again/i }));

    await waitFor(() => {
      expect(replace).toHaveBeenCalledWith("/agent");
    });

    expect(replace).toHaveBeenCalledTimes(1);
  });

  it("shows a message when the account is still pending", async () => {
    replace.mockClear();
    document.cookie = "sh_csrf=pending-csrf-token";

    vi.spyOn(global, "fetch").mockResolvedValueOnce(
      new Response("", { status: 401 }),
    );

    render(<PendingPage />);

    fireEvent.click(screen.getByRole("button", { name: /check again/i }));

    await waitFor(() => {
      expect(screen.getByText(/still pending setup/i)).toBeInTheDocument();
    });

    expect(replace).not.toHaveBeenCalled();
  });
});
