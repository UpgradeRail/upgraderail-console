"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { PreflightView } from "@/components/preflight-view";
import { api, ApiError } from "@/lib/api";
import type { AnalysisJob, AnalysisReportRecord, ManifestRecord } from "@/lib/analysis";

type State =
  | { kind: "loading" }
  | { kind: "not_found" }
  | { kind: "error"; message: string }
  | { kind: "ready"; job: AnalysisJob; report: AnalysisReportRecord | null; manifest: ManifestRecord | null };

async function optional<T>(path: string): Promise<T | null> {
  try {
    return await api<T>(path);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

export default function AnalysisDetailPage({
  analysisId: directId,
  params,
}: {
  analysisId?: string;
  params?: Promise<{ analysisId: string }>;
} = {}) {
  const unwrapped = params ? use(params) : undefined;
  const analysisId = directId ?? unwrapped?.analysisId ?? "";
  const [fetched, setState] = useState<State>({ kind: "loading" });
  const state: State = analysisId ? fetched : { kind: "not_found" };

  useEffect(() => {
    if (!analysisId) return;
    let active = true;
    const id = encodeURIComponent(analysisId);
    void (async () => {
      try {
        const job = await api<AnalysisJob>(`/api/v1/analyses/${id}`);
        const [report, manifest] = await Promise.all([
          optional<AnalysisReportRecord>(`/api/v1/analyses/${id}/report`),
          optional<ManifestRecord>(`/api/v1/analyses/${id}/manifest`),
        ]);
        if (active) setState({ kind: "ready", job, report, manifest });
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 404) setState({ kind: "not_found" });
        else setState({ kind: "error", message: error instanceof Error ? error.message : "The analysis could not be loaded." });
      }
    })();
    return () => {
      active = false;
    };
  }, [analysisId]);

  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">PREFLIGHT</span>
          <h1>Preflight analysis</h1>
          <p>Engine findings and evidence states. Missing evidence is shown as missing, never as a pass.</p>
        </div>
      </header>
      {state.kind === "loading" && <section className="empty-state"><span className="eyebrow">LOADING</span><h2>Reading analysis</h2><p>Querying the Console API for the job, report, and manifest.</p></section>}
      {state.kind === "not_found" && <section className="empty-state"><span className="eyebrow">NOT FOUND</span><h2>Analysis not found</h2><p>No analysis matches: {analysisId || "unspecified"}</p><Link href="/app/analyses" className="text-link">Return to analyses</Link></section>}
      {state.kind === "error" && <section className="empty-state"><span className="eyebrow">ERROR</span><h2>Failed to load analysis</h2><p>{state.message}</p></section>}
      {state.kind === "ready" && <PreflightView job={state.job} report={state.report} manifest={state.manifest} />}
    </section>
  );
}
