export class ApiError extends Error { constructor(readonly status: number, message: string) { super(message); } }

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new ApiError(503, "The Console API is not configured.");
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 10_000);
  try {
    const headers = new Headers(init?.headers);
    headers.set("Accept", "application/json");
    if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
    const response = await fetch(new URL(path, base), {
      ...init,
      headers,
      credentials: init?.credentials ?? "include",
      cache: init?.cache ?? "no-store",
      signal: controller.signal,
    });
    const body: unknown = await response.json().catch(() => undefined);
    if (!response.ok) throw new ApiError(response.status, errorMessage(body) ?? `The API returned ${response.status}.`);
    return body as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error instanceof DOMException && error.name === "AbortError") throw new ApiError(504, "The API request timed out.");
    const detail = error instanceof Error && error.message ? `: ${error.message}` : ".";
    throw new ApiError(0, `The Console API could not be reached${detail}`);
  } finally { window.clearTimeout(timeout); }
}

function errorMessage(value: unknown): string | undefined {
  if (!value || typeof value !== "object") return undefined;
  const error = (value as { error?: unknown }).error;
  return error && typeof error === "object" && typeof (error as { message?: unknown }).message === "string" ? (error as { message: string }).message : undefined;
}
