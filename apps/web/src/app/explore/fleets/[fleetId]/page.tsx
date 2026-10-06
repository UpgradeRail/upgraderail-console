"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { SiteHeader } from "@/components/site-header";
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

export default function ExploreFleetPage({
  params,
}: {
  params?: Promise<{ fleetId: string }>;
} = {}) {
  const unwrappedParams = params ? use(params) : { fleetId: "" };
  const fleetId = unwrappedParams.fleetId;

  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    if (!fleetId) {
      setState({ kind: "not_found" });
      return;
    }

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
    <>
      <SiteHeader />
      <main className="explore-layout">
        <header>
          <span className="eyebrow">PUBLIC FLEET EXPLORER</span>
          <h1>{state.kind === "ready" ? `Fleet: ${state.fleet.tag}` : "Fleet detail"}</h1>
          <p className="data-note" style={{ marginTop: "12px" }}>
            This explorer reads only from the Console indexer. It does not present sample fleets or simulated activity.
          </p>
        </header>

        {state.kind === "loading" && (
          <section className="empty-state">
            <span className="eyebrow">LOADING</span>
            <h2>Reading fleet projection</h2>
            <p>Querying the indexer for fleet identity and upgrade events.</p>
          </section>
        )}

        {state.kind === "not_found" && (
          <section className="empty-state">
            <span className="eyebrow">NOT FOUND</span>
            <h2>Fleet not found</h2>
            <p>No indexed fleet matches the specified identifier: {fleetId || "unspecified"}</p>
            <Link href="/explore" className="text-link">
              Return to Explorer
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
            <div className="verification-card" style={{ marginTop: 0 }}>
              <span className="eyebrow">PROVENANCE</span>
              <h2>Fleet Identity</h2>
              <div className="verification-summary">
                <div>
                  <dt>Tag</dt>
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
                  <dt>Current WASM</dt>
                  <dd>{state.fleet.current_wasm_hash ?? "—"}</dd>
                </div>
                <div>
                  <dt>Created Ledger</dt>
                  <dd>{state.fleet.created_ledger}</dd>
                </div>
                <div>
                  <dt>Created Transaction</dt>
                  <dd>{state.fleet.created_transaction_hash}</dd>
                </div>
              </div>
            </div>

            <div>
              <span className="eyebrow">HISTORY</span>
              <h2 style={{ margin: "4px 0 16px", fontSize: "24px" }}>
                Upgrade History ({state.upgrades.length})
              </h2>
              {state.upgrades.length === 0 ? (
                <section className="empty-state">
                  <span className="eyebrow">NO UPGRADES</span>
                  <h2>No upgrades recorded</h2>
                  <p>This fleet has not been upgraded on-chain yet.</p>
                </section>
              ) : (
                <div className="data-table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Ledger</th>
                        <th>Old WASM</th>
                        <th>New WASM</th>
                        <th>Manifest Hash</th>
                        <th>Tx Hash</th>
                      </tr>
                    </thead>
                    <tbody>
                      {state.upgrades.map((u) => (
                        <tr key={u.id}>
                          <td>{u.ledger_sequence}</td>
                          <td title={u.old_wasm_hash}>{u.old_wasm_hash}</td>
                          <td title={u.new_wasm_hash}>{u.new_wasm_hash}</td>
                          <td title={u.manifest_hash}>{u.manifest_hash}</td>
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
      </main>
    </>
  );
}
