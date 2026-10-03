import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Account, nativeToScVal, rpc } from "@stellar/stellar-sdk";
import {
  assertMaintenanceFresh,
  buildMaintenanceTransaction,
  requireVerifiedTestnet,
  TESTNET_PASSPHRASE,
  VERIFIED_CONTROLLER_ID,
  VERIFIED_CONTROLLER_WASM_HASH,
  type MaintenanceSnapshot,
} from "./controller-maintenance";

const address = "GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6";
const snapshot: MaintenanceSnapshot = {
  ledger: 100,
  governanceEpoch: "1",
  controllerVersion: 1,
  accountSequence: "10",
  wasmHash: VERIFIED_CONTROLLER_WASM_HASH,
};

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_STELLAR_NETWORK", "testnet");
  vi.stubEnv("NEXT_PUBLIC_STELLAR_NETWORK_PASSPHRASE", TESTNET_PASSPHRASE);
  vi.stubEnv("NEXT_PUBLIC_CONTROLLER_ID", VERIFIED_CONTROLLER_ID);
  vi.stubEnv("NEXT_PUBLIC_STELLAR_RPC_URL", "https://soroban-testnet.stellar.org");
});
afterEach(() => vi.unstubAllEnvs());

function fakeServer(simulate: ReturnType<typeof vi.fn>) {
  return {
    getNetwork: vi.fn().mockResolvedValue({ passphrase: TESTNET_PASSPHRASE }),
    getLatestLedger: vi.fn().mockResolvedValue({ sequence: 100 }),
    getContractInstance: vi.fn().mockResolvedValue({
      executable: { type: "contractExecutableWasm", value: { value: Uint8Array.from(Buffer.from(VERIFIED_CONTROLLER_WASM_HASH, "hex")) } },
    }),
    getAccount: vi.fn().mockResolvedValue(new Account(address, "10")),
    simulateTransaction: simulate,
  } as unknown as rpc.Server;
}

describe("Testnet controller maintenance preflight", () => {
  it("refuses a wrong RPC network or changed controller code", () => {
    expect(() => requireVerifiedTestnet("Public Global Stellar Network ; September 2015", VERIFIED_CONTROLLER_WASM_HASH))
      .toThrow("Wrong network");
    expect(() => requireVerifiedTestnet(TESTNET_PASSPHRASE, "0".repeat(64)))
      .toThrow("live controller WASM differs");
  });

  it("refuses stale controller, account, and ledger state", () => {
    expect(() => assertMaintenanceFresh(snapshot, { ...snapshot, governanceEpoch: "2" })).toThrow("state changed");
    expect(() => assertMaintenanceFresh(snapshot, { ...snapshot, accountSequence: "11" })).toThrow("state changed");
    expect(() => assertMaintenanceFresh(snapshot, { ...snapshot, ledger: 121 })).toThrow("state changed");
  });

  it("stops before XDR is returned when the maintenance simulation fails", async () => {
    const simulate = vi.fn()
      .mockResolvedValueOnce({ transactionData: {}, result: { retval: nativeToScVal(1n, { type: "u64" }), auth: [] } })
      .mockResolvedValueOnce({ transactionData: {}, result: { retval: nativeToScVal(1, { type: "u32" }), auth: [] } })
      .mockResolvedValueOnce({ error: "host function failed" });
    await expect(buildMaintenanceTransaction(address, fakeServer(simulate)))
      .rejects.toThrow("Controller maintenance simulation failed: host function failed");
    expect(simulate).toHaveBeenCalledTimes(3);
  });
});
