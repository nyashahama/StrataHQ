import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";

const mockUseAuthenticatedQuery = vi.hoisted(() => vi.fn());
const mockApiFetch = vi.hoisted(() => vi.fn());
const mockInvalidateCache = vi.hoisted(() => vi.fn());
const mockAddToast = vi.hoisted(() => vi.fn());

vi.mock("@/lib/auth", () => ({
  useAuth: vi.fn(() => ({ user: { role: "admin" } })),
}));

vi.mock("@/lib/api", () => ({
  apiFetch: mockApiFetch,
}));

vi.mock("@/lib/api-contract", () => ({
  readApiError: vi.fn(),
}));

vi.mock("@/lib/data-cache", () => ({
  invalidateCache: mockInvalidateCache,
}));

vi.mock("@/lib/toast", () => ({
  useToast: vi.fn(() => ({ addToast: mockAddToast })),
}));

vi.mock("next/navigation", () => ({
  useParams: vi.fn(() => ({ schemeId: "scheme-1" })),
}));

vi.mock("@/hooks/useAuthenticatedQuery", () => ({
  useAuthenticatedQuery: mockUseAuthenticatedQuery,
}));

function mockLoadedMembersPage() {
  mockUseAuthenticatedQuery.mockImplementation((config: { queryKey: readonly string[] }) => {
    const isUnitsQuery = config.queryKey[config.queryKey.length - 1] === "units";
    return {
      data: [],
      isLoading: false,
      refetch: vi.fn(),
      ...(isUnitsQuery ? {} : { error: undefined }),
    };
  });
}

describe("MembersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders RetryState when members query fails", async () => {
    const membersRefetch = vi.fn();
    const unitsRefetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load members"),
        refetch: membersRefetch,
      })
      .mockReturnValueOnce({
        data: [],
        isLoading: false,
        refetch: unitsRefetch,
      });

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    expect(screen.getByText("Could not load members")).toBeInTheDocument();
  });

  it("renders RetryState when only units query fails", async () => {
    const membersRefetch = vi.fn();
    const unitsRefetch = vi.fn();

    mockUseAuthenticatedQuery
      .mockReturnValueOnce({
        data: [{ user_id: "u1", full_name: "John", email: "john@test.com", role: "trustee" as const }],
        isLoading: false,
        refetch: membersRefetch,
      })
      .mockReturnValueOnce({
        data: undefined,
        isLoading: false,
        error: new Error("Failed to load units"),
        refetch: unitsRefetch,
      });

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    expect(screen.getByText("Could not load members")).toBeInTheDocument();
  });

  it("does not invalidate member cache when invite validation fails", async () => {
    mockLoadedMembersPage();

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    fireEvent.click(screen.getByRole("button", { name: /\+ Invite member/i }));
    fireEvent.change(screen.getByPlaceholderText("e.g. Nkosi, A."), { target: { value: "Resident User" } });
    fireEvent.change(screen.getByPlaceholderText("e.g. nkosi@email.co.za"), { target: { value: "resident@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    expect(mockAddToast).toHaveBeenCalledWith("Residents require a unit assignment", "error");
    expect(mockInvalidateCache).not.toHaveBeenCalled();
    expect(mockApiFetch).not.toHaveBeenCalled();
  });

  it("invalidates member cache after a valid invite is accepted", async () => {
    mockApiFetch.mockResolvedValueOnce({ ok: true });
    mockLoadedMembersPage();

    const { default: MembersPage } = await import("@/app/app/[schemeId]/members/page");
    render(<MembersPage />);

    fireEvent.click(screen.getByRole("button", { name: /\+ Invite member/i }));
    fireEvent.change(screen.getByPlaceholderText("e.g. Nkosi, A."), { target: { value: "Trustee User" } });
    fireEvent.change(screen.getByPlaceholderText("e.g. nkosi@email.co.za"), { target: { value: "trustee@example.com" } });
    const roleSelect = screen.getAllByRole("combobox").at(0);
    if (!roleSelect) throw new Error("missing role select");
    fireEvent.change(roleSelect, { target: { value: "trustee" } });
    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    await waitFor(() => {
      expect(mockApiFetch).toHaveBeenCalledWith("/api/v1/invitations", expect.any(Object));
    });
    expect(mockInvalidateCache).toHaveBeenCalledWith("scheme:scheme-1:members");
    expect(mockAddToast).toHaveBeenCalledWith("Invite sent to trustee@example.com", "success");
  });
});
