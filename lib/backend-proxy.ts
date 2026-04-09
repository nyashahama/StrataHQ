export function buildAllowedBackendProxyPath(
  pathSegments: string[],
  search: string,
): string | null {
  if (pathSegments.length < 2) {
    return null;
  }

  if (pathSegments[0] !== "api" || pathSegments[1] !== "v1") {
    return null;
  }

  for (const segment of pathSegments) {
    if (!segment || segment === "." || segment === "..") {
      return null;
    }
  }

  return `/${pathSegments.join("/")}${search}`;
}
