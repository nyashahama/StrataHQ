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

    async function tryRefreshSession(attempt = 0): Promise<SessionUser | null> {
      try {
        const csrfToken = readBrowserCSRFToken();
        const refreshResponse = await fetch("/api/session/refresh", {
          method: "POST",
          headers: csrfToken ? { "x-csrf-token": csrfToken } : undefined,
        });
        if (refreshResponse.status === 503 && attempt < MAX_REFRESH_RETRIES) {
          return tryRefreshSession(attempt + 1);
        }
        if (!refreshResponse.ok) {
          return null;
        }

        return (await refreshResponse.json()) as SessionUser | null;
      } catch {
        if (attempt < MAX_REFRESH_RETRIES) {
          return tryRefreshSession(attempt + 1);
        }
        return null;
      }
    }

    async function loadSession() {
      try {
        const sessionResponse = await fetch("/api/session");
        const session = (await sessionResponse.json()) as SessionUser | null;
        if (session) {
          setUser(session);
          return;
        }

        const refreshed = await tryRefreshSession();
        setUser(refreshed);
      } catch {
        setUser(null);
      } finally {
        setLoading(false);
      }
    }

    void loadSession();
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
