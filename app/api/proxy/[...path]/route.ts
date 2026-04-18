import { cookies } from "next/headers";
import { NextRequest } from "next/server";

import { buildAllowedBackendProxyPath } from "@/lib/backend-proxy";
import { clearAuthCookies, refreshAuthSession } from "@/lib/server-auth";
import { getOrCreateRequestId } from "@/lib/request-id";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const UPSTREAM_TIMEOUT_MS = 10000;
const RETRYABLE_STATUS_CODES = [502, 503, 504] as const;

const PROXY_HEADERS = [
  "content-type",
  "accept",
  "accept-language",
  "cache-control",
  "pragma",
] as const;

const IDENTITY_HEADERS = [
  "x-forwarded-for",
  "x-real-ip",
  "user-agent",
  "x-request-id",
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

function upstreamUnavailableResponse() {
  return new Response(
    JSON.stringify({
      error: {
        code: "UPSTREAM_UNAVAILABLE",
        message: "Temporary service issue. Please retry.",
      },
    }),
    {
      status: 503,
      headers: { "Content-Type": "application/json" },
    },
  );
}

function proxyResponse(response: Response, requestId?: string) {
  const responseHeaders = new Headers();
  const contentType = response.headers.get("content-type");
  if (contentType) responseHeaders.set("content-type", contentType);
  if (requestId) responseHeaders.set("x-request-id", requestId);

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
  requestId: string,
  signal?: AbortSignal,
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

  const headers: Record<string, string> = {
    "x-request-id": requestId,
  };
  if (accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`;
  }
  for (const name of PROXY_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers[name] = value;
  }
  for (const name of IDENTITY_HEADERS) {
    if (name === "x-request-id") continue;
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
    signal,
  });

  return response;
}

function shouldRetry(method: string, status: number): boolean {
  return method === "GET" && (status === 502 || status === 503 || status === 504);
}

async function proxyRequest(request: NextRequest, pathSegments: string[]) {
  const requestId = getOrCreateRequestId(request.headers);
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  const auth = request.headers.get("x-skip-auth") !== "true";

  const url = new URL(request.url);
  const backendPath = buildAllowedBackendProxyPath(pathSegments, url.search);
  if (!backendPath) {
    return new Response(null, { status: 404 });
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), UPSTREAM_TIMEOUT_MS);

  let firstResponse: Response;
  try {
    firstResponse = await forwardRequest(
      request,
      url,
      accessToken,
      requestId,
      controller.signal,
    );
  } catch (error) {
    clearTimeout(timeout);
    if (error instanceof DOMException && error.name === "AbortError") {
      return upstreamUnavailableResponse();
    }
    throw error;
  }
  clearTimeout(timeout);

  if (!auth || firstResponse.status !== 401) {
    if (shouldRetry(request.method, firstResponse.status)) {
      const retryController = new AbortController();
      const retryTimeout = setTimeout(
        () => retryController.abort(),
        UPSTREAM_TIMEOUT_MS,
      );

      try {
        const retryResponse = await forwardRequest(
          request,
          url,
          accessToken,
          requestId,
          retryController.signal,
        );
        clearTimeout(retryTimeout);
        return proxyResponse(retryResponse, requestId);
      } catch (error) {
        clearTimeout(retryTimeout);
        if (error instanceof DOMException && error.name === "AbortError") {
          return upstreamUnavailableResponse();
        }
        throw error;
      }
    }
    return proxyResponse(firstResponse, requestId);
  }

  const refreshed = await refreshAuthSession();
  if (!refreshed) {
    await clearAuthCookies();
    return unauthorizedResponse();
  }

  const updatedAccessToken = cookieStore.get("sh_access")?.value;
  const retryResponse = await forwardRequest(request, url, updatedAccessToken, requestId);

  if (retryResponse.status === 401) {
    await clearAuthCookies();
  }

  return proxyResponse(retryResponse, requestId);
}