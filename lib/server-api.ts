import { cookies } from "next/headers";

import { readApiData, buildApiHttpError } from "@/lib/api-contract";
import { clearAuthCookies, refreshAuthSession } from "@/lib/server-auth";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

async function forwardGet(path: string, accessToken?: string): Promise<Response> {
  return fetch(`${BACKEND()}${path}`, {
    method: "GET",
    headers: {
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
    cache: "no-store",
  });
}

export async function fetchBackendJson<T>(path: string): Promise<T> {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;

  let response = await forwardGet(path, accessToken);
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
    response = await forwardGet(path, updatedToken);
  }

  if (!response.ok) {
    throw await buildApiHttpError(response, "Failed to load server data");
  }

  return readApiData<T>(response);
}
