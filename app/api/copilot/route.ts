import { cookies } from 'next/headers'
import { NextRequest } from 'next/server'

import { readApiData, readApiError } from '@/lib/api-contract'
import { clearAuthCookies, refreshAuthSession } from '@/lib/server-auth'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'
const COPILOT_TIMEOUT_MS = 15_000;

async function callCopilot(
  accessToken: string,
  body: unknown,
  signal: AbortSignal,
): Promise<Response> {
  return fetch(`${BACKEND()}/api/v1/ai/copilot`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(body),
    signal,
  });
}

export async function POST(request: NextRequest) {
  const cookieStore = await cookies()
  const accessToken = cookieStore.get('sh_access')?.value
  if (!accessToken) {
    return new Response('Missing access token.', { status: 401, headers: { 'Content-Type': 'text/plain' } })
  }

  const csrfCookie = cookieStore.get('sh_csrf')?.value
  const csrfHeader = request.headers.get('x-csrf-token')
  if (!csrfCookie || !csrfHeader || csrfHeader !== csrfCookie) {
    return new Response(JSON.stringify({ error: { code: 'FORBIDDEN', message: 'Invalid CSRF token' } }), {
      status: 403,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  let body: unknown
  try {
    body = await request.json()
  } catch {
    return new Response('Invalid JSON request body.', {
      status: 400,
      headers: { 'Content-Type': 'text/plain' },
    })
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), COPILOT_TIMEOUT_MS);
  let response: Response;
  try {
    response = await callCopilot(accessToken, body, controller.signal);
  } catch (error) {
    clearTimeout(timeout);
    if (error instanceof DOMException && error.name === "AbortError") {
      return new Response("Copilot temporarily unavailable. Please retry.", {
        status: 503,
        headers: { "Content-Type": "text/plain" },
      });
    }
    throw error;
  }
  clearTimeout(timeout);

  if (response.status === 401) {
    const refreshed = await refreshAuthSession();
    if (refreshed.kind === "invalid") {
      await clearAuthCookies();
      return new Response('Unauthorized', { status: 401, headers: { 'Content-Type': 'text/plain' } });
    }
    if (refreshed.kind === "unavailable") {
      return new Response("Copilot temporarily unavailable. Please retry.", {
        status: 503,
        headers: { "Content-Type": "text/plain" },
      });
    }
    const updatedAccessToken = cookieStore.get("sh_access")?.value;
    if (!updatedAccessToken) {
      await clearAuthCookies();
      return new Response('Unauthorized', { status: 401, headers: { 'Content-Type': 'text/plain' } });
    }
    const retryController = new AbortController();
    const retryTimeout = setTimeout(() => retryController.abort(), COPILOT_TIMEOUT_MS);
    try {
      response = await callCopilot(updatedAccessToken, body, retryController.signal);
    } catch (error) {
      clearTimeout(retryTimeout);
      if (error instanceof DOMException && error.name === "AbortError") {
        return new Response("Copilot temporarily unavailable. Please retry.", {
          status: 503,
          headers: { "Content-Type": "text/plain" },
        });
      }
      throw error;
    }
    clearTimeout(retryTimeout);
    if (response.status === 401) {
      await clearAuthCookies();
    }
  }

  if (!response.ok) {
    return new Response(
      await readApiError(response, 'Failed to generate copilot response.'),
      { status: response.status, headers: { 'Content-Type': 'text/plain' } },
    )
  }

  const data = await readApiData<{ answer: string }>(response)
  const encoder = new TextEncoder()
  const readable = new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(data.answer))
      controller.close()
    },
  })

  return new Response(readable, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'no-cache',
    },
  })
}
