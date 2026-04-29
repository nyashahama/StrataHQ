import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import Sidebar from "./Sidebar";

vi.mock("next/navigation", () => ({
  usePathname: vi.fn(() => "/app/scheme-1"),
  useRouter: vi.fn(() => ({ push: vi.fn() })),
}));

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ clearUser: vi.fn() })),
}));

describe("Sidebar", () => {
  it("shows Audit log for agent-scheme role", () => {
    render(
      <Sidebar
        role="agent-scheme"
        headerLabel="Test Scheme"
        schemeId="scheme-1"
      />
    );
    expect(screen.getByText("Audit log")).toBeInTheDocument();
  });

  it("shows Audit log for trustee role", () => {
    render(
      <Sidebar
        role="trustee"
        headerLabel="Test Scheme"
        schemeId="scheme-1"
      />
    );
    expect(screen.getByText("Audit log")).toBeInTheDocument();
  });

  it("does not show Audit log for resident role", () => {
    render(
      <Sidebar
        role="resident"
        headerLabel="Test Scheme"
        schemeId="scheme-1"
      />
    );
    expect(screen.queryByText("Audit log")).not.toBeInTheDocument();
  });
});
