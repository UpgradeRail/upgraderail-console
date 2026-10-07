import type { AnalysisReportRecord, ManifestRecord } from "./analysis";
import type { ProposalKindInput } from "./governance-tx";

export type FleetRecord = {
  id: string;
  controller_id: string;
  fleet_hash: string;
  tag: string;
  current_wasm_hash: string | null;
};

export type DraftResult =
  | { ok: true; kind: ProposalKindInput; summary: ReadonlyArray<readonly [string, string]> }
  | { ok: false; reason: string };

const HEX32 = /^[0-9a-f]{64}$/;

function bytes(hex: string): Buffer {
  return Buffer.from(hex, "hex");
}

/**
 * Builds the UpgradeFleet proposal kind from an analysis. Every hash must be
 * known and consistent; nothing is guessed. The fleet's projected hash must
 * equal the analysis' current hash, otherwise the contract would reject the
 * proposal as stale.
 */
export function buildUpgradeFleetKind(
  fleet: FleetRecord | undefined,
  report: AnalysisReportRecord | null,
  manifest: ManifestRecord | null
): DraftResult {
  if (!fleet) return { ok: false, reason: "Select an indexed fleet." };
  if (!report || !manifest) return { ok: false, reason: "The analysis has no persisted report and manifest." };
  if (!HEX32.test(fleet.fleet_hash)) return { ok: false, reason: "The fleet identifier is not a 32-byte hex value." };
  if (!fleet.current_wasm_hash) return { ok: false, reason: "The indexed fleet has no known current WASM hash yet." };
  if (!HEX32.test(fleet.current_wasm_hash)) return { ok: false, reason: "The fleet's current WASM hash is not a 32-byte hex value." };
  if (fleet.current_wasm_hash !== report.current_wasm_hash) {
    return {
      ok: false,
      reason: "Stale analysis: the fleet's indexed WASM hash differs from the analysis' current WASM. Run a new analysis against the fleet's current code.",
    };
  }
  if (!HEX32.test(report.candidate_wasm_hash) || !HEX32.test(manifest.sha256)) {
    return { ok: false, reason: "The candidate WASM or manifest hash is not a 32-byte hex value." };
  }
  if (report.candidate_wasm_hash === report.current_wasm_hash) {
    return { ok: false, reason: "The candidate WASM is identical to the current WASM." };
  }
  return {
    ok: true,
    kind: {
      tag: "UpgradeFleet",
      values: [
        {
          fleet_id: bytes(fleet.fleet_hash),
          expected_wasm_hash: bytes(fleet.current_wasm_hash),
          new_wasm_hash: bytes(report.candidate_wasm_hash),
          manifest_hash: bytes(manifest.sha256),
        },
      ],
    },
    summary: [
      ["Proposal kind", "UpgradeFleet"],
      ["Fleet", `${fleet.tag} (${fleet.fleet_hash})`],
      ["Expected current WASM", fleet.current_wasm_hash],
      ["New WASM", report.candidate_wasm_hash],
      ["Manifest hash", manifest.sha256],
    ],
  };
}

/**
 * Builds the CreateFleet proposal kind from an analysis, for controllers
 * with no indexed fleet yet. The fleet identifier is derived from the tag
 * via SHA-256 so it is deterministic and collision-resistant, matching how
 * the contract scopes fleets by a caller-chosen 32-byte id.
 */
export async function buildCreateFleetKind(
  tag: string,
  report: AnalysisReportRecord | null,
  manifest: ManifestRecord | null
): Promise<DraftResult> {
  const trimmed = tag.trim();
  if (trimmed.length === 0 || trimmed.length > 64) return { ok: false, reason: "The fleet tag must be 1-64 characters." };
  if (!report || !manifest) return { ok: false, reason: "The analysis has no persisted report and manifest." };
  if (!HEX32.test(report.candidate_wasm_hash) || !HEX32.test(manifest.sha256)) {
    return { ok: false, reason: "The candidate WASM or manifest hash is not a 32-byte hex value." };
  }
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(`upgraderail-fleet:${trimmed}`));
  const fleetId = Buffer.from(digest);
  return {
    ok: true,
    kind: {
      tag: "CreateFleet",
      values: [
        {
          fleet_id: fleetId,
          initial_wasm_hash: bytes(report.candidate_wasm_hash),
          manifest_hash: bytes(manifest.sha256),
          tag: trimmed,
        },
      ],
    },
    summary: [
      ["Proposal kind", "CreateFleet"],
      ["Tag", trimmed],
      ["Fleet id (sha256 of tag)", fleetId.toString("hex")],
      ["Initial WASM", report.candidate_wasm_hash],
      ["Manifest hash", manifest.sha256],
    ],
  };
}
