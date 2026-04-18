import { QueryClient } from "@tanstack/react-query";

export interface ApiError extends Error {
  status?: number;
}

function getStatus(error: unknown): number | undefined {
  if (error instanceof Response) {
    return error.status;
  }
  if (error && typeof error === "object" && "status" in error) {
    return (error as ApiError).status;
  }
  return undefined;
}

export function isAuthError(error: unknown): boolean {
  const status = getStatus(error);
  return status === 401;
}

export function isForbidden(error: unknown): boolean {
  const status = getStatus(error);
  return status === 403;
}

export function isNotFound(error: unknown): boolean {
  const status = getStatus(error);
  return status === 404;
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry(failureCount, error) {
        const status = getStatus(error);
        if (status === 401 || status === 403 || status === 404) {
          return false;
        }
        return failureCount < 2;
      },
    },
  },
});