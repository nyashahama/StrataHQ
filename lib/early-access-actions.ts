'use server'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'
const EARLY_ACCESS_FETCH_TIMEOUT_MS = 10_000
const TEMP_UNAVAILABLE = 'Service temporarily unavailable — please try again'

async function fetchWithTimeout(
  url: string,
  options: RequestInit,
  timeoutMs = EARLY_ACCESS_FETCH_TIMEOUT_MS,
): Promise<Response> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), timeoutMs)
  try {
    return await fetch(url, { ...options, signal: controller.signal })
  } finally {
    clearTimeout(timeout)
  }
}

export type EarlyAccessSubmitInput = {
  full_name: string
  email: string
  scheme_name: string
  unit_count: number
}

export async function submitEarlyAccessRequest(
  data: EarlyAccessSubmitInput,
): Promise<{ ok: true } | { error: string }> {
  let res: Response
  try {
    res = await fetchWithTimeout(`${BACKEND()}/api/v1/early-access`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
  } catch {
    return { error: TEMP_UNAVAILABLE }
  }
  if (!res.ok) {
    return { error: 'Failed to submit request — please try again' }
  }
  return { ok: true }
}
