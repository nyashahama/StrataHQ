"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { logoutAction } from "./auth-actions";
import type { SessionUser } from "./session";

interface AuthContextValue {
  user: SessionUser | null;
  loading: boolean;
  clearUser: () => void;
  setUser: (user: SessionUser | null) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadSession() {
      try {
        const sessionResponse = await fetch("/api/session");
        const session = (await sessionResponse.json()) as SessionUser | null;

        if (session) {
          setUser(session);
          return;
        }

        const refreshResponse = await fetch("/api/session/refresh", {
          method: "POST",
        });
        if (!refreshResponse.ok) {
          setUser(null);
          return;
        }

        const refreshed = (await refreshResponse.json()) as SessionUser | null;
        setUser(refreshed);
      } catch {
        setUser(null);
      } finally {
        setLoading(false);
      }
    }

    void loadSession();
  }, []);

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
