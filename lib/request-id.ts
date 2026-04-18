export function getOrCreateRequestId(headers: Headers): string {
  const existing = headers.get("x-request-id");
  if (existing) {
    return existing;
  }

  const id = crypto.randomUUID();
  return id;
}