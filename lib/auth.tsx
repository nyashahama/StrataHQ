"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { logoutAction } from "./auth-actions";
import type { SessionUser } from "./session";

interface AuthContextValue {
  user: SessionUser | null;
  loading: boolean;
  clearUser: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function setSessionCookie(user: SessionUser) {
  document.cookie = `sh_session=${encodeURIComponent(JSON.stringify(user))}; path=/; max-age=${30 * 24 * 60 * 60}; SameSite=Lax`;
}

function readSessionCookie(): SessionUser | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)sh_session=([^;]+)/);
  if (!match) return null;
  try {
    return JSON.parse(decodeURIComponent(match[1]));
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setUser(readSessionCookie());
    setLoading(false);
  }, []);

  function clearUser() {
    logoutAction().finally(() => {
      window.location.replace("/auth/login");
    });
  }

  return (
    <AuthContext.Provider value={{ user, loading, clearUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}
