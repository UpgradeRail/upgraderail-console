"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { ArtifactUploadPanel } from "./artifact-upload-panel";
import { api, ApiError } from "@/lib/api";
import type { ArtifactRecord } from "@/lib/artifacts";
import type { AnalysisJob } from "@/lib/analysis";

type Session = { address: string; network: string };
type Job =
  | { kind: "idle" }
  | { kind: "creating" }
  | { kind: "error"; message: string }
  | { kind: "tracking"; id: string; job: AnalysisJob };

const POLL_INTERVAL_MS = 2000;
const TERMINAL_STATUSES = new Set(["ready", "blocked", "failed", "cancelled"]);

/**
 * Uploads a current and a candidate WASM artifact, then creates and tracks
 * an analysis job from the uploaded artifact IDs. Status is shown exactly
 * as the API reports it (queued/running/ready/blocked/failed/cancelled);
 * nothing here fabricates progress or a result ahead of the worker
 * actually finishing the job.
 */
export function CreateAnalysisFromUploads({ session }: { session: Session }) {
  const [current, setCurrent] = useState<ArtifactRecord | null>(null);
  const [candidate, setCandidate] = useState<ArtifactRecord | null>(null);
  const [job, setJob] = useState<Job>({ kind: "idle" });
  const pollTimer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (pollTimer.current) window.clearTimeout(pollTimer.current);
    };
  }, []);

  function pollJob(id: string) {
    const tick = async () => {
      try {
        const value = await api<AnalysisJob>(`/api/v1/analyses/${encodeURIComponent(id)}`);
        setJob({ kind: "tracking", id, job: value });
        if (!TERMINAL_STATUSES.has(value.status)) {
          pollTimer.current = window.setTimeout(tick, POLL_INTERVAL_MS);
        }
      } catch (error) {
        setJob({ kind: "error", message: error instanceof Error ? error.message : "Analysis status could not be read." });
      }
    };
    void tick();
  }

  async function createAnalysis() {
    if (!current || !candidate) return;
    setJob({ kind: "creating" });
    try {
      const created = await api<{ id: string; status: string }>("/api/v1/analyses", {
        method: "POST",
        body: JSON.stringify({ network: session.network, current_artifact_id: current.id, candidate_artifact_id: candidate.id }),
      });
      pollJob(created.id);
    } catch (error) {
      setJob({ kind: "error", message: error instanceof ApiError ? error.message : "The analysis job could not be created." });
    }
  }

  const canCreate = Boolean(current && candidate) && job.kind !== "creating" && job.kind !== "tracking";

  return (
    <div style={{ display: "grid", gap: "24px" }}>
      <div style={{ display: "grid", gap: "16px", gridTemplateColumns: "1fr 1fr" }}>
        <ArtifactUploadPanel label="CURRENT WASM" description="The WASM currently live for the fleet being upgraded." onUploaded={setCurrent} />
        <ArtifactUploadPanel label="CANDIDATE WASM" description="The WASM proposed to replace it." onUploaded={setCandidate} />
      </div>
      <div>
        <button type="button" className="button button-accent" disabled={!canCreate} onClick={() => void createAnalysis()}>
          Create analysis job
        </button>
      </div>
      {job.kind === "creating" && (
        <section className="empty-state">
          <span className="eyebrow">SUBMITTING</span>
          <h2>Creating analysis job</h2>
          <p>Queuing the job for the worker.</p>
        </section>
      )}
      {job.kind === "error" && (
        <section className="empty-state">
          <span className="eyebrow">ERROR</span>
          <h2>Analysis job unavailable</h2>
          <p>{job.message}</p>
        </section>
      )}
      {job.kind === "tracking" && <JobStatus id={job.id} job={job.job} />}
    </div>
  );
}

export function JobStatus({ id, job }: { id: string; job: AnalysisJob }) {
  const statusLabel = job.status.toUpperCase();
  return (
    <section className="empty-state">
      <span className="eyebrow">JOB {id}</span>
      <h2>Status: {statusLabel}</h2>
      {job.status === "queued" && <p>Waiting for a worker to claim the job.</p>}
      {job.status === "running" && <p>The worker is running the Engine against the uploaded artifacts.</p>}
      {job.status === "ready" && (
        <p>
          The Engine finished with no blocking findings. <Link className="text-link" href={`/app/analyses/${encodeURIComponent(id)}`}>View the preflight analysis</Link>.
        </p>
      )}
      {job.status === "blocked" && (
        <p>
          The Engine reported blocking findings. <Link className="text-link" href={`/app/analyses/${encodeURIComponent(id)}`}>View the preflight analysis</Link> for details.
        </p>
      )}
      {job.status === "failed" && <p>The job failed: {job.error_message || "No error message was recorded."}</p>}
      {job.status === "cancelled" && <p>The job was cancelled before it produced a result.</p>}
    </section>
  );
}
