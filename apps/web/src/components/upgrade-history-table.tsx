import Link from "next/link";

export type UpgradeRecord = {
  id: string;
  fleet_id: string;
  proposal_id: string;
  old_wasm_hash: string;
  new_wasm_hash: string;
  manifest_hash: string;
  ledger_sequence: number;
  transaction_hash: string;
};

/**
 * The API records ledger and transaction but no block timestamp or analysis
 * link for an upgrade, so those columns say so instead of inventing values.
 */
export function UpgradeHistoryTable({ upgrades }: { upgrades: UpgradeRecord[] }) {
  if (upgrades.length === 0) {
    return <section className="empty-state"><span className="eyebrow">NO UPGRADES</span><h2>No executed upgrades are indexed</h2><p>Upgrades appear here after the indexer observes an executed UpgradeFleet proposal.</p></section>;
  }
  return (
    <div className="data-table-wrap">
      <table>
        <thead>
          <tr><th>Fleet</th><th>Old hash</th><th>New hash</th><th>Proposal</th><th>Manifest hash</th><th>Ledger</th><th>Transaction</th><th>Timestamp</th><th>Report</th></tr>
        </thead>
        <tbody>
          {upgrades.map((u) => (
            <tr key={u.id}>
              <td><Link className="text-link" href={`/app/fleets/${encodeURIComponent(u.fleet_id)}`}>{u.fleet_id}</Link></td>
              <td>{u.old_wasm_hash}</td>
              <td>{u.new_wasm_hash}</td>
              <td><Link className="text-link" href={`/app/upgrades/${encodeURIComponent(u.proposal_id)}`}>{u.proposal_id}</Link></td>
              <td>{u.manifest_hash}</td>
              <td>{u.ledger_sequence}</td>
              <td>{u.transaction_hash}</td>
              <td>Not recorded</td>
              <td>No analysis linked</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
