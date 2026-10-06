"use client";

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";

interface ControllerEvent {
  id: string;
  controller_id: string;
  network_id: string;
  ledger_sequence: number;
  transaction_hash: string;
  event_index: number;
  event_type: string;
  topics: unknown;
  data: unknown;
  observed_at: string;
}

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; events: ControllerEvent[] };

export default function ActivityPage() {
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const events = await api<ControllerEvent[]>("/api/v1/events");
        if (active) setState({ kind: "ready", events });
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
          <span className="eyebrow">ACTIVITY</span>
          <h1>Event history</h1>
          <p>Every displayed event retains its ledger, transaction hash, and event index.</p>
        </div>
      </header>

      {state.kind === "loading" && (
        <section className="empty-state">
          <span className="eyebrow">LOADING</span>
          <h2>Reading event journal</h2>
          <p>Waiting for indexed events from the Console API.</p>
        </section>
      )}

      {state.kind === "error" && (
        <section className="empty-state">
          <span className="eyebrow">API UNAVAILABLE</span>
          <h2>Event history could not be loaded</h2>
          <p>{state.message}</p>
        </section>
      )}

      {state.kind === "ready" && state.events.length === 0 && (
        <section className="empty-state">
          <span className="eyebrow">NO EVENTS</span>
          <h2>No event history is indexed</h2>
          <p>The indexer has not observed any UpgradeController events yet.</p>
        </section>
      )}

      {state.kind === "ready" && state.events.length > 0 && (
        <div className="data-table-wrap">
          <table>
            <thead>
              <tr>
                <th>Ledger</th>
                <th>Event Type</th>
                <th>Tx Hash</th>
                <th>Index</th>
                <th>Observed At</th>
                <th>Payload</th>
              </tr>
            </thead>
            <tbody>
              {state.events.map((event) => (
                <tr key={event.id}>
                  <td>{event.ledger_sequence}</td>
                  <td>
                    <span
                      style={{
                        padding: "2px 6px",
                        borderRadius: "4px",
                        background: "var(--raised)",
                        border: "1px solid var(--line)",
                        fontSize: "12px",
                      }}
                    >
                      {event.event_type}
                    </span>
                  </td>
                  <td title={event.transaction_hash}>{event.transaction_hash}</td>
                  <td>#{event.event_index}</td>
                  <td>{event.observed_at ? new Date(event.observed_at).toLocaleString() : "—"}</td>
                  <td title={JSON.stringify(event.data)}>
                    {formatPayload(event.data)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function formatPayload(data: unknown): string {
  if (!data || typeof data !== "object") return "—";
  try {
    const entries = Object.entries(data as Record<string, unknown>);
    if (entries.length === 0) return "—";
    return entries.map(([k, v]) => `${k}: ${String(v)}`).join(", ");
  } catch {
    return "—";
  }
}
