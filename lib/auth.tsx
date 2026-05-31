"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { logoutAction } from "./auth-actions";
import { readBrowserCSRFToken } from "./csrf";
import type { SessionUser } from "./session";

interface AuthContextValue {
  user: SessionUser | null;
  loading: boolean;
  clearUser: () => void;
  setUser: (user: SessionUser | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface AuthProviderProps {
  children: React.ReactNode;
  initialUser?: SessionUser | null;
}

export function AuthProvider({ children, initialUser }: AuthProviderProps) {
  const [user, setUser] = useState<SessionUser | null>(initialUser ?? null);
  const [loading, setLoading] = useState(initialUser === undefined);

  useEffect(() => {
    if (initialUser !== undefined) {
      return;
    }

    const MAX_REFRESH_RETRIES = 2;
    const controller = new AbortController();
    let active = true;

    async function tryRefreshSession(attempt = 0): Promise<SessionUser | null> {
      try {
        const csrfToken = readBrowserCSRFToken();
        const refreshResponse = await fetch("/api/session/refresh", {
          method: "POST",
          headers: csrfToken ? { "x-csrf-token": csrfToken } : undefined,
          signal: controller.signal,
        });
        if (refreshResponse.status === 503 && attempt < MAX_REFRESH_RETRIES) {
          return tryRefreshSession(attempt + 1);
        }
        if (!refreshResponse.ok) {
          return null;
        }

        return (await refreshResponse.json()) as SessionUser | null;
      } catch {
        if (controller.signal.aborted) {
          return null;
        }
        if (attempt < MAX_REFRESH_RETRIES) {
          return tryRefreshSession(attempt + 1);
        }
        return null;
      }
    }

    async function loadSession() {
      try {
        const sessionResponse = await fetch("/api/session", {
          signal: controller.signal,
        });
        const session = (await sessionResponse.json()) as SessionUser | null;
        if (session) {
          if (active) setUser(session);
          return;
        }

        const refreshed = await tryRefreshSession();
        if (active) setUser(refreshed);
      } catch {
        if (active) setUser(null);
      } finally {
        if (active) setLoading(false);
      }
    }

    void loadSession();

    return () => {
      active = false;
      controller.abort();
    };
  }, [initialUser]);

  function clearUser() {
    logoutAction().finally(() => {
      window.location.replace("/auth/login");
    });
  }

  return (
    <AuthContext.Provider value={{ user, loading, clearUser, setUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
