'use server'

import { cookies } from 'next/headers'
import { readApiData } from './api-contract'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

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

export async function listEarlyAccessRequests(): Promise<EarlyAccessRequest[]> {
  const token = await getAccessToken()
  if (!token) return []
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: 'no-store',
  })
  if (!res.ok) return []
  return readApiData<EarlyAccessRequest[]>(res)
}

export async function approveEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const token = await getAccessToken()
  if (!token) return { error: 'Not authenticated' }
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access/${id}/approve`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return { error: 'Failed to approve' }
  return { ok: true }
}

export async function rejectEarlyAccessRequest(id: string): Promise<{ ok: true } | { error: string }> {
  const token = await getAccessToken()
  if (!token) return { error: 'Not authenticated' }
  const res = await fetch(`${BACKEND()}/api/v1/admin/early-access/${id}/reject`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return { error: 'Failed to reject' }
  return { ok: true }
}
