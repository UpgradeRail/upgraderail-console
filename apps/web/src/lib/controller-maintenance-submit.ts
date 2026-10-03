import { Transaction, TransactionBuilder, rpc } from "@stellar/stellar-sdk";
import { maintenanceRpc, TESTNET_PASSPHRASE, type UnsignedMaintenance } from "./controller-maintenance";
import { validateSignedMaintenance } from "./controller-maintenance-signing";

export type MaintenanceResult =
  | { kind: "pending"; hash: string }
  | { kind: "confirmed"; hash: string; ledger: number }
  | { kind: "failed"; hash: string; ledger?: number; reason: string };

function resultCode(result: { result: { type?: string } } | undefined): string {
  return result?.result.type ?? "No result code returned";
}

export async function submitMaintenance(draft: UnsignedMaintenance, signedXdr: string, server = maintenanceRpc()): Promise<MaintenanceResult> {
  validateSignedMaintenance(draft, signedXdr);
  const transaction = TransactionBuilder.fromXDR(signedXdr, TESTNET_PASSPHRASE);
  if (!(transaction instanceof Transaction)) throw new Error("Expected a controller maintenance transaction.");
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

export async function pollMaintenance(hash: string, server = maintenanceRpc()): Promise<MaintenanceResult> {
  const response = await server.getTransaction(hash);
  if (response.txHash.toLowerCase() !== hash.toLowerCase()) throw new Error("Stellar RPC returned a different transaction hash.");
  if (response.status === "SUCCESS") return { kind: "confirmed", hash, ledger: response.ledger };
  if (response.status === "FAILED") return { kind: "failed", hash, ledger: response.ledger, reason: `Transaction failed: ${resultCode(response.resultXdr)}` };
  return { kind: "pending", hash };
}
