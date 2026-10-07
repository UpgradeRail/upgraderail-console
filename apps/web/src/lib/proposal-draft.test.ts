import { describe, expect, it } from "vitest";
import type { AnalysisReportRecord, ManifestRecord } from "./analysis";
import { buildUpgradeFleetKind, type FleetRecord } from "./proposal-draft";

const h = (c: string) => c.repeat(64);
const fleet: FleetRecord = { id: "c:f", fleet_hash: h("1"), tag: "payments", current_wasm_hash: h("a") };
const report = (over: Partial<AnalysisReportRecord> = {}): AnalysisReportRecord => ({
  status: "READY", current_wasm_hash: h("a"), candidate_wasm_hash: h("b"), findings: [], runtime_evidence: {},
  report: {}, engine_version: "e", created_at: "", ...over,
});
const manifest: ManifestRecord = { sha256: h("c"), bytes: "", created_at: "" };

describe("buildUpgradeFleetKind", () => {
  it("binds fleet, expected hash, candidate hash, and manifest hash", () => {
    const result = buildUpgradeFleetKind(fleet, report(), manifest);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    const payload = result.kind.values[0] as { fleet_id: Buffer; expected_wasm_hash: Buffer; new_wasm_hash: Buffer; manifest_hash: Buffer };
    expect(result.kind.tag).toBe("UpgradeFleet");
    expect(payload.fleet_id.toString("hex")).toBe(h("1"));
    expect(payload.expected_wasm_hash.toString("hex")).toBe(h("a"));
    expect(payload.new_wasm_hash.toString("hex")).toBe(h("b"));
    expect(payload.manifest_hash.toString("hex")).toBe(h("c"));
  });

  it.each([
    ["no fleet", undefined, report(), manifest, "Select an indexed fleet"],
    ["no report", fleet, null, manifest, "no persisted report"],
    ["no manifest", fleet, report(), null, "no persisted report"],
    ["unknown fleet wasm", { ...fleet, current_wasm_hash: null }, report(), manifest, "no known current WASM"],
    ["stale analysis", { ...fleet, current_wasm_hash: h("d") }, report(), manifest, "Stale analysis"],
    ["same candidate", fleet, report({ candidate_wasm_hash: h("a") }), manifest, "identical"],
    ["non-hex fleet id", { ...fleet, fleet_hash: "nothex" }, report(), manifest, "32-byte hex"],
    ["bad candidate hash", fleet, report({ candidate_wasm_hash: "xyz" }), manifest, "32-byte hex"],
  ])("refuses %s", (_name, f, r, m, reason) => {
    const result = buildUpgradeFleetKind(f as FleetRecord | undefined, r as AnalysisReportRecord | null, m as ManifestRecord | null);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.reason).toContain(reason);
  });
});
