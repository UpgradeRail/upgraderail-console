import { describe, expect, it } from "vitest";
import {
  countBySeverity,
  evidenceField,
  parseFindings,
  proposalReadiness,
  runtimeEvidence,
  sortFindings,
  type AnalysisJob,
  type AnalysisReportRecord,
  type ManifestRecord,
} from "./analysis";

const job = (status: string): AnalysisJob => ({
  id: "job1", network: "testnet", status, engine_version: "", error_message: "",
  current_artifact_id: "a", candidate_artifact_id: "b", created_by: null,
  created_at: "2026-01-01T00:00:00Z", started_at: null, finished_at: null,
});
const hex = "ab".repeat(32);
const report = (status: string): AnalysisReportRecord => ({
  status, current_wasm_hash: hex, candidate_wasm_hash: hex, findings: [], runtime_evidence: {},
  report: {}, engine_version: "e", created_at: "2026-01-01T00:00:00Z",
});
const manifest: ManifestRecord = { sha256: hex, bytes: "", created_at: "" };

describe("analysis evidence", () => {
  it("never reports missing or unproven evidence as proven", () => {
    expect(evidenceField(undefined, "storage_compatibility")).toEqual({ label: "NOT RECORDED", proven: false });
    expect(evidenceField({ storage_compatibility: "NOT PROVEN BY STATIC ANALYSIS" }, "storage_compatibility").proven).toBe(false);
    expect(evidenceField({ authorization_behavior: "NOT TESTED" }, "authorization_behavior").proven).toBe(false);
  });

  it("treats runtime evidence as absent unless the report has scenarios", () => {
    expect(runtimeEvidence({ storage_compatibility: "x" })).toEqual({ kind: "not_recorded" });
    expect(runtimeEvidence({ scenarios: [] })).toEqual({ kind: "not_recorded" });
    expect(runtimeEvidence({ scenarios: [{ name: "s" }] }).kind).toBe("recorded");
  });

  it("parses the Engine finding shape and drops malformed rows", () => {
    const findings = parseFindings([
      { code: "SPEC010", severity: "INFO", title: "t", message: "m", evidence: [] },
      { code: "FUNC001", severity: "BLOCKING", title: "removed", message: "m", evidence: [{ kind: "current_function", reference: "f" }, { kind: 1 }] },
      { code: "X", severity: "NOPE", title: "bad" },
      "junk",
    ]);
    expect(findings).toHaveLength(2);
    expect(sortFindings(findings)[0].code).toBe("FUNC001");
    expect(findings.find((f) => f.code === "FUNC001")?.evidence).toHaveLength(1);
    expect(countBySeverity(findings)).toEqual({ BLOCKING: 1, WARNING: 0, INFO: 1 });
    expect(parseFindings(null)).toEqual([]);
  });
});

describe("proposalReadiness", () => {
  it("allows only a ready analysis with report and manifest", () => {
    expect(proposalReadiness(job("ready"), report("READY"), manifest)).toEqual({ ok: true });
    expect(proposalReadiness(job("ready"), report("READY_WITH_WARNINGS"), manifest)).toEqual({ ok: true });
  });
  it.each([
    ["queued", null, null, "queued"],
    ["failed", null, null, "failed"],
    ["blocked", report("BLOCKED"), manifest, "BLOCKED"],
    ["ready", null, manifest, "No Engine report"],
    ["ready", report("READY"), null, "No release manifest"],
  ])("refuses %s analyses", (status, rep, man, reason) => {
    const result = proposalReadiness(job(status), rep, man);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.reason).toContain(reason);
  });
});
