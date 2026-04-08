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
    fetch("/api/session")
      .then((res) => res.json())
      .then((data) => setUser(data))
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
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