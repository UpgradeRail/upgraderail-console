"use client";

import { useEffect, useState } from "react";
import { CreateAnalysisFromUploads } from "@/components/create-analysis-from-uploads";
import { api, ApiError } from "@/lib/api";

type Session = { address: string; network: string };
type State =
  | { kind: "loading" }
  | { kind: "signed_out"; message: string }
  | { kind: "error"; message: string }
  | { kind: "ready"; session: Session };

export default function NewAnalysisPage() {
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const session = await api<Session>("/api/v1/auth/session");
        if (active) setState({ kind: "ready", session });
      } catch (error) {
        if (!active) return;
        if (error instanceof ApiError && error.status === 401) {
          setState({ kind: "signed_out", message: "Connect and sign in with Freighter in the sidebar before uploading artifacts." });
        } else {
          setState({ kind: "error", message: error instanceof Error ? error.message : "Session could not be read." });
        }
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  return (
    <section className="console-page">
      <header className="console-heading">
        <div>
          <span className="eyebrow">PREFLIGHT</span>
          <h1>New analysis</h1>
          <p>
            Upload the current and candidate WASM to run the Engine. Artifacts are stored on the API&apos;s local filesystem
            (<code>ARTIFACT_LOCAL_DIR</code>); the worker must be able to read that same path, which is not yet a production
            object store.
          </p>
        </div>
      </header>
      {state.kind === "loading" && (
        <section className="empty-state">
          <span className="eyebrow">LOADING</span>
          <h2>Reading session</h2>
          <p>Checking whether a wallet session is active.</p>
        </section>
      )}
      {state.kind === "signed_out" && (
        <section className="empty-state">
          <span className="eyebrow">SIGN IN REQUIRED</span>
          <h2>Wallet session needed</h2>
          <p>{state.message}</p>
        </section>
      )}
      {state.kind === "error" && (
        <section className="empty-state">
          <span className="eyebrow">ERROR</span>
          <h2>Session could not be read</h2>
          <p>{state.message}</p>
        </section>
      )}
      {state.kind === "ready" && <CreateAnalysisFromUploads session={state.session} />}
    </section>
  );
}
