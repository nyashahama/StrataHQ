import { refreshTokens, clearAuth } from "./auth-actions";

function getAccessToken(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(/(?:^|;\s*)sh_access=([^;]+)/);
  return match ? match[1] : null;
}

function buildHeaders(token: string | null, extra?: HeadersInit): HeadersInit {
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(extra ?? {}),
  };
}

export async function apiFetch(
  path: string,
  options: RequestInit = {},
): Promise<Response> {
  const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  const token = getAccessToken();

  const res = await fetch(`${base}${path}`, {
    ...options,
    headers: buildHeaders(token, options.headers),
  });

  if (res.status !== 401) return res;

  // Token expired — attempt refresh via server action
  const newToken = await refreshTokens();
  if (!newToken) {
    await clearAuth();
    window.location.replace("/auth/login");
    return res;
  }

  // Retry once with new token
  const retry = await fetch(`${base}${path}`, {
    ...options,
    headers: buildHeaders(newToken, options.headers),
  });

  if (retry.status === 401) {
    await clearAuth();
    window.location.replace("/auth/login");
  }

  return retry;
}
