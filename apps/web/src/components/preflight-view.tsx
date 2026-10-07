import {
  STORAGE_LIMITATION,
  countBySeverity,
  evidenceField,
  parseFindings,
  runtimeEvidence,
  sortFindings,
  type AnalysisJob,
  type AnalysisReportRecord,
  type ManifestRecord,
  type Severity,
} from "@/lib/analysis";

const severityMark: Record<Severity, string> = { BLOCKING: "✖ BLOCKING", WARNING: "▲ WARNING", INFO: "ℹ INFO" };

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><dt>{label}</dt><dd>{children}</dd></div>;
}

export function PreflightView({
  job,
  report,
  manifest,
  children,
}: {
  job: AnalysisJob;
  report: AnalysisReportRecord | null;
  manifest: ManifestRecord | null;
  children?: React.ReactNode;
}) {
  const findings = sortFindings(parseFindings(report?.findings));
  const counts = countBySeverity(findings);
  const storage = evidenceField(report?.report, "storage_compatibility");
  const authorization = evidenceField(report?.report, "authorization_behavior");
  const runtime = runtimeEvidence(report?.report);

  return (
    <div style={{ display: "grid", gap: "32px", marginTop: "32px" }}>
      <section className="verification-card" style={{ marginTop: 0 }}>
        <span className="eyebrow">ENGINE RESULT</span>
        <h2>{report ? `Engine status: ${report.status}` : `Job status: ${job.status}`}</h2>
        {job.status === "failed" && <p role="alert">Analysis failed: {job.error_message || "no error message recorded"}</p>}
        {(job.status === "queued" || job.status === "running") && <p role="status">The worker has not produced an Engine report yet. No result is shown until it does.</p>}
        <dl className="verification-summary">
          <Row label="Analysis ID">{job.id}</Row>
          <Row label="Network">{job.network}</Row>
          <Row label="Engine version">{report?.engine_version || job.engine_version || "Not recorded"}</Row>
          <Row label="Current WASM">{report?.current_wasm_hash ?? "Not available"}</Row>
          <Row label="Candidate WASM">{report?.candidate_wasm_hash ?? "Not available"}</Row>
          <Row label="Manifest hash">{manifest?.sha256 ?? "Not available"}</Row>
          <Row label="Report">
            {report ? <a className="text-link" href={`${process.env.NEXT_PUBLIC_API_BASE_URL ?? ""}/api/v1/analyses/${encodeURIComponent(job.id)}/report`}>Raw Engine report (JSON)</a> : "Not available"}
          </Row>
          <Row label="Manifest file">
            {manifest ? <a className="text-link" href={`${process.env.NEXT_PUBLIC_API_BASE_URL ?? ""}/api/v1/analyses/${encodeURIComponent(job.id)}/manifest`}>Manifest record (JSON, base64 bytes)</a> : "Not available"}
          </Row>
        </dl>
      </section>

      <section className="verification-card" style={{ marginTop: 0 }}>
        <span className="eyebrow">EVIDENCE STATES</span>
        <h2>What is and is not proven</h2>
        <dl className="verification-summary">
          <Row label="Storage compatibility">{storage.proven ? "✔ " : "⚠ "}{storage.label}</Row>
          <Row label="Authorization behavior">{authorization.proven ? "✔ " : "⚠ "}{authorization.label}</Row>
          <Row label="Runtime simulation">
            {runtime.kind === "recorded" ? <pre style={{ margin: 0, whiteSpace: "pre-wrap" }}>{JSON.stringify(runtime.raw, null, 2)}</pre> : "⚠ NOT CONFIGURED — no runtime simulation scenario ran or was recorded"}
          </Row>
        </dl>
        <p className="data-note">{STORAGE_LIMITATION}</p>
      </section>

      <section>
        <span className="eyebrow">FINDINGS</span>
        <h2 style={{ margin: "4px 0 8px", fontSize: "24px" }}>
          {findings.length} finding{findings.length === 1 ? "" : "s"}
        </h2>
        <p>{counts.BLOCKING} blocking · {counts.WARNING} warning · {counts.INFO} info</p>
        {report === null ? (
          <section className="empty-state"><span className="eyebrow">NO REPORT</span><h2>No findings recorded</h2><p>Absence of a report is not a passing result.</p></section>
        ) : findings.length === 0 ? (
          <section className="empty-state"><span className="eyebrow">NO FINDINGS</span><h2>The Engine emitted no findings</h2><p>Evidence states above still apply: no finding is not proof of storage or authorization safety.</p></section>
        ) : (
          <div className="data-table-wrap">
            <table>
              <thead><tr><th>Severity</th><th>Code</th><th>Finding</th><th>Evidence</th></tr></thead>
              <tbody>
                {findings.map((f, index) => (
                  <tr key={`${f.code}-${index}`}>
                    <td>{severityMark[f.severity]}</td>
                    <td>{f.code}</td>
                    <td><strong>{f.title}</strong><br />{f.message}</td>
                    <td>{f.evidence.length === 0 ? "None recorded" : f.evidence.map((e) => `${e.kind}: ${e.reference}`).join("; ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      {children}
    </div>
  );
}
