import { isSafeMethod, readBrowserCSRFToken } from "./csrf";

type ApiFetchOptions = RequestInit & {
  auth?: boolean;
};

export async function apiFetch(path: string, options: ApiFetchOptions = {}) {
  const { auth = true, ...requestOptions } = options;
  const csrfToken = isSafeMethod(requestOptions.method)
    ? null
    : readBrowserCSRFToken();
  const isFormData = requestOptions.body instanceof FormData;
  const headers = new Headers(requestOptions.headers);

  if (!isFormData && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  if (csrfToken) {
    headers.set("x-csrf-token", csrfToken);
  }

  if (auth === false) {
    headers.set("x-skip-auth", "true");
  }

  return fetch(`/api/proxy${path}`, {
    ...requestOptions,
    headers,
  });
}
