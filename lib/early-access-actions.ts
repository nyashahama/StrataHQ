'use server'

const BACKEND = () => process.env.BACKEND_URL ?? 'http://localhost:8080'

export type EarlyAccessSubmitInput = {
  full_name: string
  email: string
  scheme_name: string
  unit_count: number
}

export async function submitEarlyAccessRequest(
  data: EarlyAccessSubmitInput,
): Promise<{ ok: true } | { error: string }> {
  const res = await fetch(`${BACKEND()}/api/v1/early-access`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    return { error: 'Failed to submit request — please try again' }
  }
  return { ok: true }
}
