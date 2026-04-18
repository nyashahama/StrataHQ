"use client";

import { useQuery, type UseQueryOptions } from "@tanstack/react-query";

type QueryFetcher<T> = () => Promise<T>;

export function useAuthenticatedQuery<T>(
  options: Omit<UseQueryOptions<T>, "queryKey" | "queryFn"> & {
    queryKey: readonly unknown[];
    queryFn: QueryFetcher<T>;
  }
) {
  const { queryKey, queryFn, ...rest } = options;

  return useQuery<T>({
    queryKey,
    queryFn,
    placeholderData: (previousData) => previousData,
    ...rest,
  });
}