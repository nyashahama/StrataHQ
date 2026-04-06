"use client";

import useSWR, { type SWRConfiguration } from "swr";
import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";

function fetcher<T>(path: string): Promise<T> {
  return apiFetch(path).then(async (res) => {
    if (!res.ok) {
      throw new Error(await readApiError(res, "Failed to fetch"));
    }
    return readApiData<T>(res);
  });
}

export function useApiData<T>(
  path: string | null,
  config?: SWRConfiguration<T>,
) {
  return useSWR<T>(path, (path) => fetcher<T>(path), {
    revalidateOnFocus: true,
    revalidateOnReconnect: true,
    dedupingInterval: 0,
    ...config,
  });
}

export function useMutation<T>(
  path: string,
  options?: RequestInit,
): () => Promise<T> {
  return () =>
    apiFetch(path, options).then(async (res) => {
      if (!res.ok) {
        throw new Error(await readApiError(res, "Request failed"));
      }
      return readApiData<T>(res);
    });
}
