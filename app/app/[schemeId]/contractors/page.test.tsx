import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import ContractorsPage from "./page";

vi.mock("next/navigation", () => ({ useParams: () => ({ schemeId: "scheme-1" }) }));
vi.mock("@/lib/auth", () => ({ useAuth: () => ({ user: { role: "admin" } }) }));
vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: () => ({
    data: [{
      id: "c1", org_id: "o1", name: "AquaFix Plumbing", trade: "plumbing",
      suburb: "Sea Point", city: "Cape Town", province: "Western Cape",
      scheme_ids: ["scheme-1"], average_rating: 4.8, review_count: 3,
      completed_job_count: 5, created_at: "", updated_at: "",
      public_profile: true, vetted: true, active: true, preferred: true,
    }],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
}));

describe("ContractorsPage", () => {
  it("renders contractor directory rows", () => {
    render(<ContractorsPage />);
    expect(screen.getByText("Contractors")).toBeInTheDocument();
    expect(screen.getByText("AquaFix Plumbing")).toBeInTheDocument();
    expect(screen.getByText("Vetted")).toBeInTheDocument();
  });
});
