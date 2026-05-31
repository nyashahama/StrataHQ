import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { SessionUser } from "@/lib/session";
import Copilot from "./Copilot";

const { useAuth, usePathname } = vi.hoisted(() => ({
  useAuth: vi.fn(),
  usePathname: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  useAuth,
}));

vi.mock("next/navigation", () => ({
  usePathname,
}));

function makeUser(overrides: Partial<SessionUser> = {}): SessionUser {
  return {
    id: "user-1",
    email: "person@example.com",
    full_name: "Person Example",
    phone: null,
    role: "admin",
    wizard_complete: true,
    scheme_memberships: [
      {
        scheme_id: "scheme-1",
        scheme_name: "Scheme 1",
        unit_id: null,
        unit_identifier: null,
        role: "admin",
      },
    ],
    org: { id: "org-1", name: "Org 1" },
    ...overrides,
  };
}

describe("Copilot", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    usePathname.mockReturnValue("/agent");
    Element.prototype.scrollIntoView = vi.fn();
  });

  it("prevents concurrent sends from the same session", async () => {
    const responseStream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('hello'))
        controller.close()
      },
    })

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(responseStream, { status: 200 }),
    )

    useAuth.mockReturnValue({ user: makeUser() });
    usePathname.mockReturnValue('/app/scheme-1/financials');

    const { getByRole } = render(<Copilot />);

    fireEvent.click(getByRole('button', { name: 'Open AI Copilot' }));

    const firstQuestion = getByRole('button', {
      name: 'Which schemes have the lowest levy collection rates?',
    });
    fireEvent.click(firstQuestion);
    fireEvent.click(firstQuestion);

    expect(fetchSpy).toHaveBeenCalledTimes(1);

    await new Promise(resolve => setTimeout(resolve, 0));
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("preserves hook order when auth state changes from hidden to visible", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    useAuth.mockReturnValue({ user: null });

    const { rerender } = render(<Copilot />);

    useAuth.mockReturnValue({ user: makeUser() });

    expect(() => rerender(<Copilot />)).not.toThrow();
    expect(consoleError).not.toHaveBeenCalled();
  });
});
