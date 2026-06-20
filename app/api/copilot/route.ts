import { cookies } from 'next/headers'
import { NextRequest } from 'next/server'

import { readApiData, readApiError } from '@/lib/api-contract'
import { withAuthRetry } from '@/lib/server-auth'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'
const COPILOT_TIMEOUT_MS = 15_000;

function unauthorized(): Response {
  return new Response('Unauthorized', { status: 401, headers: { 'Content-Type': 'text/plain' } });
}

function upstreamUnavailable(): Response {
  return new Response("Copilot temporarily unavailable. Please retry.", {
    status: 503,
    headers: { 'Content-Type': 'text/plain' },
  });
}

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

async function callCopilotWithTimeout(
  accessToken: string,
  body: unknown,
): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), COPILOT_TIMEOUT_MS);
  try {
    return await callCopilot(accessToken, body, controller.signal);
  } finally {
    clearTimeout(timeout);
  }
}

// CopilotTimeoutError signals that the upstream call exceeded the per-attempt
// deadline. We surface it as 503 to the client instead of throwing.
class CopilotTimeoutError extends Error {}

async function callCopilotOrTimeout(
  accessToken: string,
  body: unknown,
): Promise<Response> {
  try {
    return await callCopilotWithTimeout(accessToken, body);
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new CopilotTimeoutError();
    }
    throw error;
  }
}

export async function POST(request: NextRequest) {
  const cookieStore = await cookies()

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

  let result
  try {
    result = await withAuthRetry((token) => callCopilotOrTimeout(token, body))
  } catch (error) {
    if (error instanceof CopilotTimeoutError) {
      return upstreamUnavailable();
    }
    throw error;
  }

  if (result.kind === 'unauthorized') return unauthorized();
  if (result.kind === 'unavailable') return upstreamUnavailable();

  const response = result.response;
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
