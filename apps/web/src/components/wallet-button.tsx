"use client";

import { useEffect, useState } from "react";
import { connectWallet, readWallet, type WalletState } from "@/lib/wallet";

export function WalletButton() {
  const [state, setState] = useState<WalletState>({ kind: "checking" });
  useEffect(() => { void readWallet().then(setState); }, []);
  if (state.kind === "connected") return <span className="wallet-address" title={state.address}>{state.address.slice(0, 5)}…{state.address.slice(-4)}</span>;
  if (state.kind === "unavailable") return <a className="wallet-help" href="https://www.freighter.app/" target="_blank" rel="noreferrer">Install Freighter</a>;
  if (state.kind === "checking") return <span className="wallet-help">Checking wallet…</span>;
  return <button className="wallet-connect" onClick={() => void connectWallet().then(setState)}>{state.kind === "error" ? "Retry wallet" : "Connect wallet"}</button>;
}
