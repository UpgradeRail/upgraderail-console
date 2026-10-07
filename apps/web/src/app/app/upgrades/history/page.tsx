"use client";

import { useEffect, useState } from "react";
import { UpgradeHistoryTable, type UpgradeRecord } from "@/components/upgrade-history-table";
import { api, ApiError } from "@/lib/api";

type State = { kind: "loading" } | { kind: "error"; message: string } | { kind: "ready"; upgrades: UpgradeRecord[] };

export default function UpgradeHistoryPage() {
  const [state, setState] = useState<State>({ kind: "loading" });
  useEffect(() => {
    let active = true;
    api<UpgradeRecord[]>("/api/v1/upgrades?limit=100")
      .then((upgrades) => active && setState({ kind: "ready", upgrades }))
      .catch((error: unknown) => active && setState({ kind: "error", message: error instanceof ApiError ? error.message : "Upgrade history could not be loaded." }));
    return () => {
      active = false;
    };
  }, []);

  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">HISTORY</span>
          <h1>Upgrade history</h1>
          <p>Executed fleet upgrades observed by the indexer, newest first.</p>
        </div>
      </header>
      {state.kind === "loading" && <section className="empty-state"><span className="eyebrow">LOADING</span><h2>Reading upgrade history</h2><p>Waiting for the Console API.</p></section>}
      {state.kind === "error" && <section className="empty-state"><span className="eyebrow">API UNAVAILABLE</span><h2>Upgrade history could not be loaded</h2><p>{state.message}</p></section>}
      {state.kind === "ready" && <UpgradeHistoryTable upgrades={state.upgrades} />}
    </section>
  );
}
