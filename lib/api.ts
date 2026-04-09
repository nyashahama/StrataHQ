import { refreshTokens } from "./auth-actions";

type ApiFetchOptions = RequestInit & {
  auth?: boolean;
};

export async function apiFetch(
  path: string,
  options: ApiFetchOptions = {},
): Promise<Response> {
  const proxyPath = `/api/proxy${path}`;
  const { auth = true, ...requestOptions } = options;

  const res = await fetch(proxyPath, {
    ...requestOptions,
    headers: {
      "Content-Type": "application/json",
      ...(requestOptions.headers ?? {}),
    },
  });

  if (!auth || res.status !== 401) return res;

  const newToken = await refreshTokens();
  if (!newToken) {
    window.location.replace("/auth/login");
    return res;
  }

  const retry = await fetch(proxyPath, {
    ...requestOptions,
    headers: {
      "Content-Type": "application/json",
      ...(requestOptions.headers ?? {}),
    },
  });

  if (retry.status === 401) {
    window.location.replace("/auth/login");
  }

  return retry;
}
