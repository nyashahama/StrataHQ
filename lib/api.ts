import { refreshTokens } from "./auth-actions";

export async function apiFetch(
  path: string,
  options: RequestInit = {},
): Promise<Response> {
  const proxyPath = `/api/proxy${path}`;

  const res = await fetch(proxyPath, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
  });

  if (res.status !== 401) return res;

  const newToken = await refreshTokens();
  if (!newToken) {
    window.location.replace("/auth/login");
    return res;
  }

  const retry = await fetch(proxyPath, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
  });

  if (retry.status === 401) {
    window.location.replace("/auth/login");
  }

  return retry;
}