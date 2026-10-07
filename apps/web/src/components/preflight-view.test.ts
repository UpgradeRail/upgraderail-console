import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { PreflightView } from "./preflight-view";
import type { AnalysisJob, AnalysisReportRecord } from "@/lib/analysis";

const job = (status: string, error = ""): AnalysisJob => ({
  id: "job1", network: "testnet", status, engine_version: "", error_message: error,
  current_artifact_id: "a", candidate_artifact_id: "b", created_by: null,
  created_at: "", started_at: null, finished_at: null,
});
const hex = "cd".repeat(32);
const blocked: AnalysisReportRecord = {
  status: "BLOCKED", current_wasm_hash: hex, candidate_wasm_hash: "ef".repeat(32),
  engine_version: "upgraderail 0.1", created_at: "", runtime_evidence: {},
  findings: [
    { code: "SPEC010", severity: "INFO", title: "Storage compatibility is not statically proven", message: "m", evidence: [] },
    { code: "FUNC001", severity: "BLOCKING", title: "Public function removed", message: "gone", evidence: [{ kind: "current_function", reference: "get_value" }] },
  ],
  report: { status: "BLOCKED", storage_compatibility: "NOT PROVEN BY STATIC ANALYSIS", authorization_behavior: "NOT TESTED" },
};
const render = (j: AnalysisJob, r: AnalysisReportRecord | null, m: { sha256: string; bytes: string; created_at: string } | null = null) =>
  renderToStaticMarkup(React.createElement(PreflightView, { job: j, report: r, manifest: m }));

describe("PreflightView", () => {
  it("shows blocking findings first, exact evidence states, and no score", () => {
    const html = render(job("blocked"), blocked, { sha256: "12".repeat(32), bytes: "", created_at: "" });
    expect(html).toContain("Engine status: BLOCKED");
    expect(html.indexOf("Public function removed")).toBeLessThan(html.indexOf("Storage compatibility is not statically proven</strong>"));
    expect(html).toContain("✖ BLOCKING");
    expect(html).toContain("NOT PROVEN BY STATIC ANALYSIS");
    expect(html).toContain("NOT TESTED");
    expect(html).toContain("NOT CONFIGURED — no runtime simulation");
    expect(html).toContain("1 blocking · 0 warning · 1 info");
    expect(html).toContain("12".repeat(32));
    expect(html).not.toMatch(/score|PASS/i);
  });

  it("does not present a queued job with no report as a result", () => {
    const html = render(job("queued"), null);
    expect(html).toContain("has not produced an Engine report yet");
    expect(html).toContain("Absence of a report is not a passing result");
    expect(html).toContain("NOT RECORDED");
  });

  it("surfaces job failure text", () => {
    expect(render(job("failed", "engine timed out"), null)).toContain("engine timed out");
  });
});
