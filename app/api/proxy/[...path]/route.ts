import { cookies } from "next/headers";
import { NextRequest } from "next/server";

import { buildAllowedBackendProxyPath } from "@/lib/backend-proxy";
import { clearAuthCookies, refreshAuthSession } from "@/lib/server-auth";
import { getOrCreateRequestId } from "@/lib/request-id";

const BACKEND = () => process.env.BACKEND_URL ?? "http://localhost:8080";

const UPSTREAM_TIMEOUT_MS = 10000;

const PROXY_HEADERS = [
  "content-type",
  "accept",
  "accept-language",
  "cache-control",
  "pragma",
] as const;

const IDENTITY_HEADERS = [
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
  const serverTiming = response.headers.get("server-timing");
  const upstreamStatus = response.headers.get("x-upstream-status");
  if (contentType) responseHeaders.set("content-type", contentType);
  if (serverTiming) responseHeaders.set("server-timing", serverTiming);
  if (upstreamStatus) responseHeaders.set("x-upstream-status", upstreamStatus);
  if (requestId) responseHeaders.set("x-request-id", requestId);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}

async function forwardRequest(args: {
  backendPath: string
  method: string
  headers: Record<string, string>
  body?: BodyInit
  signal?: AbortSignal
}) {
  return fetch(`${BACKEND()}${args.backendPath}`, {
    method: args.method,
    headers: args.headers,
    body: args.body as BodyInit | undefined,
    signal: args.signal,
  })
}

function shouldRetry(method: string, status: number): boolean {
  return method === "GET" && (status === 502 || status === 503 || status === 504);
}

function forbiddenResponse(message: string) {
  return new Response(
    JSON.stringify({
      error: {
        code: "FORBIDDEN",
        message,
      },
    }),
    {
      status: 403,
      headers: { "Content-Type": "application/json" },
    },
  );
}

function requiresCSRFProtection(method: string): boolean {
  return method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
}

async function proxyRequest(request: NextRequest, pathSegments: string[]) {
  const startedAt = performance.now();
  const requestId = getOrCreateRequestId(request.headers);
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;
  const csrfCookie = cookieStore.get("sh_csrf")?.value;
  const auth = request.headers.get("x-skip-auth") !== "true";
  const csrfHeader = request.headers.get("x-csrf-token");

  if (requiresCSRFProtection(request.method)) {
    if (!csrfCookie || !csrfHeader || csrfHeader !== csrfCookie) {
      return forbiddenResponse("Invalid CSRF token");
    }
  }

  const url = new URL(request.url);
  const backendPath = buildAllowedBackendProxyPath(pathSegments, url.search);
  if (!backendPath) {
    return new Response(null, { status: 404 });
  }

  let requestBody: BodyInit | undefined;
  let requestBodyRetry: BodyInit | undefined;
  if (request.method !== "GET" && request.method !== "HEAD" && request.body) {
    const [firstBody, secondBody] = request.body.tee();
    requestBody = firstBody;
    requestBodyRetry = secondBody;
  }

  const proxyHeaders: Record<string, string> = {
    "x-request-id": requestId,
  };
  if (accessToken) {
    proxyHeaders["Authorization"] = `Bearer ${accessToken}`;
  }
  for (const name of PROXY_HEADERS) {
    const value = request.headers.get(name);
    if (value) proxyHeaders[name] = value;
  }
  for (const name of IDENTITY_HEADERS) {
    if (name === "x-request-id") continue;
    const value = request.headers.get(name);
    if (value) proxyHeaders[name] = value;
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), UPSTREAM_TIMEOUT_MS);

  let firstResponse: Response;
  try {
    firstResponse = await forwardRequest({
      backendPath,
      method: request.method,
      headers: proxyHeaders,
      body: requestBody,
      signal: controller.signal,
    });
  } catch (error) {
    clearTimeout(timeout);
    if (error instanceof DOMException && error.name === "AbortError") {
      return upstreamUnavailableResponse();
    }
    throw error;
  }
  clearTimeout(timeout);

  function withTiming(response: Response): Response {
    const headers = new Headers(response.headers);
    headers.set("server-timing", `upstream;dur=${(performance.now() - startedAt).toFixed(1)}`);
    headers.set("x-upstream-status", String(response.status));

    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers,
    });
  }

  if (!auth || firstResponse.status !== 401) {
    if (shouldRetry(request.method, firstResponse.status)) {
      const retryController = new AbortController();
      const retryTimeout = setTimeout(
        () => retryController.abort(),
        UPSTREAM_TIMEOUT_MS,
      );

      try {
        const retryResponse = await forwardRequest({
          backendPath,
          method: request.method,
          headers: proxyHeaders,
          body: requestBodyRetry,
          signal: retryController.signal,
        });
        clearTimeout(retryTimeout);
        return proxyResponse(withTiming(retryResponse), requestId);
      } catch (error) {
        clearTimeout(retryTimeout);
        if (error instanceof DOMException && error.name === "AbortError") {
          return upstreamUnavailableResponse();
        }
        throw error;
      }
    }
    return proxyResponse(withTiming(firstResponse), requestId);
  }

  const refreshed = await refreshAuthSession();
  if (refreshed.kind === "invalid") {
    await clearAuthCookies();
    return unauthorizedResponse();
  }
  if (refreshed.kind === "unavailable") {
    return upstreamUnavailableResponse();
  }

  const updatedAccessToken = cookieStore.get("sh_access")?.value;
  const updatedHeaders = { ...proxyHeaders, Authorization: `Bearer ${updatedAccessToken}` };
  const refreshRetryController = new AbortController();
  const refreshRetryTimeout = setTimeout(
    () => refreshRetryController.abort(),
    UPSTREAM_TIMEOUT_MS,
  );
  let retryResponse: Response;
  try {
    retryResponse = await forwardRequest({
      backendPath,
      method: request.method,
      headers: updatedHeaders,
      body: requestBodyRetry ?? requestBody,
      signal: refreshRetryController.signal,
    });
  } catch (error) {
    clearTimeout(refreshRetryTimeout);
    if (error instanceof DOMException && error.name === "AbortError") {
      return upstreamUnavailableResponse();
    }
    throw error;
  }
  clearTimeout(refreshRetryTimeout);

  if (retryResponse.status === 401) {
    await clearAuthCookies();
  }

  return proxyResponse(withTiming(retryResponse), requestId);
}
