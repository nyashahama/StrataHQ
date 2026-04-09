import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

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
  beforeEach(() => {
    vi.restoreAllMocks();
    usePathname.mockReturnValue("/agent");
    Element.prototype.scrollIntoView = vi.fn();
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
