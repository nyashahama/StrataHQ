import { cookies } from 'next/headers'
import { NextRequest } from 'next/server'

import { readApiData, readApiError } from '@/lib/api-contract'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

export async function POST(request: NextRequest) {
  const cookieStore = await cookies()
  const accessToken = cookieStore.get('sh_access')?.value
  if (!accessToken) {
    return new Response('Missing access token.', { status: 401, headers: { 'Content-Type': 'text/plain' } })
  }

  const body = await request.json()
  const response = await fetch(`${BACKEND()}/api/v1/ai/copilot`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(body),
  })

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
