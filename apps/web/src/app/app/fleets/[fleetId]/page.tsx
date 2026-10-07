"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { decodeRouteParam } from "@/lib/route-param";
import { api, ApiError } from "@/lib/api";

interface Fleet {
  id: string;
  controller_id: string;
  fleet_hash: string;
  tag: string;
  current_wasm_hash?: string | null;
  created_ledger: number;
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

type State =
  | { kind: "loading" }
  | { kind: "not_found" }
  | { kind: "error"; message: string }
  | { kind: "ready"; fleet: Fleet; upgrades: FleetUpgrade[] };

export default function FleetDetailPage({
  fleetId: directFleetId,
  params,
}: {
  fleetId?: string;
  params?: Promise<{ fleetId: string }>;
} = {}) {
  const unwrappedParams = params ? use(params) : undefined;
  const fleetId = decodeRouteParam(directFleetId ?? unwrappedParams?.fleetId ?? "");

  const [fetched, setState] = useState<State>({ kind: "loading" });
  const state: State = fleetId ? fetched : { kind: "not_found" };

  useEffect(() => {
    if (!fleetId) return;

    let active = true;

    async function load() {
      try {
        const [fleet, upgrades] = await Promise.all([
          api<Fleet>(`/api/v1/fleets/${encodeURIComponent(fleetId)}`),
          api<FleetUpgrade[]>(`/api/v1/fleets/${encodeURIComponent(fleetId)}/upgrades`).catch(() => [] as FleetUpgrade[]),
        ]);

        if (active) setState({ kind: "ready", fleet, upgrades });
      } catch (err) {
        if (active) {
          if (err instanceof ApiError && err.status === 404) {
            setState({ kind: "not_found" });
          } else {
            setState({
              kind: "error",
              message: err instanceof Error ? err.message : "The fleet could not be loaded.",
            });
          }
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, [fleetId]);

  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">FLEET</span>
          <h1>{state.kind === "ready" ? `Fleet: ${state.fleet.tag}` : "Fleet detail"}</h1>
          <p>Inspect fleet executable references, provenance ledgers, and upgrade history.</p>
        </div>
        {state.kind === "ready" && (
          <div>
            <Link
              href={`/app/fleets/${encodeURIComponent(state.fleet.id)}/analysis`}
              className="button button-accent button-small"
            >
              Preflight Analysis
            </Link>
          </div>
        )}
      </header>

      {state.kind === "loading" && (
        <section className="empty-state">
          <span className="eyebrow">LOADING</span>
          <h2>Reading fleet projection</h2>
          <p>Querying the Console API for fleet identity and upgrade events.</p>
        </section>
      )}

      {state.kind === "not_found" && (
        <section className="empty-state">
          <span className="eyebrow">NOT FOUND</span>
          <h2>Fleet not found</h2>
          <p>No indexed fleet matches the specified identifier: {fleetId || "unspecified"}</p>
          <Link href="/app/fleets" className="text-link">
            Return to Fleets
          </Link>
        </section>
      )}

      {state.kind === "error" && (
        <section className="empty-state">
          <span className="eyebrow">ERROR</span>
          <h2>Failed to load fleet</h2>
          <p>{state.message}</p>
        </section>
      )}

      {state.kind === "ready" && (
        <div style={{ display: "grid", gap: "32px", marginTop: "32px" }}>
          {/* Identity & Current Hash */}
          <div className="verification-card" style={{ marginTop: 0 }}>
            <span className="eyebrow">PROVENANCE & STATE</span>
            <h2>Fleet Identity</h2>
            <dl className="verification-summary">
              <div>
                <dt>Tag / Name</dt>
                <dd>{state.fleet.tag}</dd>
              </div>
              <div>
                <dt>Fleet Hash</dt>
                <dd>{state.fleet.fleet_hash}</dd>
              </div>
              <div>
                <dt>Controller ID</dt>
                <dd>{state.fleet.controller_id}</dd>
              </div>
              <div>
                <dt>Current WASM Hash</dt>
                <dd>{state.fleet.current_wasm_hash ?? "Unknown / Not set"}</dd>
              </div>
              <div>
                <dt>Created Ledger</dt>
                <dd>{state.fleet.created_ledger}</dd>
              </div>
              <div>
                <dt>Created Tx Hash</dt>
                <dd>{state.fleet.created_transaction_hash}</dd>
              </div>
              <div>
                <dt>Last Upgrade</dt>
                <dd>
                  {state.upgrades.length > 0
                    ? `Ledger ${state.upgrades[0].ledger_sequence} (tx ${state.upgrades[0].transaction_hash})`
                    : "None (initial WASM version)"}
                </dd>
              </div>
            </dl>
          </div>

          {/* Scope Limitations */}
          <div className="data-note" style={{ marginTop: 0 }}>
            <strong>Observed Scope Limitations:</strong> UpgradeController governs WASM bytecode upgrades and
            manifest hash integrity only. Instance data migration, contract invocation routing, and live ledger storage
            preservation remain the responsibility of the contract fleet&apos;s internal implementation.
          </div>

          {/* Upgrade History */}
          <div>
            <span className="eyebrow">HISTORY</span>
            <h2 style={{ margin: "4px 0 16px", fontSize: "24px" }}>
              Upgrade History ({state.upgrades.length})
            </h2>
            {state.upgrades.length === 0 ? (
              <section className="empty-state">
                <span className="eyebrow">NO UPGRADES</span>
                <h2>No upgrades recorded</h2>
                <p>This fleet has not undergone an on-chain upgrade yet.</p>
              </section>
            ) : (
              <div className="data-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Ledger</th>
                      <th>Previous WASM</th>
                      <th>New WASM</th>
                      <th>Manifest Hash</th>
                      <th>Proposal ID</th>
                      <th>Transaction</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.upgrades.map((u) => (
                      <tr key={u.id}>
                        <td>{u.ledger_sequence}</td>
                        <td title={u.old_wasm_hash}>{u.old_wasm_hash}</td>
                        <td title={u.new_wasm_hash}>{u.new_wasm_hash}</td>
                        <td title={u.manifest_hash}>{u.manifest_hash}</td>
                        <td>
                          <Link href={`/app/upgrades/${encodeURIComponent(u.proposal_id)}`} className="text-link">
                            {u.proposal_id}
                          </Link>
                        </td>
                        <td title={u.transaction_hash}>{u.transaction_hash}</td>
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
