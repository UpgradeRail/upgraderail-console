import { describe, expect, it } from "vitest";
import type { AnalysisReportRecord, ManifestRecord } from "./analysis";
import { buildCreateFleetKind, buildUpgradeFleetKind, type FleetRecord } from "./proposal-draft";

const h = (c: string) => c.repeat(64);
const fleet: FleetRecord = { id: "c:f", controller_id: "c", fleet_hash: h("1"), tag: "payments", current_wasm_hash: h("a") };
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

describe("buildCreateFleetKind", () => {
  it("derives a deterministic fleet id from the tag and binds candidate hash and manifest hash", async () => {
    const result = await buildCreateFleetKind("payments", report(), manifest);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.kind.tag).toBe("CreateFleet");
    const payload = result.kind.values[0] as { fleet_id: Buffer; initial_wasm_hash: Buffer; manifest_hash: Buffer; tag: string };
    expect(payload.tag).toBe("payments");
    expect(payload.initial_wasm_hash.toString("hex")).toBe(h("b"));
    expect(payload.manifest_hash.toString("hex")).toBe(h("c"));
    expect(payload.fleet_id).toHaveLength(32);
    const again = await buildCreateFleetKind("payments", report(), manifest);
    if (again.ok) {
      const payload2 = again.kind.values[0] as { fleet_id: Buffer };
      expect(payload2.fleet_id.toString("hex")).toBe(payload.fleet_id.toString("hex"));
    }
  });

  it.each([
    ["empty tag", "", report(), manifest, "1-64 characters"],
    ["no report", "payments", null, manifest, "no persisted report"],
    ["no manifest", "payments", report(), null, "no persisted report"],
    ["bad candidate hash", "payments", report({ candidate_wasm_hash: "xyz" }), manifest, "32-byte hex"],
  ])("refuses %s", async (_name, tag, r, m, reason) => {
    const result = await buildCreateFleetKind(tag, r as AnalysisReportRecord | null, m as ManifestRecord | null);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.reason).toContain(reason);
  });
});
