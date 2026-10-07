import { Transaction, TransactionBuilder, rpc } from "@stellar/stellar-sdk";
import {
  assertStateFresh,
  defaultRpc,
  readLiveGovernanceState,
  TESTNET_PASSPHRASE,
  type UnsignedGovernanceTx,
} from "./governance-tx";

export type GovernanceResult =
  | { kind: "pending"; hash: string }
  | { kind: "confirmed"; hash: string; ledger: number }
  | { kind: "failed"; hash: string; ledger?: number; reason: string };

export const DRAFT_MAX_AGE_MS = 120_000;

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function resultCode(result: { result: { type?: string } } | undefined): string {
  return result?.result.type ?? "No result code returned";
}

/** Re-reads live controller/account state and rejects a draft that has drifted. */
export async function refreshGovernanceState(draft: UnsignedGovernanceTx, server = defaultRpc(), now = Date.now()): Promise<void> {
  if (now - draft.builtAt > DRAFT_MAX_AGE_MS) {
    throw new Error("The unsigned transaction is older than two minutes. Build a new one.");
  }
  const { snapshot } = await readLiveGovernanceState(server, draft.address);
  assertStateFresh(draft.snapshot, snapshot);
}

/** Confirms the wallet returned exactly the transaction that was built and simulated. */
export function validateSignedGovernance(draft: UnsignedGovernanceTx, signedXdr: string): void {
  const unsigned = TransactionBuilder.fromXDR(draft.unsignedXdr, TESTNET_PASSPHRASE);
  const signed = TransactionBuilder.fromXDR(signedXdr, TESTNET_PASSPHRASE);
  if (!(unsigned instanceof Transaction) || !(signed instanceof Transaction)) {
    throw new Error("Freighter returned a fee-bump transaction instead of the governance call.");
  }
  if (
    signed.source !== draft.address ||
    signed.sequence !== unsigned.sequence ||
    toHex(unsigned.hash()) !== draft.transactionHash ||
    toHex(signed.hash()) !== draft.transactionHash
  ) {
    throw new Error("Freighter returned a transaction with different contents. Rebuild before signing.");
  }
  if (signed.signatures.length === 0) throw new Error("Freighter returned an unsigned transaction.");
}

export async function submitGovernance(
  draft: UnsignedGovernanceTx,
  signedXdr: string,
  server = defaultRpc()
): Promise<GovernanceResult> {
  validateSignedGovernance(draft, signedXdr);
  const transaction = TransactionBuilder.fromXDR(signedXdr, TESTNET_PASSPHRASE);
  if (!(transaction instanceof Transaction)) throw new Error("Expected a governance transaction.");
  const response = await server.sendTransaction(transaction);
  if (response.hash.toLowerCase() !== draft.transactionHash) {
    throw new Error("Stellar RPC returned a transaction hash that differs from the signed transaction.");
  }
  if (response.status === "ERROR") {
    return { kind: "failed", hash: response.hash, reason: `RPC rejected the transaction: ${resultCode(response.errorResult)}` };
  }
  if (response.status === "TRY_AGAIN_LATER") {
    return { kind: "failed", hash: response.hash, reason: "RPC is temporarily unable to accept this transaction. Rebuild and try again later." };
  }
  return { kind: "pending", hash: response.hash };
}

export async function pollGovernance(hash: string, server: Pick<rpc.Server, "getTransaction"> = defaultRpc()): Promise<GovernanceResult> {
  const response = await server.getTransaction(hash);
  if (response.txHash.toLowerCase() !== hash.toLowerCase()) throw new Error("Stellar RPC returned a different transaction hash.");
  if (response.status === "SUCCESS") return { kind: "confirmed", hash, ledger: response.ledger };
  if (response.status === "FAILED") return { kind: "failed", hash, ledger: response.ledger, reason: `Transaction failed: ${resultCode(response.resultXdr)}` };
  return { kind: "pending", hash };
}
