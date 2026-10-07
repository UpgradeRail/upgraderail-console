"use client";

import { useId, useState } from "react";
import { ApiError } from "@/lib/api";
import { uploadArtifact, type ArtifactRecord } from "@/lib/artifacts";

type Status =
  | { kind: "empty" }
  | { kind: "uploading"; fileName: string }
  | { kind: "uploaded"; fileName: string; artifact: ArtifactRecord }
  | { kind: "error"; fileName: string; message: string };

/**
 * One WASM upload slot (current or candidate). Uploads immediately on file
 * selection, shows the server-computed hash once stored, and reports the
 * uploaded artifact (or null, on error) to the parent so it can gate
 * analysis-job creation on both slots being filled.
 */
export function ArtifactUploadPanel({
  label,
  description,
  onUploaded,
}: {
  label: string;
  description: string;
  onUploaded: (artifact: ArtifactRecord | null) => void;
}) {
  const [status, setStatus] = useState<Status>({ kind: "empty" });
  const inputId = useId();

  async function handleFile(file: File | undefined) {
    if (!file) return;
    setStatus({ kind: "uploading", fileName: file.name });
    onUploaded(null);
    try {
      const artifact = await uploadArtifact(file);
      setStatus({ kind: "uploaded", fileName: file.name, artifact });
      onUploaded(artifact);
    } catch (error) {
      const message = error instanceof ApiError ? error.message : error instanceof Error ? error.message : "The upload failed.";
      setStatus({ kind: "error", fileName: file.name, message });
      onUploaded(null);
    }
  }

  return (
    <div style={{ border: "1px solid var(--line)", padding: "20px", display: "grid", gap: "10px" }}>
      <div>
        <span className="eyebrow">{label}</span>
        <p style={{ margin: "4px 0 0", color: "var(--muted)" }}>{description}</p>
      </div>
      <label htmlFor={inputId} style={{ fontWeight: 650, fontSize: "14px" }}>
        {status.kind === "uploaded" ? "Replace WASM file" : "Choose WASM file"}
      </label>
      <input
        id={inputId}
        type="file"
        accept=".wasm,application/wasm"
        disabled={status.kind === "uploading"}
        onChange={(event) => {
          void handleFile(event.target.files?.[0]);
        }}
      />
      {status.kind === "uploading" && <p>Uploading {status.fileName}…</p>}
      {status.kind === "uploaded" && (
        <dl style={{ margin: 0, fontSize: "13px", color: "var(--muted)", display: "grid", gap: "2px" }}>
          <div>
            <dt style={{ display: "inline" }}>File: </dt>
            <dd style={{ display: "inline", margin: 0 }}>{status.fileName}</dd>
          </div>
          <div>
            <dt style={{ display: "inline" }}>SHA-256 (server-computed): </dt>
            <dd style={{ display: "inline", margin: 0, fontFamily: "var(--font-mono)" }}>{status.artifact.sha256}</dd>
          </div>
          <div>
            <dt style={{ display: "inline" }}>Size: </dt>
            <dd style={{ display: "inline", margin: 0 }}>{status.artifact.size_bytes.toLocaleString()} bytes</dd>
          </div>
        </dl>
      )}
      {status.kind === "error" && (
        <p role="alert" style={{ color: "var(--danger)", margin: 0 }}>
          {status.message}
        </p>
      )}
    </div>
  );
}
