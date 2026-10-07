import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Account, Address, Operation, SorobanDataBuilder, Transaction, TransactionBuilder, nativeToScVal, rpc, xdr } from "@stellar/stellar-sdk";
import {
  assertStateFresh,
  buildApprove,
  buildCancelProposal,
  buildCreateProposal,
  buildExecuteProposal,
  buildRevokeApproval,
  TESTNET_PASSPHRASE,
  VERIFIED_CONTROLLER_ID,
  VERIFIED_CONTROLLER_WASM_HASH,
  type GovernanceSnapshot,
  type ProposalKindInput,
} from "./governance-tx";

const address = "GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6";
const hash = Buffer.alloc(32, 7);
const kind: ProposalKindInput = {
  tag: "UpgradeFleet",
  values: [
    {
      fleet_id: hash,
      expected_wasm_hash: hash,
      new_wasm_hash: Buffer.alloc(32, 9),
      manifest_hash: hash,
    },
  ],
};

const builders = [
  ["create_proposal", (s: rpc.Server) => buildCreateProposal(address, kind, s)],
  ["approve", (s: rpc.Server) => buildApprove(address, 4n, s)],
  ["revoke_approval", (s: rpc.Server) => buildRevokeApproval(address, 4, s)],
  ["cancel_proposal", (s: rpc.Server) => buildCancelProposal(address, "4", s)],
  ["execute_proposal", (s: rpc.Server) => buildExecuteProposal(address, 4n, s)],
] as const;

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_STELLAR_NETWORK", "testnet");
  vi.stubEnv("NEXT_PUBLIC_CONTROLLER_ID", VERIFIED_CONTROLLER_ID);
  vi.stubEnv("NEXT_PUBLIC_STELLAR_RPC_URL", "https://soroban-testnet.stellar.org");
});
afterEach(() => {
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});

function readOk(value: number | bigint, type: "u64" | "u32") {
  return {
    transactionData: {},
    result: { retval: nativeToScVal(value, { type }), auth: [] },
  };
}

function fakeServer(
  simulate: ReturnType<typeof vi.fn>,
  opts: { passphrase?: string; wasmHash?: string } = {}
) {
  const wasmHash = opts.wasmHash ?? VERIFIED_CONTROLLER_WASM_HASH;
  return {
    serverURL: new URL("https://soroban-testnet.stellar.org"),
    getNetwork: vi.fn().mockResolvedValue({ passphrase: opts.passphrase ?? TESTNET_PASSPHRASE }),
    getLatestLedger: vi.fn().mockResolvedValue({ sequence: 100 }),
    getContractInstance: vi.fn().mockResolvedValue({
      executable: {
        type: "contractExecutableWasm",
        value: { value: Uint8Array.from(Buffer.from(wasmHash, "hex")) },
      },
    }),
    getAccount: vi.fn().mockResolvedValue(new Account(address, "10")),
    simulateTransaction: simulate,
  } as unknown as rpc.Server;
}

describe.each(builders)("%s builder", (action, build) => {
  it("rejects a wrong network before simulating the write", async () => {
    const simulate = vi.fn();
    const server = fakeServer(simulate, {
      passphrase: "Public Global Stellar Network ; September 2015",
    });
    await expect(build(server)).rejects.toThrow("Wrong network");
    expect(simulate).not.toHaveBeenCalled();
  });

  it("rejects a controller whose live WASM differs from the verified record", async () => {
    const simulate = vi.fn();
    await expect(build(fakeServer(simulate, { wasmHash: "0".repeat(64) })))
      .rejects.toThrow("live controller WASM differs");
    expect(simulate).not.toHaveBeenCalled();
  });

  it("returns no XDR when the write simulation fails", async () => {
    const simulate = vi
      .fn()
      .mockResolvedValueOnce(readOk(1n, "u64"))
      .mockResolvedValueOnce(readOk(1, "u32"))
      .mockResolvedValueOnce({ error: "HostError: Error(Contract, #7)" });
    await expect(build(fakeServer(simulate))).rejects.toThrow(
      /simulation failed: HostError: Error\(Contract, #7\)/
    );
  });

  it("builds an unsigned transaction invoking the controller method", async () => {
    const simulate = vi
      .fn()
      .mockResolvedValueOnce(readOk(1n, "u64"))
      .mockResolvedValueOnce(readOk(1, "u32"))
      .mockResolvedValueOnce({
        _parsed: true,
        latestLedger: 100,
        minResourceFee: "1000",
        transactionData: new SorobanDataBuilder(),
        result: { retval: xdr.ScVal.scvVoid(), auth: [] },
      });

    const result = await build(fakeServer(simulate));

    expect(result.action).toBe(action);
    expect(result.contractId).toBe(VERIFIED_CONTROLLER_ID);
    expect(result.networkPassphrase).toBe(TESTNET_PASSPHRASE);
    expect(result.address).toBe(address);
    expect(result.snapshot).toMatchObject({
      governanceEpoch: "1",
      controllerVersion: 1,
      accountSequence: "10",
      wasmHash: VERIFIED_CONTROLLER_WASM_HASH,
    });
    expect(result.feeStroops).toMatch(/^\d+$/);
    const tx = TransactionBuilder.fromXDR(result.unsignedXdr, TESTNET_PASSPHRASE) as Transaction;
    expect(tx.signatures).toHaveLength(0);
    expect(tx.networkPassphrase ?? TESTNET_PASSPHRASE).toBe(TESTNET_PASSPHRASE);
    const invoke = (tx.operations[0] as Operation.InvokeHostFunction).func.value as xdr.InvokeContractArgs;
    expect(String(invoke.functionName)).toBe(action);
    expect(Address.fromScAddress(invoke.contractAddress).toString()).toBe(
      VERIFIED_CONTROLLER_ID
    );
  });
});

describe("assertStateFresh", () => {
  const snapshot: GovernanceSnapshot = {
    ledger: 100,
    governanceEpoch: "1",
    controllerVersion: 1,
    accountSequence: "10",
    wasmHash: VERIFIED_CONTROLLER_WASM_HASH,
  };

  it("accepts an unchanged snapshot", () => {
    expect(() => assertStateFresh(snapshot, { ...snapshot, ledger: 110 })).not.toThrow();
  });

  it.each([
    { governanceEpoch: "2" },
    { controllerVersion: 2 },
    { accountSequence: "11" },
    { wasmHash: "0".repeat(64) },
    { ledger: 121 },
    { ledger: 99 },
  ])("refuses stale state %o", (change) => {
    expect(() => assertStateFresh(snapshot, { ...snapshot, ...change })).toThrow("state changed");
  });
});
