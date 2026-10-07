"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";

interface Controller {
  id: string;
  network_id: string;
  contract_id: string;
  wasm_hash?: string | null;
  governance_epoch?: number | null;
  controller_version?: number | null;
}

interface Fleet {
  id: string;
  controller_id: string;
  fleet_hash: string;
  tag: string;
  current_wasm_hash?: string | null;
  created_ledger: number;
  created_transaction_hash: string;
}

interface Proposal {
  id: string;
  controller_id: string;
  proposal_id: number;
  proposer: string;
  kind?: string | null;
  governance_epoch: number;
  created_ledger: number;
  expires_ledger: number;
  approval_count: number;
  approved_ledger?: number | null;
  execute_after_ledger?: number | null;
  status: string;
  manifest_hash?: string | null;
  created_transaction_hash: string;
}

interface FleetUpgrade {
  id: string;
  fleet_id: string;
  proposal_id: string;
  old_wasm_hash: string;
  new_wasm_hash: string;
  manifest_hash: string;
  ledger_sequence: number;
  transaction_hash: string;
}

interface AnalysisSummary {
  id: string;
  status: string;
  network: string;
  created_at: string;
}

interface DashboardData {
  controllers: Controller[];
  fleets: Fleet[];
  proposals: Proposal[];
  upgrades: FleetUpgrade[];
  analyses: AnalysisSummary[];
}

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; data: DashboardData };

export default function ConsolePage() {
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const [controllers, fleets, proposals, upgrades, analyses] = await Promise.all([
          api<Controller[]>("/api/v1/controllers"),
          api<Fleet[]>("/api/v1/fleets"),
          api<Proposal[]>("/api/v1/proposals"),
          api<FleetUpgrade[]>("/api/v1/upgrades"),
          api<AnalysisSummary[]>("/api/v1/analyses?limit=5"),
        ]);

        if (active) {
          setState({
            kind: "ready",
            data: { controllers, fleets, proposals, upgrades, analyses },
          });
        }
      } catch (err) {
        if (active) {
          setState({
            kind: "error",
            message:
              err instanceof ApiError
                ? err.message
                : err instanceof Error
                  ? err.message
                  : "The Console API could not be reached.",
          });
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, []);

  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">OVERVIEW</span>
          <h1>Operations</h1>
          <p>Review controller activity, indexed fleets, and release evidence.</p>
        </div>
      </header>

      {state.kind === "loading" && (
        <section className="empty-state">
          <span className="eyebrow">LOADING</span>
          <h2>Reading controller telemetry</h2>
          <p>Waiting for the Console API read models.</p>
        </section>
      )}

      {state.kind === "error" && (
        <section className="empty-state">
          <span className="eyebrow">API UNAVAILABLE</span>
          <h2>Telemetry could not be loaded</h2>
          <p>{state.message}</p>
        </section>
      )}

      {state.kind === "ready" && (
        <div style={{ display: "grid", gap: "32px", marginTop: "32px" }}>
          {/* Controller Summary */}
          <div className="verification-card" style={{ marginTop: 0 }}>
            <span className="eyebrow">UPGRADECONTROLLER</span>
            <h2>Controller Status</h2>
            {state.data.controllers.length === 0 ? (
              <p>No controller instance is currently indexed. Telemetry will appear after events are ingested.</p>
            ) : (
              <>
                {state.data.controllers.map((c) => (
                  <dl className="verification-summary" key={c.id}>
                    <div>
                      <dt>Contract ID</dt>
                      <dd>{c.contract_id}</dd>
                    </div>
                    <div>
                      <dt>Network</dt>
                      <dd>{c.network_id}</dd>
                    </div>
                    <div>
                      <dt>Controller Version</dt>
                      <dd>{c.controller_version !== null && c.controller_version !== undefined ? c.controller_version : "—"}</dd>
                    </div>
                    <div>
                      <dt>Governance Epoch</dt>
                      <dd>{c.governance_epoch !== null && c.governance_epoch !== undefined ? c.governance_epoch : "—"}</dd>
                    </div>
                    <div>
                      <dt>WASM Hash</dt>
                      <dd>{c.wasm_hash ?? "—"}</dd>
                    </div>
                  </dl>
                ))}
              </>
            )}
          </div>

          {/* Fleets */}
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
              <div>
                <span className="eyebrow">FLEETS</span>
                <h2 style={{ margin: "4px 0 0", fontSize: "24px" }}>
                  Indexed Fleets ({state.data.fleets.length})
                </h2>
              </div>
              <Link href="/app/fleets" className="text-link">
                View all fleets →
              </Link>
            </div>
            {state.data.fleets.length === 0 ? (
              <section className="empty-state">
                <span className="eyebrow">EMPTY</span>
                <h2>No indexed fleets</h2>
                <p>No fleets have been created under the indexed controller yet.</p>
              </section>
            ) : (
              <div className="data-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Tag</th>
                      <th>Fleet Hash</th>
                      <th>Current WASM</th>
                      <th>Created Ledger</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.data.fleets.map((fleet) => (
                      <tr key={fleet.id}>
                        <td>{fleet.tag}</td>
                        <td title={fleet.fleet_hash}>{fleet.fleet_hash}</td>
                        <td title={fleet.current_wasm_hash ?? ""}>{fleet.current_wasm_hash ?? "—"}</td>
                        <td>{fleet.created_ledger}</td>
                        <td>
                          <Link href={`/app/fleets/${encodeURIComponent(fleet.id)}`} className="text-link">
                            Detail
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Proposals */}
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
              <div>
                <span className="eyebrow">GOVERNANCE</span>
                <h2 style={{ margin: "4px 0 0", fontSize: "24px" }}>
                  Proposals ({state.data.proposals.length})
                </h2>
              </div>
              <Link href="/app/upgrades" className="text-link">
                View all proposals →
              </Link>
            </div>
            {state.data.proposals.length === 0 ? (
              <section className="empty-state">
                <span className="eyebrow">EMPTY</span>
                <h2>No proposals indexed</h2>
                <p>No governance proposals have been submitted yet.</p>
              </section>
            ) : (
              <div className="data-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Proposal ID</th>
                      <th>Kind</th>
                      <th>Status</th>
                      <th>Approvals</th>
                      <th>Proposer</th>
                      <th>Created Ledger</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.data.proposals.map((proposal) => (
                      <tr key={proposal.id}>
                        <td>#{proposal.proposal_id}</td>
                        <td>{proposal.kind ?? "—"}</td>
                        <td>
                          <span style={{ textTransform: "capitalize" }}>{proposal.status}</span>
                        </td>
                        <td>{proposal.approval_count}</td>
                        <td title={proposal.proposer}>{proposal.proposer}</td>
                        <td>{proposal.created_ledger}</td>
                        <td>
                          <Link href={`/app/upgrades/${encodeURIComponent(proposal.id)}`} className="text-link">
                            Review
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Recent Upgrades */}
          <div>
            <div>
              <span className="eyebrow">UPGRADES</span>
              <h2 style={{ margin: "4px 0 0", fontSize: "24px" }}>
                Recent Upgrade Events ({state.data.upgrades.length})
              </h2>
            </div>
            {state.data.upgrades.length === 0 ? (
              <section className="empty-state">
                <span className="eyebrow">EMPTY</span>
                <h2>No executed upgrades</h2>
                <p>No fleet upgrades have been recorded on-chain yet.</p>
              </section>
            ) : (
              <div className="data-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Ledger</th>
                      <th>Fleet ID</th>
                      <th>New Executable Hash</th>
                      <th>Manifest Hash</th>
                      <th>Transaction</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.data.upgrades.map((upgrade) => (
                      <tr key={upgrade.id}>
                        <td>{upgrade.ledger_sequence}</td>
                        <td title={upgrade.fleet_id}>{upgrade.fleet_id}</td>
                        <td title={upgrade.new_wasm_hash}>{upgrade.new_wasm_hash}</td>
                        <td title={upgrade.manifest_hash}>{upgrade.manifest_hash}</td>
                        <td title={upgrade.transaction_hash}>{upgrade.transaction_hash}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Latest analyses */}
          <div>
            <span className="eyebrow">PREFLIGHT</span>
            <h2 style={{ margin: "4px 0 16px", fontSize: "24px" }}>
              Latest Analysis Jobs ({state.data.analyses.length})
            </h2>
            {state.data.analyses.length === 0 ? (
              <section className="empty-state">
                <span className="eyebrow">EMPTY</span>
                <h2>No analysis jobs</h2>
                <p>No Engine comparison jobs have been created yet.</p>
              </section>
            ) : (
              <div className="data-table-wrap">
                <table>
                  <thead>
                    <tr><th>Analysis</th><th>Status</th><th>Network</th><th>Created</th></tr>
                  </thead>
                  <tbody>
                    {state.data.analyses.map((a) => (
                      <tr key={a.id}>
                        <td><Link className="text-link" href={`/app/analyses/${encodeURIComponent(a.id)}`}>{a.id}</Link></td>
                        <td>{a.status}</td>
                        <td>{a.network}</td>
                        <td>{a.created_at}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  );
}
