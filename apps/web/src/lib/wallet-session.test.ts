import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@/lib/api";
import { signWalletMessage } from "@/lib/wallet";
import { authenticateWallet } from "./wallet-session";

vi.mock("@/lib/api", () => ({ api: vi.fn() }));
vi.mock("@/lib/wallet", () => ({ signWalletMessage: vi.fn() }));

const mockedApi = vi.mocked(api);
const mockedSignMessage = vi.mocked(signWalletMessage);
const network = { id: "testnet", name: "TESTNET", passphrase: "Test SDF Network ; September 2015" };

beforeEach(() => vi.resetAllMocks());

describe("wallet session authentication", () => {
  it("requests, signs, verifies, then reads the cookie-backed session", async () => {
    mockedApi
      .mockResolvedValueOnce({ id: "challenge-1", nonce: "nonce-1", message: "UpgradeRail authentication" })
      .mockResolvedValueOnce({ address: "GTEST", network: "testnet" })
      .mockResolvedValueOnce({ address: "GTEST", network: "testnet" });
    mockedSignMessage.mockResolvedValue("base64-signature");

    await expect(authenticateWallet("GTEST", network)).resolves.toEqual({ address: "GTEST", network: "testnet" });

    expect(mockedApi).toHaveBeenNthCalledWith(1, "/api/v1/auth/challenge", {
      method: "POST",
      body: JSON.stringify({ address: "GTEST", network: "testnet", purpose: "console_session" }),
    });
    expect(mockedSignMessage).toHaveBeenCalledWith("UpgradeRail authentication", network.passphrase, "GTEST");
    expect(mockedApi).toHaveBeenNthCalledWith(2, "/api/v1/auth/verify", {
      method: "POST",
      body: JSON.stringify({ challengeID: "challenge-1", nonce: "nonce-1", signature: "base64-signature" }),
    });
    expect(mockedApi).toHaveBeenNthCalledWith(3, "/api/v1/auth/session");
  });

  it("does not report authentication when the verified identity does not match", async () => {
    mockedApi
      .mockResolvedValueOnce({ id: "challenge-1", nonce: "nonce-1", message: "challenge" })
      .mockResolvedValueOnce({ address: "GOTHER", network: "testnet" });
    mockedSignMessage.mockResolvedValue("base64-signature");

    await expect(authenticateWallet("GTEST", network)).rejects.toThrow("The API authenticated a different wallet or network.");
    expect(mockedApi).toHaveBeenCalledTimes(2);
  });

  it("propagates a rejected Freighter challenge signature", async () => {
    mockedApi.mockResolvedValueOnce({ id: "challenge-1", nonce: "nonce-1", message: "challenge" });
    mockedSignMessage.mockRejectedValue(new Error("User rejected message signing"));

    await expect(authenticateWallet("GTEST", network)).rejects.toThrow("User rejected message signing");
    expect(mockedApi).toHaveBeenCalledTimes(1);
  });
});
