import { api } from "./api";

export type ArtifactRecord = {
  id: string;
  sha256: string;
  size_bytes: number;
  content_type: string;
  uploader: string | null;
  created_at: string;
};

/**
 * Uploads a WASM file to the Console API and returns the server-recorded
 * artifact: a stable ID plus the server-computed SHA-256. The server never
 * trusts a client-supplied hash, so none is sent here; the UI shows
 * whatever the API returns, not a hash computed in the browser.
 */
export async function uploadArtifact(file: File): Promise<ArtifactRecord> {
  const form = new FormData();
  form.append("file", file);
  // Uploads can take longer than a typical read; 30s gives real artifacts
  // over a slow connection room without hanging indefinitely on a stalled one.
  return api<ArtifactRecord>("/api/v1/artifacts", { method: "POST", body: form }, 30_000);
}
