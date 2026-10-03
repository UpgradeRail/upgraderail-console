import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { connectWallet, readWallet, signWalletMessage, signWalletTransaction, stateForWalletError, transactionLifecycle } from "./wallet";
import { getNetwork, isConnected, requestAccess, signMessage, signTransaction } from "@stellar/freighter-api";

vi.mock("@stellar/freighter-api", () => ({
  getNetwork: vi.fn(),
  isConnected: vi.fn(),
  requestAccess: vi.fn(),
  signMessage: vi.fn(),
  signTransaction: vi.fn(),
}));

const mocked = {
  getNetwork: vi.mocked(getNetwork),
  isConnected: vi.mocked(isConnected),
  requestAccess: vi.mocked(requestAccess),
  signMessage: vi.mocked(signMessage),
  signTransaction: vi.mocked(signTransaction),
};

const testnet = { network: "TESTNET", networkPassphrase: "Test SDF Network ; September 2015" };

beforeEach(() => vi.resetAllMocks());
afterEach(() => vi.useRealTimers());

describe("wallet detection and signing", () => {
  it("reports a missing extension", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: false });
    await expect(readWallet()).resolves.toEqual({ kind: "unavailable" });
    expect(mocked.getNetwork).not.toHaveBeenCalled();
  });

  it("detects an unlocked wallet without requesting account approval", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: true });
    mocked.getNetwork.mockResolvedValue(testnet);
    await expect(readWallet(testnet.networkPassphrase)).resolves.toEqual({ kind: "connect_ready", network: { id: "testnet", name: "TESTNET", passphrase: testnet.networkPassphrase } });
    expect(mocked.requestAccess).not.toHaveBeenCalled();
  });

  it("shows a recoverable network error when network lookup fails", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: true });
    mocked.getNetwork.mockResolvedValue({ network: "", networkPassphrase: "", error: { code: 7, message: "RPC network unavailable" } });
    await expect(readWallet()).resolves.toEqual({ kind: "network_unavailable", message: "RPC network unavailable" });
  });

  it("does not stay checking when Freighter never answers", async () => {
    vi.useFakeTimers();
    mocked.isConnected.mockReturnValue(new Promise(() => {}));
    const result = readWallet();
    await vi.advanceTimersByTimeAsync(5_001);
    await expect(result).resolves.toMatchObject({ kind: "network_unavailable", message: "Freighter did not answer the extension check." });
  });

  it("reports a wrong network", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: true });
    mocked.getNetwork.mockResolvedValue({ network: "PUBLIC", networkPassphrase: "Public Global Stellar Network ; September 2015" });
    await expect(readWallet(testnet.networkPassphrase)).resolves.toEqual({ kind: "wrong_network", expected: testnet.networkPassphrase, actual: "PUBLIC" });
  });

  it("keeps API connectivity failures separate from wallet network failures", () => {
    expect(stateForWalletError("The Console API could not be reached: Failed to fetch")).toEqual({
      kind: "failed",
      message: "The Console API could not be reached: Failed to fetch",
    });
  });

  it("connects with the explicitly approved account and selected network", async () => {
    mocked.requestAccess.mockResolvedValue({ address: "GTEST" });
    mocked.getNetwork.mockResolvedValue(testnet);
    await expect(connectWallet(testnet.networkPassphrase)).resolves.toEqual({ kind: "connected", address: "GTEST", network: { id: "testnet", name: "TESTNET", passphrase: testnet.networkPassphrase } });
  });

  it("keeps access rejection distinct from a wallet failure", async () => {
    mocked.requestAccess.mockResolvedValue({ address: "", error: { code: -1, message: "User rejected access" } });
    await expect(connectWallet()).resolves.toEqual({ kind: "user_rejected", message: "User rejected access" });
  });

  it("returns signed challenge messages and transactions", async () => {
    mocked.signMessage.mockResolvedValue({ signedMessage: "signed", signerAddress: "GTEST" });
    mocked.signTransaction.mockResolvedValue({ signedTxXdr: "signed-xdr", signerAddress: "GTEST" });
    await expect(signWalletMessage("challenge", testnet.networkPassphrase, "GTEST")).resolves.toBe("signed");
    await expect(signWalletTransaction("xdr", testnet.networkPassphrase, "GTEST")).resolves.toBe("signed-xdr");
  });

  it("surfaces challenge and transaction signing rejection", async () => {
    mocked.signMessage.mockResolvedValue({ signedMessage: null, signerAddress: "", error: { code: -1, message: "User rejected message signing" } });
    mocked.signTransaction.mockResolvedValue({ signedTxXdr: "", signerAddress: "", error: { code: -1, message: "User rejected transaction signing" } });
    await expect(signWalletMessage("challenge", testnet.networkPassphrase, "GTEST")).rejects.toThrow("User rejected message signing");
    await expect(signWalletTransaction("xdr", testnet.networkPassphrase, "GTEST")).rejects.toThrow("User rejected transaction signing");
  });

  it("rejects signatures returned for another account", async () => {
    mocked.signTransaction.mockResolvedValue({ signedTxXdr: "signed-xdr", signerAddress: "GOTHER" });
    await expect(signWalletTransaction("xdr", testnet.networkPassphrase, "GTEST")).rejects.toThrow("Freighter signed with a different account.");
  });

  it("classifies transaction polling states", () => {
    expect(transactionLifecycle("pending", "tx")).toEqual({ kind: "submit_pending", hash: "tx" });
    expect(transactionLifecycle("confirmed", "tx")).toEqual({ kind: "confirmed", hash: "tx" });
    expect(transactionLifecycle("failed", "tx", "bad auth")).toEqual({ kind: "failed", hash: "tx", message: "bad auth" });
  });
});
