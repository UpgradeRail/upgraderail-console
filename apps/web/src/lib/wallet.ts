import { getNetwork, isConnected, requestAccess, signMessage, signTransaction } from "@stellar/freighter-api";

const walletCallTimeoutMs = 5_000;
const connectionTimeoutMs = 60_000;
const testnetPassphrase = "Test SDF Network ; September 2015";
const publicNetworkPassphrase = "Public Global Stellar Network ; September 2015";

export type WalletNetwork = { id: string; name: string; passphrase: string };

export type WalletState =
  | { kind: "checking" }
  | { kind: "unavailable" }
  | { kind: "account_locked"; message: string }
  | { kind: "network_unavailable"; message: string; address?: string }
  | { kind: "connect_ready"; network: WalletNetwork }
  | { kind: "connecting" }
  | { kind: "connected"; address: string; network: WalletNetwork }
  | { kind: "signing"; address: string; network: WalletNetwork }
  | { kind: "signed"; address: string; network: WalletNetwork }
  | { kind: "submit_pending"; address: string; network: WalletNetwork; hash: string }
  | { kind: "confirmed"; address: string; network: WalletNetwork; hash: string }
  | { kind: "wrong_network"; expected: string; actual: string }
  | { kind: "user_rejected"; message: string }
  | { kind: "failed"; message: string };

export type TransactionLifecycle =
  | { kind: "idle" }
  | { kind: "submit_pending"; hash: string }
  | { kind: "confirmed"; hash: string }
  | { kind: "failed"; hash?: string; message: string };

export async function readWallet(expectedNetworkPassphrase?: string): Promise<WalletState> {
  try {
    const connection = await withTimeout(isConnected(), walletCallTimeoutMs, "Freighter did not answer the extension check.");
    if (connection.error) return stateForWalletError(connection.error.message);
    if (!connection.isConnected) return { kind: "unavailable" };

    const result = await withTimeout(getNetwork(), walletCallTimeoutMs, "Freighter did not return its selected network.");
    if (result.error || !result.networkPassphrase) {
      return networkStateForError(result.error?.message || "Freighter returned no network passphrase.");
    }

    const network = walletNetwork(result.network, result.networkPassphrase);
    if (expectedNetworkPassphrase && network.passphrase !== expectedNetworkPassphrase) {
      return { kind: "wrong_network", expected: expectedNetworkPassphrase, actual: network.name || network.passphrase };
    }
    return { kind: "connect_ready", network };
  } catch (error) {
    return stateForWalletError(errorMessage(error));
  }
}

export async function connectWallet(expectedNetworkPassphrase?: string): Promise<WalletState> {
  try {
    const access = await withTimeout(requestAccess(), connectionTimeoutMs, "Freighter access request timed out. Check the wallet popup and try again.");
    if (access.error) return stateForWalletError(access.error.message);
    if (!access.address) return { kind: "account_locked", message: "Freighter did not return an account. Unlock a wallet account and retry." };

    const result = await withTimeout(getNetwork(), walletCallTimeoutMs, "Freighter did not return its selected network.");
    if (result.error || !result.networkPassphrase) {
      return { kind: "network_unavailable", message: result.error?.message || "Freighter returned no network passphrase.", address: access.address };
    }

    const network = walletNetwork(result.network, result.networkPassphrase);
    if (expectedNetworkPassphrase && network.passphrase !== expectedNetworkPassphrase) {
      return { kind: "wrong_network", expected: expectedNetworkPassphrase, actual: network.name || network.passphrase };
    }
    return { kind: "connected", address: access.address, network };
  } catch (error) {
    return stateForWalletError(errorMessage(error));
  }
}

export async function signWalletTransaction(transactionXdr: string, networkPassphrase: string, address: string): Promise<string> {
  const result = await withTimeout(
    signTransaction(transactionXdr, { networkPassphrase, address }),
    connectionTimeoutMs,
    "Freighter transaction signing timed out. Check the wallet popup and try again.",
  );
  if (result.error) throw new Error(result.error.message || "Freighter could not sign the transaction.");
  if (!result.signedTxXdr) throw new Error("Freighter did not return a signed transaction.");
  if (result.signerAddress !== address) throw new Error("Freighter signed with a different account.");
  return result.signedTxXdr;
}

export async function signWalletMessage(message: string, networkPassphrase: string, address: string): Promise<string> {
  const result = await withTimeout(
    signMessage(message, { networkPassphrase, address }),
    connectionTimeoutMs,
    "Freighter message signing timed out. Check the wallet popup and try again.",
  );
  if (result.error) throw new Error(result.error.message || "Freighter could not sign the message.");
  if (!result.signedMessage) throw new Error("Freighter did not return a signed message.");
  if (result.signerAddress !== address) throw new Error("Freighter signed with a different account.");
  return typeof result.signedMessage === "string" ? result.signedMessage : result.signedMessage.toString("base64");
}

export function expectedNetworkPassphrase(): string | undefined {
  const configured = process.env.NEXT_PUBLIC_STELLAR_NETWORK_PASSPHRASE?.trim();
  if (configured) return configured;
  switch (process.env.NEXT_PUBLIC_STELLAR_NETWORK?.toLowerCase()) {
    case "testnet": return testnetPassphrase;
    case "mainnet":
    case "public": return publicNetworkPassphrase;
    default: return undefined;
  }
}

function walletNetwork(name: string, passphrase: string): WalletNetwork {
  const configured = process.env.NEXT_PUBLIC_STELLAR_NETWORK?.trim().toLowerCase();
  const id = configured || (passphrase === testnetPassphrase ? "testnet" : passphrase === publicNetworkPassphrase ? "mainnet" : name.toLowerCase());
  return { id, name, passphrase };
}

export function transactionLifecycle(status: string, hash?: string, message?: string): TransactionLifecycle {
  if (status === "pending" && hash) return { kind: "submit_pending", hash };
  if (status === "confirmed" && hash) return { kind: "confirmed", hash };
  if (status === "failed") return { kind: "failed", hash, message: message || "Transaction failed." };
  return { kind: "idle" };
}

export function stateForWalletError(message: string): WalletState {
  if (/reject|declin|cancel/i.test(message)) return { kind: "user_rejected", message };
  if (/locked|unlock|account not loaded|no account/i.test(message)) return { kind: "account_locked", message };
  if (/network|timed out|did not answer/i.test(message)) return { kind: "network_unavailable", message };
  return { kind: "failed", message };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Freighter could not be reached.";
}

function networkStateForError(message: string): WalletState {
  if (/locked|unlock|account not loaded|no account/i.test(message)) return { kind: "account_locked", message };
  return { kind: "network_unavailable", message };
}

async function withTimeout<T>(promise: Promise<T>, timeoutMs: number, message: string): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error(message)), timeoutMs);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}
