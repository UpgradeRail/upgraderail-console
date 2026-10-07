"use client";

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import {
  pollGovernance,
  refreshGovernanceState,
  submitGovernance,
  validateSignedGovernance,
  type GovernanceResult,
} from "@/lib/governance-flow";
import { TESTNET_PASSPHRASE, type UnsignedGovernanceTx } from "@/lib/governance-tx";
import { requireWalletAccount, signWalletTransaction, stateForWalletError } from "@/lib/wallet";

type Phase = "idle" | "building" | "unsigned" | "signing" | "signed" | "submitting" | "pending" | "confirmed" | "user_rejected" | "failed";
type Session = { address: string; network: string };

export type GovernanceActionPanelProps = {
  title: string;
  description: string;
  /** Contract method name shown to the reviewer. */
  action: string;
  /** Key/value facts the signer should review before signing. */
  details: ReadonlyArray<readonly [string, string]>;
  confirmLabel: string;
  disabledReason?: string;
  build: (address: string) => Promise<UnsignedGovernanceTx>;
  onConfirmed?: () => void;
};

/**
 * One governance transaction lifecycle: build + simulate, review, Freighter
 * signature, explicit confirmation, Testnet submission, and result polling.
 * Live state is re-read before signing and again before submitting.
 */
export function GovernanceActionPanel(props: GovernanceActionPanelProps) {
  const [phase, setPhase] = useState<Phase>("idle");
  const [draft, setDraft] = useState<UnsignedGovernanceTx | null>(null);
  const [signedXdr, setSignedXdr] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [result, setResult] = useState<GovernanceResult | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const busy = phase === "building" || phase === "signing" || phase === "submitting";

  async function build() {
    setPhase("building");
    setMessage("");
    setDraft(null);
    setSignedXdr(null);
    setResult(null);
    setConfirmed(false);
    try {
      const session = await api<Session>("/api/v1/auth/session");
      if (session.network !== "testnet") throw new Error("Sign in with a Testnet wallet before building.");
      await requireWalletAccount(session.address, TESTNET_PASSPHRASE);
      setDraft(await props.build(session.address));
      setPhase("unsigned");
    } catch (error) {
      setMessage(
        error instanceof ApiError && error.status === 401
          ? "Connect and sign in with Freighter in the sidebar first."
          : error instanceof Error ? error.message : "The unsigned transaction could not be built."
      );
      setPhase("failed");
    }
  }

  async function sign() {
    if (!draft) return;
    setPhase("signing");
    setMessage("");
    try {
      await requireWalletAccount(draft.address, draft.networkPassphrase);
      await refreshGovernanceState(draft);
      const signed = await signWalletTransaction(draft.unsignedXdr, draft.networkPassphrase, draft.address);
      validateSignedGovernance(draft, signed);
      setSignedXdr(signed);
      setPhase("signed");
    } catch (error) {
      const detail = error instanceof Error ? error.message : "Freighter could not sign the transaction.";
      setMessage(detail);
      setPhase(stateForWalletError(detail).kind === "user_rejected" ? "user_rejected" : "failed");
    }
  }

  async function poll(hash: string) {
    try {
      for (let attempt = 0; attempt < 20; attempt++) {
        const next = await pollGovernance(hash);
        setResult(next);
        if (next.kind !== "pending") {
          setPhase(next.kind);
          if (next.kind === "confirmed") props.onConfirmed?.();
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
    if (!draft || !signedXdr || !confirmed) return;
    setPhase("submitting");
    setMessage("");
    try {
      await requireWalletAccount(draft.address, draft.networkPassphrase);
      await refreshGovernanceState(draft);
      const next = await submitGovernance(draft, signedXdr);
      setResult(next);
      setPhase(next.kind);
      if (next.kind === "pending") await poll(next.hash);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "The transaction could not be submitted.");
      setPhase("failed");
    }
  }

  return (
    <section className="verification-card" aria-live="polite" style={{ marginTop: 0 }}>
      <span className="eyebrow">GOVERNANCE · TESTNET</span>
      <h2>{props.title}</h2>
      <p>{props.description}</p>
      <dl className="verification-summary">
        <div><dt>Contract method</dt><dd>{props.action}</dd></div>
        <div><dt>Network</dt><dd>Stellar Testnet</dd></div>
        {props.details.map(([label, value]) => <div key={label}><dt>{label}</dt><dd className="verification-hash">{value}</dd></div>)}
      </dl>
      {props.disabledReason && <p role="status">Unavailable: {props.disabledReason}</p>}
      <div className="verification-actions">
        <button className="button button-accent" type="button" onClick={() => void build()} disabled={busy || Boolean(props.disabledReason)}>Build and simulate transaction</button>
        {draft && (phase === "unsigned" || phase === "user_rejected" || phase === "failed") &&
          <button className="button" type="button" onClick={() => void sign()}>Sign in Freighter</button>}
        {phase === "signed" && <button className="button button-accent" type="button" onClick={() => void submit()} disabled={!confirmed}>{props.confirmLabel}</button>}
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
      {phase === "unsigned" && <p role="status">Unsigned transaction built and simulated. Nothing has been signed or submitted.</p>}
      {phase === "signing" && <p role="status">Waiting for the Freighter signing prompt…</p>}
      {phase === "signed" && <div role="status"><p>Freighter returned a signature and it matches the simulated transaction. It has not been submitted.</p><label><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} /> I confirm I want to submit this {props.action} transaction to Stellar Testnet and pay up to the displayed maximum fee.</label></div>}
      {phase === "submitting" && <p role="status">Submitting to Stellar Testnet RPC…</p>}
      {phase === "pending" && <p role="status">Pending. Hash: {result?.hash}. {message}</p>}
      {phase === "confirmed" && result?.kind === "confirmed" && <p role="status">Confirmed on Stellar Testnet in ledger {result.ledger}. Hash: {result.hash}. The indexer projection may lag; refresh to see the projected state.</p>}
      {phase === "user_rejected" && <p role="alert">Request declined: {message}. No transaction was submitted.</p>}
      {phase === "failed" && <p role="alert">{result?.kind === "failed" ? `Transaction ${result.hash}${result.ledger ? ` in ledger ${result.ledger}` : ""}: ${result.reason}` : message}</p>}
    </section>
  );
}
