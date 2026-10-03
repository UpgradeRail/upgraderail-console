"use client";

import { useCallback, useEffect, useState } from "react";
import { connectWallet, expectedNetworkPassphrase, readWallet, type WalletState } from "@/lib/wallet";

export function WalletButton() {
  const [state, setState] = useState<WalletState>({ kind: "checking" });

  const refresh = useCallback(async () => {
    setState({ kind: "checking" });
    setState(await readWallet(expectedNetworkPassphrase()));
  }, []);

  useEffect(() => {
    let active = true;
    void readWallet(expectedNetworkPassphrase()).then((result) => {
      if (active) setState(result);
    });
    return () => { active = false; };
  }, []);

  async function handleConnect() {
    setState({ kind: "connecting" });
    setState(await connectWallet(expectedNetworkPassphrase()));
  }

  return (
    <div className="wallet-panel" aria-live="polite" aria-atomic="true">
      {state.kind === "checking" && <span className="wallet-message">Checking Freighter…</span>}
      {state.kind === "unavailable" && (
        <>
          <span className="wallet-message" role="status">Freighter not detected.</span>
          <a className="wallet-help" href="https://www.freighter.app/" target="_blank" rel="noreferrer">Install Freighter</a>
          <button className="wallet-secondary" onClick={() => void refresh()}>Retry detection</button>
        </>
      )}
      {state.kind === "account_locked" && (
        <>
          <span className="wallet-error" role="alert">{state.message}</span>
          <button className="wallet-connect" onClick={() => void handleConnect()}>Unlock or connect</button>
          <button className="wallet-secondary" onClick={() => void refresh()}>Retry detection</button>
        </>
      )}
      {state.kind === "network_unavailable" && (
        <>
          <span className="wallet-error" role="alert">Network unavailable: {state.message}</span>
          {state.address && <span className="wallet-message">Wallet access granted: {shortAddress(state.address)}</span>}
          <button className="wallet-connect" onClick={() => void handleConnect()}>Connect Freighter</button>
          <button className="wallet-secondary" onClick={() => void refresh()}>Retry network check</button>
        </>
      )}
      {state.kind === "connect_ready" && (
        <>
          <span className="wallet-message">Freighter available · {state.network.name || "Network detected"}</span>
          <button className="wallet-connect" onClick={() => void handleConnect()}>Connect Freighter</button>
        </>
      )}
      {state.kind === "connecting" && <button className="wallet-connect" disabled>Waiting for Freighter…</button>}
      {state.kind === "connected" && (
        <>
          <span className="wallet-address" title={state.address}>{shortAddress(state.address)}</span>
          <span className="wallet-message">Connected · {state.network.name}</span>
        </>
      )}
      {state.kind === "signing" && <span className="wallet-message">Approve the challenge in Freighter…</span>}
      {state.kind === "signed" && <span className="wallet-message">Wallet signature verified.</span>}
      {state.kind === "submit_pending" && <span className="wallet-message">Transaction pending · {state.hash}</span>}
      {state.kind === "confirmed" && <span className="wallet-message">Transaction confirmed · {state.hash}</span>}
      {state.kind === "wrong_network" && (
        <>
          <span className="wallet-error" role="alert">Wrong network: using {state.actual}. Expected {state.expected}.</span>
          <button className="wallet-secondary" onClick={() => void refresh()}>Retry network check</button>
        </>
      )}
      {state.kind === "user_rejected" && (
        <>
          <span className="wallet-error" role="alert">Request declined: {state.message}</span>
          <button className="wallet-connect" onClick={() => void handleConnect()}>Try again</button>
        </>
      )}
      {state.kind === "failed" && (
        <>
          <span className="wallet-error" role="alert">{state.message}</span>
          <button className="wallet-connect" onClick={() => void handleConnect()}>Retry wallet</button>
          <button className="wallet-secondary" onClick={() => void refresh()}>Retry detection</button>
        </>
      )}
    </div>
  );
}

function shortAddress(address: string): string {
  return `${address.slice(0, 5)}…${address.slice(-4)}`;
}
