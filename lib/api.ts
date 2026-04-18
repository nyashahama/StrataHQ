type ApiFetchOptions = RequestInit & {
  auth?: boolean;
};

export async function apiFetch(path: string, options: ApiFetchOptions = {}) {
  const { auth = true, ...requestOptions } = options;

  return fetch(`/api/proxy${path}`, {
    ...requestOptions,
    headers: {
      "Content-Type": "application/json",
      ...(requestOptions.headers ?? {}),
      ...(auth === false ? { "x-skip-auth": "true" } : {}),
    },
  });
}