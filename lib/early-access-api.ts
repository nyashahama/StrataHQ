'use server'

import { cookies } from 'next/headers'
import { buildApiHttpError, readApiData, readApiError } from './api-contract'
import { clearAuthCookies, refreshAuthSession } from './server-auth'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'
const NOT_AUTHENTICATED = 'Not authenticated'
const TEMPORARY_UNAVAILABLE = 'Temporary service issue. Please retry.'

export type EarlyAccessRequest = {
  id: string
  full_name: string
  email: string
  scheme_name: string
  unit_count: number
  status: 'pending' | 'approved' | 'rejected'
  created_at: string
  reviewed_at?: string
}

async function getAccessToken(): Promise<string | null> {
  const cookieStore = await cookies()
  return cookieStore.get('sh_access')?.value ?? null
}

function apiErrorResponse(status: number, message: string): Response {
  return new Response(JSON.stringify({ error: { message } }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function withAuthHeader(headers: HeadersInit | undefined, token: string): Record<string, string> {
  return {
    ...Object.fromEntries(new Headers(headers).entries()),
    Authorization: `Bearer ${token}`,
  }
}

async function fetchAdmin(path: string, init: RequestInit = {}): Promise<Response> {
  const send = (token: string) => {
    const { headers, ...rest } = init
    return fetch(`${BACKEND()}${path}`, {
      ...rest,
      headers: withAuthHeader(headers, token),
    })
  }

  const token = await getAccessToken()
  if (!token) return apiErrorResponse(401, NOT_AUTHENTICATED)

  const res = await send(token)
  if (res.status !== 401) return res

  const refreshed = await refreshAuthSession()
  if (refreshed.kind === 'invalid') {
    await clearAuthCookies()
    return apiErrorResponse(401, NOT_AUTHENTICATED)
  }
  if (refreshed.kind !== 'success') {
    return apiErrorResponse(503, TEMPORARY_UNAVAILABLE)
  }

  const refreshedToken = await getAccessToken()
  if (!refreshedToken) return apiErrorResponse(401, NOT_AUTHENTICATED)

  return send(refreshedToken)
}

export async function listEarlyAccessRequests(): Promise<EarlyAccessRequest[]> {
  const res = await fetchAdmin('/api/v1/admin/early-access', {
    cache: 'no-store',
  })
  if (!res.ok) throw await buildApiHttpError(res, 'Failed to load early access requests')
  return readApiData<EarlyAccessRequest[]>(res)
}

export async function approveEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const res = await fetchAdmin(`/api/v1/admin/early-access/${id}/approve`, {
    method: 'POST',
  })
  if (!res.ok) return { error: await readApiError(res, 'Failed to approve') }
  return { ok: true }
}

export async function rejectEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const res = await fetchAdmin(`/api/v1/admin/early-access/${id}/reject`, {
    method: 'POST',
  })
  if (!res.ok) return { error: await readApiError(res, 'Failed to reject') }
  return { ok: true }
}
