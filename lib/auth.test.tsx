import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider, useAuth } from "./auth";

vi.mock("./auth-actions", () => ({
  logoutAction: vi.fn(),
}));

function AuthProbe() {
  const { user, loading } = useAuth();

  return (
    <div>
      <span data-testid="loading">{String(loading)}</span>
      <span data-testid="email">{user?.email ?? "none"}</span>
    </div>
  );
}

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("uses initialUser without calling /api/session", () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    render(
      <AuthProvider
        initialUser={{
          id: "user-1",
          email: "person@example.com",
          full_name: "Person Example",
          phone: null,
          role: "admin",
          wizard_complete: true,
          scheme_memberships: [],
          org: { id: "org-1", name: "Org 1" },
        }}
      >
        <AuthProbe />
      </AuthProvider>,
    );

    expect(screen.getByTestId("loading")).toHaveTextContent("false");
    expect(screen.getByTestId("email")).toHaveTextContent("person@example.com");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("refreshes session state when the cached session is invalid but recoverable", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response("null", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "user-1",
            email: "person@example.com",
            full_name: "Person Example",
            phone: null,
            role: "admin",
            wizard_complete: true,
            scheme_memberships: [],
            org: { id: "org-1", name: "Org 1" },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    render(
      <AuthProvider>
        <AuthProbe />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
      expect(screen.getByTestId("email")).toHaveTextContent("person@example.com");
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/session");
    expect(fetchMock).toHaveBeenNthCalledWith(2, "/api/session/refresh", {
      method: "POST",
    });
  });

  it("retries temporary refresh unavailability before giving up", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response("null", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            error: {
              code: "UPSTREAM_UNAVAILABLE",
              message: "Temporary service issue. Please retry.",
            },
          }),
          {
            status: 503,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: "user-1",
            email: "person@example.com",
            full_name: "Person Example",
            phone: null,
            role: "admin",
            wizard_complete: true,
            scheme_memberships: [],
            org: { id: "org-1", name: "Org 1" },
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );

    vi.stubGlobal("fetch", fetchMock);

    render(
      <AuthProvider>
        <AuthProbe />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("loading")).toHaveTextContent("false");
      expect(screen.getByTestId("email")).toHaveTextContent("person@example.com");
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, "/api/session");
    expect(fetchMock).toHaveBeenNthCalledWith(2, "/api/session/refresh", {
      method: "POST",
    });
    expect(fetchMock).toHaveBeenNthCalledWith(3, "/api/session/refresh", {
      method: "POST",
    });
  });
});
