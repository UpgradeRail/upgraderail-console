/**
 * Next.js passes dynamic segments to client pages still percent-encoded, and
 * indexed ids contain ":" (e.g. "controller:fleet"). Decode once so API paths
 * built with encodeURIComponent are not double-encoded.
 */
export function decodeRouteParam(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
