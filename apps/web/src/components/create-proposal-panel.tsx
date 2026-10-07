"use client";

import { useEffect, useState } from "react";
import { GovernanceActionPanel } from "./governance-action-panel";
import { api } from "@/lib/api";
import type { AnalysisReportRecord, ManifestRecord } from "@/lib/analysis";
import { buildCreateProposal } from "@/lib/governance-tx";
import { buildUpgradeFleetKind, type FleetRecord } from "@/lib/proposal-draft";

type Fleets = { kind: "loading" } | { kind: "error"; message: string } | { kind: "ready"; value: FleetRecord[] };

export function CreateProposalPanel({ report, manifest }: { report: AnalysisReportRecord; manifest: ManifestRecord }) {
  const [fleets, setFleets] = useState<Fleets>({ kind: "loading" });
  const [fleetId, setFleetId] = useState("");

  useEffect(() => {
    let active = true;
    api<FleetRecord[]>("/api/v1/fleets?limit=100")
      .then((value) => active && setFleets({ kind: "ready", value }))
      .catch((error: unknown) => active && setFleets({ kind: "error", message: error instanceof Error ? error.message : "Fleets could not be loaded." }));
    return () => {
      active = false;
    };
  }, []);

  if (fleets.kind === "loading") return <section className="empty-state"><span className="eyebrow">LOADING</span><h2>Reading indexed fleets</h2><p>Needed to bind the proposal to a fleet.</p></section>;
  if (fleets.kind === "error") return <section className="empty-state"><span className="eyebrow">ERROR</span><h2>Fleets unavailable</h2><p>{fleets.message}</p></section>;
  if (fleets.value.length === 0) return <section className="empty-state"><span className="eyebrow">NO FLEETS</span><h2>No indexed fleets</h2><p>An UpgradeFleet proposal needs an indexed fleet to upgrade.</p></section>;

  const fleet = fleets.value.find((f) => f.id === fleetId);
  const draft = buildUpgradeFleetKind(fleet, report, manifest);

  return (
    <div style={{ display: "grid", gap: "16px" }}>
      <label>
        <span className="eyebrow">TARGET FLEET</span>
        <br />
        <select value={fleetId} onChange={(event) => setFleetId(event.target.value)}>
          <option value="">Select a fleet…</option>
          {fleets.value.map((f) => <option key={f.id} value={f.id}>{f.tag || f.fleet_hash}</option>)}
        </select>
      </label>
      <GovernanceActionPanel
        title="Create upgrade proposal"
        description="Submits an UpgradeFleet proposal that binds the analyzed candidate WASM and its release manifest hash. Approval and timelock still apply before anything executes."
        action="create_proposal"
        details={draft.ok ? draft.summary : []}
        confirmLabel="Submit proposal to Testnet"
        disabledReason={draft.ok ? undefined : draft.reason}
        build={(address) => {
          if (!draft.ok) throw new Error(draft.reason);
          return buildCreateProposal(address, draft.kind);
        }}
      />
    </div>
  );
}
