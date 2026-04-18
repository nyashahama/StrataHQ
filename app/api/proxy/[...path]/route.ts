import { cookies } from "next/headers";
import { NextRequest } from "next/server";

import { buildAllowedBackendProxyPath } from "@/lib/backend-proxy";
import { clearAuthCookies, refreshAuthSession } from "@/lib/server-auth";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const PROXY_HEADERS = [
  "content-type",
  "accept",
  "accept-language",
  "cache-control",
  "pragma",
] as const;

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return proxyRequest(request, (await params).path);
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return proxyRequest(request, (await params).path);
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return proxyRequest(request, (await params).path);
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return proxyRequest(request, (await params).path);
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return proxyRequest(request, (await params).path);
}

function unauthorizedResponse() {
  return new Response(null, {
    status: 401,
    statusText: "Unauthorized",
  });
}

function proxyResponse(response: Response) {
  const responseHeaders = new Headers();
  const contentType = response.headers.get("content-type");
  if (contentType) responseHeaders.set("content-type", contentType);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}

async function forwardRequest(
  request: NextRequest,
  url: URL,
  accessToken: string | undefined,
) {
  const cookieStore = await cookies();

  const backendPath = buildAllowedBackendProxyPath(
    request.url.split("/api/proxy/")[1]?.split("/") ?? [],
    url.search,
  );
  if (!backendPath) {
    return new Response(null, { status: 404 });
  }
  const backendUrl = `${BACKEND()}${backendPath}`;

  const headers: Record<string, string> = {};
  if (accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`;
  }
  for (const name of PROXY_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers[name] = value;
  }

  let body: BodyInit | undefined;
  if (request.method !== "GET" && request.method !== "HEAD") {
    body = await request.bytes();
  }

  const response = await fetch(backendUrl, {
    method: request.method,
    headers,
    body,
  });

  return response;
}

async function proxyRequest(request: NextRequest, pathSegments: string[]) {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  const auth = request.headers.get("x-skip-auth") !== "true";

  const url = new URL(request.url);
  const backendPath = buildAllowedBackendProxyPath(pathSegments, url.search);
  if (!backendPath) {
    return new Response(null, { status: 404 });
  }

  const firstResponse = await forwardRequest(request, url, accessToken);
  if (!auth || firstResponse.status !== 401) {
    return proxyResponse(firstResponse);
  }

  const refreshed = await refreshAuthSession();
  if (!refreshed) {
    await clearAuthCookies();
    return unauthorizedResponse();
  }

  const updatedAccessToken = cookieStore.get("sh_access")?.value;
  const retryResponse = await forwardRequest(request, url, updatedAccessToken);

  if (retryResponse.status === 401) {
    await clearAuthCookies();
  }

  return proxyResponse(retryResponse);
}