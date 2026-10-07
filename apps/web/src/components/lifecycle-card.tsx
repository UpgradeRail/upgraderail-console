import type { LiveProposal } from "@/lib/governance-tx";
import { projectionLags, type Lifecycle } from "@/lib/proposal-lifecycle";

const terminalMark: Record<string, string> = { Executed: "✔", Cancelled: "✖", Expired: "⌛", Stale: "⚠" };

export function LifecycleCard({ lifecycle, live, projectedStatus }: { lifecycle: Lifecycle; live: LiveProposal; projectedStatus?: string }) {
  const mark = terminalMark[lifecycle.state] ?? (lifecycle.state === "Ready" ? "▶" : lifecycle.state === "Timelocked" ? "⏳" : "…");
  return (
    <section className="verification-card" style={{ marginTop: 0 }} aria-labelledby="lifecycle-heading">
      <span className="eyebrow">LIVE LIFECYCLE · READ FROM TESTNET AT LEDGER {live.ledger}</span>
      <h2 id="lifecycle-heading">{mark} {lifecycle.label}</h2>
      <p>{lifecycle.detail}</p>
      <dl className="verification-summary">
        <div><dt>Approvals</dt><dd>{live.approvalCount} of {live.threshold} required</dd></div>
        <div><dt>Timelock length</dt><dd>{live.timelockLedgers} ledgers</dd></div>
        <div><dt>Execute after ledger</dt><dd>{live.executeAfterLedger ?? "Not started (threshold not reached)"}</dd></div>
        <div><dt>Timelock remaining</dt><dd>{lifecycle.timelockRemaining === null ? (lifecycle.state === "Ready" ? "Elapsed" : "Not running") : `${lifecycle.timelockRemaining} ledgers`}</dd></div>
        <div><dt>Expires at ledger</dt><dd>{live.expiresLedger}{lifecycle.terminal ? "" : ` (${lifecycle.expiresIn} ledgers remaining)`}</dd></div>
        <div><dt>Indexed status</dt><dd>{projectedStatus ?? "Unknown"}{projectionLags(projectedStatus, lifecycle.state) ? " — the indexed projection is behind the live state" : ""}</dd></div>
      </dl>
    </section>
  );
}
