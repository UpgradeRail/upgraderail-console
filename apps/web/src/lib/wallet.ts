import { getAddress, isConnected, requestAccess, signMessage, signTransaction } from "@stellar/freighter-api";

export type WalletState =
  | { kind: "checking" }
  | { kind: "unavailable" }
  | { kind: "disconnected" }
  | { kind: "connected"; address: string }
  | { kind: "error"; message: string };

export async function readWallet(): Promise<WalletState> {
  try {
    const connection = await isConnected();
    if (connection.error || !connection.isConnected) return { kind: "unavailable" };
    const address = await getAddress();
    if (address.error || !address.address) return { kind: "disconnected" };
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

function messageFor(error: unknown): string { return error instanceof Error ? error.message : "Freighter could not be reached."; }
