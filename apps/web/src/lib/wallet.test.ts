import { beforeEach, describe, expect, it, vi } from "vitest";
import { connectWallet, readWallet, signWalletMessage, signWalletTransaction, transactionLifecycle } from "./wallet";
import { getAddress, getNetwork, isConnected, requestAccess, signMessage, signTransaction } from "@stellar/freighter-api";

vi.mock("@stellar/freighter-api", () => ({
  getAddress: vi.fn(),
  getNetwork: vi.fn(),
  isConnected: vi.fn(),
  requestAccess: vi.fn(),
  signMessage: vi.fn(),
  signTransaction: vi.fn(),
}));

const mocked = {
  getAddress: vi.mocked(getAddress),
  getNetwork: vi.mocked(getNetwork),
  isConnected: vi.mocked(isConnected),
  requestAccess: vi.mocked(requestAccess),
  signMessage: vi.mocked(signMessage),
  signTransaction: vi.mocked(signTransaction),
};

beforeEach(() => {
  vi.resetAllMocks();
});

describe("wallet lifecycle", () => {
  it("reports a missing wallet", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: false });
    await expect(readWallet()).resolves.toEqual({ kind: "unavailable" });
  });

  it("reports a wrong network", async () => {
    mocked.isConnected.mockResolvedValue({ isConnected: true });
    mocked.getAddress.mockResolvedValue({ address: "GB..." });
    mocked.getNetwork.mockResolvedValue({ network: "PUBLIC", networkPassphrase: "Public Global Stellar Network ; September 2015" });
    await expect(readWallet("Test SDF Network ; September 2015")).resolves.toEqual({
      kind: "wrong_network",
      expected: "Test SDF Network ; September 2015",
      actual: "PUBLIC",
    });
  });

  it("reports user rejection during connection", async () => {
    mocked.requestAccess.mockResolvedValue({ address: "", error: { code: -1, message: "Rejected" } });
    await expect(connectWallet()).resolves.toEqual({ kind: "error", message: "Freighter did not grant wallet access." });
  });

  it("returns signed challenge messages and transactions", async () => {
    mocked.signMessage.mockResolvedValue({ signedMessage: "signed", signerAddress: "GB..." });
    mocked.signTransaction.mockResolvedValue({ signedTxXdr: "signed-xdr", signerAddress: "GB..." });
    await expect(signWalletMessage("challenge", "Test SDF Network ; September 2015", "GB...")).resolves.toBe("signed");
    await expect(signWalletTransaction("xdr", "Test SDF Network ; September 2015", "GB...")).resolves.toBe("signed-xdr");
  });

  it("surfaces submit and signing failures", async () => {
    mocked.signTransaction.mockResolvedValue({ signedTxXdr: "", signerAddress: "", error: { code: -1, message: "Rejected" } });
    await expect(signWalletTransaction("xdr", "Test SDF Network ; September 2015", "GB...")).rejects.toThrow("Freighter did not return a signed transaction.");
  });

  it("classifies transaction polling states", () => {
    expect(transactionLifecycle("pending", "tx")).toEqual({ kind: "pending", hash: "tx" });
    expect(transactionLifecycle("confirmed", "tx")).toEqual({ kind: "confirmed", hash: "tx" });
    expect(transactionLifecycle("failed", "tx", "bad auth")).toEqual({ kind: "failed", hash: "tx", message: "bad auth" });
  });
});
