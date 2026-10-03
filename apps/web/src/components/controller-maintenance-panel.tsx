"use client";

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { buildMaintenanceTransaction, refreshMaintenanceState, TESTNET_PASSPHRASE, VERIFIED_CONTROLLER_ID, type UnsignedMaintenance } from "@/lib/controller-maintenance";
import { validateSignedMaintenance } from "@/lib/controller-maintenance-signing";
import { pollMaintenance, submitMaintenance, type MaintenanceResult } from "@/lib/controller-maintenance-submit";
import { requireWalletAccount, signWalletTransaction, stateForWalletError } from "@/lib/wallet";

type Phase = "idle" | "building" | "unsigned" | "signing" | "signed" | "submitting" | "pending" | "confirmed" | "user_rejected" | "failed";
type Session = { address: string; network: string };

export function ControllerMaintenancePanel() {
  const [phase, setPhase] = useState<Phase>("idle");
  const [draft, setDraft] = useState<UnsignedMaintenance | null>(null);
  const [signedXdr, setSignedXdr] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [result, setResult] = useState<MaintenanceResult | null>(null);
  const [disposable, setDisposable] = useState(false);

  async function build() {
    setPhase("building");
    setMessage("");
    setDraft(null);
    setSignedXdr(null);
    setResult(null);
    setDisposable(false);
    try {
      const session = await api<Session>("/api/v1/auth/session");
      if (session.network !== "testnet") throw new Error("Sign in with a Testnet wallet before building.");
      await requireWalletAccount(session.address, TESTNET_PASSPHRASE);
      const built = await buildMaintenanceTransaction(session.address);
      setDraft(built);
      setPhase("unsigned");
    } catch (error) {
      setMessage(error instanceof ApiError && error.status === 401
        ? "Connect and sign in with Freighter in the sidebar first."
        : error instanceof Error ? error.message : "The unsigned transaction could not be built.");
      setPhase("failed");
    }
  }

  async function poll(hash: string) {
    try {
      for (let attempt = 0; attempt < 20; attempt++) {
        const next = await pollMaintenance(hash);
        setResult(next);
        if (next.kind !== "pending") {
          setPhase(next.kind);
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 3000));
      }
      setMessage("Still pending. Continue polling to check the public Testnet result.");
    } catch (error) {
      setMessage(`Polling failed: ${error instanceof Error ? error.message : "RPC unavailable"}. Continue polling to retry.`);
    }
    setPhase("pending");
  }

  async function submit() {
    if (!draft || !signedXdr || !disposable) return;
    setPhase("submitting");
    setMessage("");
    try {
      await requireWalletAccount(draft.address, draft.networkPassphrase);
      await refreshMaintenanceState(draft);
      const next = await submitMaintenance(draft, signedXdr);
      setResult(next);
      setPhase(next.kind);
      if (next.kind === "pending") await poll(next.hash);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "The transaction could not be submitted.");
      setPhase("failed");
    }
  }

  async function sign() {
    if (!draft) return;
    setPhase("signing");
    setMessage("");
    try {
      await requireWalletAccount(draft.address, draft.networkPassphrase);
      await refreshMaintenanceState(draft);
      const signed = await signWalletTransaction(draft.unsignedXdr, draft.networkPassphrase, draft.address);
      validateSignedMaintenance(draft, signed);
      setSignedXdr(signed);
      setPhase("signed");
    } catch (error) {
      const detail = error instanceof Error ? error.message : "Freighter could not sign the transaction.";
      setMessage(detail);
      setPhase(stateForWalletError(detail).kind === "user_rejected" ? "user_rejected" : "failed");
    }
  }

  return (
    <section className="verification-card" aria-live="polite">
      <span className="eyebrow">TESTNET VERIFICATION</span>
      <h2>Maintain controller storage</h2>
      <p>This call extends the verified controller&apos;s instance storage lifetime. It does not change policy, proposals, fleets, approvals, or controller version. Build and simulate it before reviewing the unsigned transaction in Freighter.</p>
      <dl className="verification-summary">
        <div><dt>Action</dt><dd>maintain_controller</dd></div>
        <div><dt>Network</dt><dd>Stellar Testnet</dd></div>
        <div><dt>Contract</dt><dd className="verification-hash">{VERIFIED_CONTROLLER_ID}</dd></div>
      </dl>
      <div className="verification-actions">
        <button className="button button-accent" type="button" onClick={() => void build()} disabled={phase === "building" || phase === "signing"}>Build fresh unsigned transaction</button>
        {draft && (phase === "unsigned" || phase === "user_rejected" || phase === "failed") &&
          <button className="button" type="button" onClick={() => void sign()}>Sign in Freighter</button>}
        {phase === "signed" && <button className="button button-accent" type="button" onClick={() => void submit()} disabled={!disposable}>Submit to Testnet</button>}
        {phase === "pending" && result?.kind === "pending" && <button className="button" type="button" onClick={() => void poll(result.hash)}>Continue polling</button>}
      </div>
      {phase === "building" && <p role="status">Refreshing live controller state and simulating the call…</p>}
      {draft && <dl className="verification-summary">
        <div><dt>Observed ledger</dt><dd>{draft.snapshot.ledger}</dd></div>
        <div><dt>Governance epoch</dt><dd>{draft.snapshot.governanceEpoch}</dd></div>
        <div><dt>Controller version</dt><dd>{draft.snapshot.controllerVersion}</dd></div>
        <div><dt>Maximum fee</dt><dd>{draft.feeStroops} stroops</dd></div>
        <div><dt>Transaction hash</dt><dd className="verification-hash">{draft.transactionHash}</dd></div>
      </dl>}
      {phase === "unsigned" && <p role="status">Unsigned XDR built and simulated. Review the action above before signing.</p>}
      {phase === "signing" && <p role="status">Waiting for the Freighter transaction signing prompt…</p>}
      {phase === "signed" && signedXdr && <div role="status"><p>Signed XDR returned by Freighter and checked against the unsigned transaction. It has not been submitted.</p><label><input type="checkbox" checked={disposable} onChange={(event) => setDisposable(event.target.checked)} /> I confirm this is a disposable Testnet account and approve paying the displayed maximum fee to extend controller storage lifetime.</label></div>}
      {phase === "submitting" && <p role="status">Submitting the signed transaction to Stellar Testnet RPC…</p>}
      {phase === "pending" && <p role="status">Transaction pending. Hash: {result?.hash}. {message}</p>}
      {phase === "confirmed" && result?.kind === "confirmed" && <p role="status">Confirmed on Stellar Testnet in ledger {result.ledger}. Transaction hash: {result.hash}.</p>}
      {phase === "user_rejected" && <p role="alert">Request declined: {message}. No transaction was submitted.</p>}
      {phase === "failed" && <p role="alert">{result?.kind === "failed" ? `Transaction ${result.hash}${result.ledger ? ` in ledger ${result.ledger}` : ""}: ${result.reason}` : message}</p>}
    </section>
  );
}
