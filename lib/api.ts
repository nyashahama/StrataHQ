import { isSafeMethod, readBrowserCSRFToken } from "./csrf";

type ApiFetchOptions = RequestInit & {
  auth?: boolean;
};

export async function apiFetch(path: string, options: ApiFetchOptions = {}) {
  const { auth = true, ...requestOptions } = options;
  const csrfToken = isSafeMethod(requestOptions.method)
    ? null
    : readBrowserCSRFToken();

  return fetch(`/api/proxy${path}`, {
    ...requestOptions,
    headers: {
      "Content-Type": "application/json",
      ...(requestOptions.headers ?? {}),
      ...(csrfToken ? { "x-csrf-token": csrfToken } : {}),
      ...(auth === false ? { "x-skip-auth": "true" } : {}),
    },
  });
}
