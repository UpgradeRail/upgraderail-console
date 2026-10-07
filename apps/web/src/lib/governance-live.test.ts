import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Account, Operation, nativeToScVal, rpc, xdr, type Transaction } from "@stellar/stellar-sdk";
import {
  controllerSpec,
  readLiveProposal,
  TESTNET_PASSPHRASE,
  VERIFIED_CONTROLLER_ID,
  VERIFIED_CONTROLLER_WASM_HASH,
} from "./governance-tx";

const address = "GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6";
const proposer = "GAAZI4TCR3TY5OJHCTJC2A4QSY6CJWJH5IAJTGKIN2ER7LBNVKOCCWN7";

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_STELLAR_NETWORK", "testnet");
  vi.stubEnv("NEXT_PUBLIC_CONTROLLER_ID", VERIFIED_CONTROLLER_ID);
  vi.stubEnv("NEXT_PUBLIC_STELLAR_RPC_URL", "https://soroban-testnet.stellar.org");
});
afterEach(() => vi.unstubAllEnvs());

function encode(method: string, value: unknown): xdr.ScVal {
  const output = controllerSpec.getFunc(method).outputs[0] as unknown as { result?: { okType: xdr.ScSpecTypeDef } };
  return controllerSpec.nativeToScVal(value, output.result?.okType ?? (output as unknown as xdr.ScSpecTypeDef));
}

function liveServer(opts: { state: string; approvals: boolean; extra?: Record<string, unknown> }) {
  const hash = Buffer.alloc(32, 1);
  const proposal = {
    approval_count: 2, approved_ledger: 40, created_ledger: 10, execute_after_ledger: 90, expires_ledger: 400,
    governance_epoch: 1n, id: 5n, proposer, status: { tag: "Active", values: undefined },
    kind: { tag: "UpgradeFleet", values: [{ fleet_id: hash, expected_wasm_hash: hash, new_wasm_hash: hash, manifest_hash: hash }] },
    ...opts.extra,
  };
  const policy = { approvers: [address, proposer], proposal_lifetime_ledgers: 100, threshold: 2, timelock_ledgers: 50 };
  const retvals: Record<string, xdr.ScVal> = {
    get_governance_epoch: nativeToScVal(1n, { type: "u64" }),
    get_controller_version: nativeToScVal(1, { type: "u32" }),
    get_proposal_state: encode("get_proposal_state", { tag: opts.state, values: undefined }),
    get_proposal: encode("get_proposal", proposal),
    get_policy: encode("get_policy", policy),
    has_approved: nativeToScVal(opts.approvals, { type: "bool" }),
  };
  const simulate = vi.fn<(tx: Transaction) => Promise<unknown>>(async (tx) => {
    const op = tx.operations[0] as Operation.InvokeHostFunction;
    const name = String((op.func.value as xdr.InvokeContractArgs).functionName);
    return { _parsed: true, latestLedger: 100, minResourceFee: "1", transactionData: {}, result: { retval: retvals[name], auth: [] } };
  });
  return {
    simulate,
    server: {
      getNetwork: vi.fn().mockResolvedValue({ passphrase: TESTNET_PASSPHRASE }),
      getLatestLedger: vi.fn().mockResolvedValue({ sequence: 70 }),
      getContractInstance: vi.fn().mockResolvedValue({
        executable: { type: "contractExecutableWasm", value: { value: Uint8Array.from(Buffer.from(VERIFIED_CONTROLLER_WASM_HASH, "hex")) } },
      }),
      getAccount: vi.fn().mockResolvedValue(new Account(address, "10")),
      simulateTransaction: simulate,
    } as unknown as rpc.Server,
  };
}

describe("readLiveProposal", () => {
  it("reads state, approvals, threshold, timelock, and membership read-only", async () => {
    const { server, simulate } = liveServer({ state: "Timelocked", approvals: true });
    const live = await readLiveProposal(server, address, 5);
    expect(live).toEqual({
      ledger: 70, state: "Timelocked", proposer, approvalCount: 2, threshold: 2, timelockLedgers: 50,
      executeAfterLedger: 90, expiresLedger: 400, isApprover: true, hasApproved: true,
      kind: "UpgradeFleet", manifestHash: "01".repeat(32), expectedWasmHash: "01".repeat(32), newWasmHash: "01".repeat(32),
    });
    // Only simulations: nothing is signed or sent.
    expect(simulate).toHaveBeenCalled();
    expect((server as unknown as { sendTransaction?: unknown }).sendTransaction).toBeUndefined();
  });

  it("reports a missing execute-after ledger as null, not zero", async () => {
    const { server } = liveServer({ state: "AwaitingApprovals", approvals: false, extra: { execute_after_ledger: undefined, approved_ledger: undefined, approval_count: 1 } });
    const live = await readLiveProposal(server, address, 5);
    expect(live.executeAfterLedger).toBeNull();
    expect(live.state).toBe("AwaitingApprovals");
    expect(live.hasApproved).toBe(false);
  });

  it("surfaces a failed read simulation instead of guessing a state", async () => {
    const { server, simulate } = liveServer({ state: "Ready", approvals: false });
    simulate.mockImplementation(async (tx: Transaction) => {
      const name = String(((tx.operations[0] as Operation.InvokeHostFunction).func.value as xdr.InvokeContractArgs).functionName);
      if (name === "get_proposal_state") return { error: "HostError: Error(Contract, #3)" };
      return { _parsed: true, latestLedger: 100, minResourceFee: "1", transactionData: {}, result: { retval: nativeToScVal(1n, { type: name === "get_controller_version" ? "u32" : "u64" }), auth: [] } };
    });
    await expect(readLiveProposal(server, address, 5)).rejects.toThrow("get_proposal_state failed");
  });
});

describe("describeProposalKind", () => {
  it("reads kind and hashes from each payload shape and never invents them", async () => {
    const { describeProposalKind } = await import("./governance-tx");
    const b = (n: number) => Uint8Array.from(Buffer.alloc(32, n));
    expect(describeProposalKind({ tag: "CreateFleet", values: [{ manifest_hash: b(2), initial_wasm_hash: b(3) }] }))
      .toEqual({ kind: "CreateFleet", manifestHash: "02".repeat(32), expectedWasmHash: null, newWasmHash: "03".repeat(32) });
    expect(describeProposalKind({ tag: "UpdatePolicy", values: [{ policy: {} }] }))
      .toEqual({ kind: "UpdatePolicy", manifestHash: null, expectedWasmHash: null, newWasmHash: null });
    expect(describeProposalKind(undefined)).toEqual({ kind: null, manifestHash: null, expectedWasmHash: null, newWasmHash: null });
  });
});
