import { cookies } from "next/headers";

import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import { clearAuthCookies, refreshAuthSession } from "@/lib/server-auth";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const SERVER_FETCH_TIMEOUT_MS = 10_000;

async function forwardGet(path: string, accessToken?: string): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), SERVER_FETCH_TIMEOUT_MS);
  try {
    return await fetch(`${BACKEND()}${path}`, {
      method: "GET",
      headers: {
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
      cache: "no-store",
      signal: controller.signal,
    });
  } finally {
    clearTimeout(timeout);
  }
}

export async function fetchBackendJson<T>(path: string): Promise<T> {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;

  let response: Response;
  try {
    response = await forwardGet(path, accessToken);
  } catch {
    throw new Error("Temporary service issue. Please retry.");
  }
  if (response.status === 401) {
    const refreshed = await refreshAuthSession();
    if (refreshed.kind === "invalid") {
      await clearAuthCookies();
      throw new Error("Unauthorized");
    }

    if (refreshed.kind !== "success") {
      throw new Error("Temporary service issue. Please retry.");
    }

    const updatedToken = cookieStore.get("sh_access")?.value;
    try {
      response = await forwardGet(path, updatedToken);
    } catch {
      throw new Error("Temporary service issue. Please retry.");
    }
  }

  if (!response.ok) {
    throw await buildApiHttpError(response, "Failed to load server data");
  }

  return readApiData<T>(response);
}
