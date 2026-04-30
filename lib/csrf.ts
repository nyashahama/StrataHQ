export function isSafeMethod(method: string | undefined): boolean {
  const normalized = (method ?? "GET").toUpperCase();
  return normalized === "GET" || normalized === "HEAD" || normalized === "OPTIONS";
}

export function readBrowserCSRFToken(): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const cookies = document.cookie.split(";").map((entry) => entry.trim());
  for (const cookie of cookies) {
    if (cookie.startsWith("sh_csrf=")) {
      return decodeURIComponent(cookie.slice("sh_csrf=".length));
    }
  }
  return null;
}
