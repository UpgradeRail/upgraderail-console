import { getAddress, getNetwork, isConnected, requestAccess, signMessage, signTransaction } from "@stellar/freighter-api";

export type WalletState =
  | { kind: "checking" }
  | { kind: "unavailable" }
  | { kind: "disconnected" }
  | { kind: "wrong_network"; expected: string; actual: string }
  | { kind: "connected"; address: string }
  | { kind: "error"; message: string };

export type TransactionLifecycle =
  | { kind: "idle" }
  | { kind: "pending"; hash: string }
  | { kind: "confirmed"; hash: string }
  | { kind: "failed"; hash?: string; message: string };

export async function readWallet(expectedNetworkPassphrase?: string): Promise<WalletState> {
  try {
    const connection = await isConnected();
    if (connection.error || !connection.isConnected) return { kind: "unavailable" };
    const address = await getAddress();
    if (address.error || !address.address) return { kind: "disconnected" };
    if (expectedNetworkPassphrase) {
      const network = await getNetwork();
      if (network.error) return { kind: "error", message: "Freighter network could not be read." };
      if (network.networkPassphrase !== expectedNetworkPassphrase) {
        return { kind: "wrong_network", expected: expectedNetworkPassphrase, actual: network.network || "unknown" };
      }
    }
    return { kind: "connected", address: address.address };
  } catch (error) {
    return { kind: "error", message: messageFor(error) };
  }
}

export async function connectWallet(): Promise<WalletState> {
  try {
    const result = await requestAccess();
    if (result.error || !result.address) return { kind: "error", message: "Freighter did not grant wallet access." };
    return { kind: "connected", address: result.address };
  } catch (error) {
    return { kind: "error", message: messageFor(error) };
  }
}

export async function signWalletTransaction(transactionXdr: string, networkPassphrase: string, address: string): Promise<string> {
  const result = await signTransaction(transactionXdr, { networkPassphrase, address });
  if (result.error || !result.signedTxXdr) throw new Error("Freighter did not return a signed transaction.");
  return result.signedTxXdr;
}

export async function signWalletMessage(message: string, networkPassphrase: string, address: string): Promise<string> {
  const result = await signMessage(message, { networkPassphrase, address });
  if (result.error || !result.signedMessage) throw new Error("Freighter did not return a signed message.");
  return typeof result.signedMessage === "string" ? result.signedMessage : result.signedMessage.toString("base64");
}

export function transactionLifecycle(status: string, hash?: string, message?: string): TransactionLifecycle {
  if (status === "pending" && hash) return { kind: "pending", hash };
  if (status === "confirmed" && hash) return { kind: "confirmed", hash };
  if (status === "failed") return { kind: "failed", hash, message: message || "Transaction failed." };
  return { kind: "idle" };
}

function messageFor(error: unknown): string { return error instanceof Error ? error.message : "Freighter could not be reached."; }
