import { cookies } from "next/headers";
import { NextRequest } from "next/server";

import { buildAllowedBackendProxyPath } from "@/lib/backend-proxy";

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

async function proxyRequest(
  request: NextRequest,
  pathSegments: string[],
) {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get("sh_access")?.value;

  const url = new URL(request.url);
  const backendPath = buildAllowedBackendProxyPath(pathSegments, url.search);
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

  const responseHeaders = new Headers();
  const contentType = response.headers.get("content-type");
  if (contentType) responseHeaders.set("content-type", contentType);

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}
