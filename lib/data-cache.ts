const STALE_MS = 5 * 60 * 1000 // 5 minutes

interface CacheEntry<T> {
  data: T
  fetchedAt: number
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const cache = new Map<string, CacheEntry<any>>()

export function getCached<T>(key: string): T | null {
  const entry = cache.get(key)
  if (!entry) return null
  if (Date.now() - entry.fetchedAt > STALE_MS) {
    cache.delete(key)
    return null
  }
  return entry.data as T
}

export function setCached<T>(key: string, data: T): void {
  cache.set(key, { data, fetchedAt: Date.now() })
}

/**
 * Invalidate all cache entries whose key starts with `prefix`.
 * Call this after any mutation that changes data for a scheme.
 * Example: invalidateCache(`scheme:${schemeId}`) clears all pages for that scheme.
 */
export function invalidateCache(prefix: string): void {
  for (const key of Array.from(cache.keys())) {
    if (key.startsWith(prefix)) cache.delete(key)
  }
}
