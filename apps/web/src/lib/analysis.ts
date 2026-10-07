export type Severity = "BLOCKING" | "WARNING" | "INFO";

export type Finding = {
  code: string;
  severity: Severity;
  title: string;
  message: string;
  evidence: { kind: string; reference: string }[];
};

export type AnalysisJob = {
  id: string;
  network: string;
  status: string;
  engine_version: string;
  error_message: string;
  current_artifact_id: string | null;
  candidate_artifact_id: string;
  created_by: string | null;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
};

export type AnalysisReportRecord = {
  status: string;
  current_wasm_hash: string;
  candidate_wasm_hash: string;
  findings: unknown;
  runtime_evidence: unknown;
  report: unknown;
  engine_version: string;
  created_at: string;
};

export type ManifestRecord = { sha256: string; bytes: string; created_at: string };

export type EvidenceState = {
  label: string;
  /** True only when the Engine positively proved the property. */
  proven: boolean;
};

export type RuntimeEvidence =
  | { kind: "not_recorded" }
  | { kind: "recorded"; raw: unknown };

const severities: Severity[] = ["BLOCKING", "WARNING", "INFO"];

export function parseFindings(value: unknown): Finding[] {
  if (!Array.isArray(value)) return [];
  const findings: Finding[] = [];
  for (const item of value) {
    if (!item || typeof item !== "object") continue;
    const row = item as Record<string, unknown>;
    if (typeof row.code !== "string" || typeof row.title !== "string") continue;
    const severity = severities.find((s) => s === row.severity);
    if (!severity) continue;
    const evidence = Array.isArray(row.evidence)
      ? row.evidence.flatMap((entry) => {
          const e = entry as Record<string, unknown> | null;
          return e && typeof e.kind === "string" && typeof e.reference === "string"
            ? [{ kind: e.kind, reference: e.reference }]
            : [];
        })
      : [];
    findings.push({
      code: row.code,
      severity,
      title: row.title,
      message: typeof row.message === "string" ? row.message : "",
      evidence,
    });
  }
  return findings;
}

export function countBySeverity(findings: Finding[]): Record<Severity, number> {
  const counts: Record<Severity, number> = { BLOCKING: 0, WARNING: 0, INFO: 0 };
  for (const finding of findings) counts[finding.severity] += 1;
  return counts;
}

export function sortFindings(findings: Finding[]): Finding[] {
  return [...findings].sort(
    (a, b) => severities.indexOf(a.severity) - severities.indexOf(b.severity) || a.code.localeCompare(b.code)
  );
}

/**
 * Reads an evidence field from the persisted Engine report. A missing or
 * unrecognized value is reported as NOT RECORDED, never as a pass.
 */
export function evidenceField(report: unknown, field: "storage_compatibility" | "authorization_behavior"): EvidenceState {
  const value = report && typeof report === "object" ? (report as Record<string, unknown>)[field] : undefined;
  if (typeof value !== "string" || value.trim() === "") return { label: "NOT RECORDED", proven: false };
  const proven = !/NOT PROVEN|NOT TESTED|NOT CONFIGURED|FAILED/i.test(value);
  return { label: value, proven };
}

/**
 * Runtime simulation evidence exists only when the Engine report carries
 * scenario results. The worker's `runtime_evidence` column holds just the
 * storage/authorization strings, so it is deliberately not consulted here.
 */
export function runtimeEvidence(report: unknown): RuntimeEvidence {
  if (!report || typeof report !== "object") return { kind: "not_recorded" };
  const row = report as Record<string, unknown>;
  const value = row.scenarios ?? row.simulation ?? row.runtime;
  const empty =
    value === null ||
    value === undefined ||
    (Array.isArray(value) && value.length === 0) ||
    (typeof value === "object" && !Array.isArray(value) && Object.keys(value as object).length === 0);
  return empty ? { kind: "not_recorded" } : { kind: "recorded", raw: value };
}

export const STORAGE_LIMITATION =
  "Soroban contract specs do not expose private storage keys or state invariants. Static analysis cannot prove storage compatibility; run an explicit migration scenario.";

export type ProposalReadiness = { ok: true } | { ok: false; reason: string };

/** Whether an analysis may be used to draft an UpgradeFleet proposal. */
export function proposalReadiness(
  job: AnalysisJob,
  report: AnalysisReportRecord | null,
  manifest: ManifestRecord | null
): ProposalReadiness {
  if (job.status !== "ready" && job.status !== "blocked") {
    return { ok: false, reason: `Analysis is ${job.status}; no Engine result exists yet.` };
  }
  if (job.status === "blocked" || report?.status === "BLOCKED") {
    return { ok: false, reason: "The Engine reports BLOCKED findings. Resolve them and run a new analysis." };
  }
  if (!report) return { ok: false, reason: "No Engine report is persisted for this analysis." };
  if (!manifest) return { ok: false, reason: "No release manifest is persisted for this analysis." };
  if (!/^[0-9a-f]{64}$/.test(manifest.sha256)) return { ok: false, reason: "The manifest hash is not a 32-byte hex value." };
  if (!/^[0-9a-f]{64}$/.test(report.candidate_wasm_hash)) return { ok: false, reason: "The candidate WASM hash is not a 32-byte hex value." };
  return { ok: true };
}
