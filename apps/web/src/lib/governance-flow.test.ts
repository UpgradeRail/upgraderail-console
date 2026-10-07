import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Account, Keypair, Operation, Networks, TransactionBuilder, Asset, rpc } from "@stellar/stellar-sdk";
import {
  DRAFT_MAX_AGE_MS,
  pollGovernance,
  refreshGovernanceState,
  submitGovernance,
  validateSignedGovernance,
} from "./governance-flow";
import {
  enumTag,
  TESTNET_PASSPHRASE,
  VERIFIED_CONTROLLER_ID,
  VERIFIED_CONTROLLER_WASM_HASH,
  type UnsignedGovernanceTx,
} from "./governance-tx";

const signer = Keypair.random();
const other = Keypair.random();

const hex = (bytes: Uint8Array) => Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
// Fixed time bounds keep the transaction hash identical across rebuilds.
function payment(amount: string) {
  return new TransactionBuilder(new Account(signer.publicKey(), "10"), {
    fee: "100", networkPassphrase: Networks.TESTNET, timebounds: { minTime: 0, maxTime: 4102444800 },
  })
    .addOperation(Operation.payment({ destination: other.publicKey(), asset: Asset.native(), amount }))
    .build();
}
const unsignedTx = () => payment("1");
function draftFor(tx: ReturnType<typeof unsignedTx> = unsignedTx(), builtAt = Date.now()): UnsignedGovernanceTx {
  return {
    action: "approve", address: signer.publicKey(), contractId: VERIFIED_CONTROLLER_ID, networkPassphrase: TESTNET_PASSPHRASE,
    rpcUrl: "https://soroban-testnet.stellar.org", unsignedXdr: tx.toXDR(), transactionHash: hex(tx.hash()), feeStroops: "100",
    snapshot: { ledger: 100, governanceEpoch: "1", controllerVersion: 1, accountSequence: "10", wasmHash: VERIFIED_CONTROLLER_WASM_HASH },
    builtAt,
  };
}
const signedXdr = () => { const tx = unsignedTx(); tx.sign(signer); return tx.toXDR(); };

beforeEach(() => {
  vi.stubEnv("NEXT_PUBLIC_STELLAR_NETWORK", "testnet");
  vi.stubEnv("NEXT_PUBLIC_CONTROLLER_ID", VERIFIED_CONTROLLER_ID);
  vi.stubEnv("NEXT_PUBLIC_STELLAR_RPC_URL", "https://soroban-testnet.stellar.org");
});
afterEach(() => vi.unstubAllEnvs());

describe("validateSignedGovernance", () => {
  it("accepts the exact transaction with a signature", () => {
    expect(() => validateSignedGovernance(draftFor(), signedXdr())).not.toThrow();
  });
  it("rejects an unsigned return", () => {
    expect(() => validateSignedGovernance(draftFor(), unsignedTx().toXDR())).toThrow("unsigned transaction");
  });
  it("rejects a transaction with different contents", () => {
    const changed = payment("2");
    changed.sign(signer);
    expect(() => validateSignedGovernance(draftFor(), changed.toXDR())).toThrow("different contents");
  });
});

describe("refreshGovernanceState", () => {
  it("refuses a draft older than two minutes without touching the network", async () => {
    const server = {} as rpc.Server;
    await expect(refreshGovernanceState(draftFor(unsignedTx(), 0), server, DRAFT_MAX_AGE_MS + 1)).rejects.toThrow("older than two minutes");
  });
});

describe("submitGovernance", () => {
  const draft = draftFor();
  const server = (response: object) => ({ sendTransaction: vi.fn().mockResolvedValue(response) }) as unknown as rpc.Server;

  it("returns pending with the matching hash", async () => {
    const s = server({ status: "PENDING", hash: draft.transactionHash });
    expect(await submitGovernance(draft, signedXdr(), s)).toEqual({ kind: "pending", hash: draft.transactionHash });
  });
  it("reports an RPC rejection as failed, not pending", async () => {
    const s = server({ status: "ERROR", hash: draft.transactionHash, errorResult: { result: { type: "txBadSeq" } } });
    expect(await submitGovernance(draft, signedXdr(), s)).toMatchObject({ kind: "failed", reason: expect.stringContaining("txBadSeq") });
  });
  it("reports TRY_AGAIN_LATER as failed", async () => {
    const s = server({ status: "TRY_AGAIN_LATER", hash: draft.transactionHash });
    expect(await submitGovernance(draft, signedXdr(), s)).toMatchObject({ kind: "failed" });
  });
  it("rejects an RPC hash that differs from the signed transaction", async () => {
    const s = server({ status: "PENDING", hash: "f".repeat(64) });
    await expect(submitGovernance(draft, signedXdr(), s)).rejects.toThrow("differs from the signed transaction");
  });
  it("never sends an invalid signed transaction", async () => {
    const s = server({ status: "PENDING", hash: draft.transactionHash });
    await expect(submitGovernance(draft, unsignedTx().toXDR(), s)).rejects.toThrow();
    expect((s.sendTransaction as ReturnType<typeof vi.fn>)).not.toHaveBeenCalled();
  });
});

describe("pollGovernance", () => {
  const hash = "ab".repeat(32);
  const poll = (response: object) => pollGovernance(hash, { getTransaction: vi.fn().mockResolvedValue({ txHash: hash, ...response }) });
  it("maps SUCCESS, FAILED and NOT_FOUND", async () => {
    expect(await poll({ status: "SUCCESS", ledger: 123 })).toEqual({ kind: "confirmed", hash, ledger: 123 });
    expect(await poll({ status: "FAILED", ledger: 124, resultXdr: { result: { type: "txFailed" } } })).toMatchObject({ kind: "failed", ledger: 124, reason: expect.stringContaining("txFailed") });
    expect(await poll({ status: "NOT_FOUND" })).toEqual({ kind: "pending", hash });
  });
  it("rejects a mismatched hash", async () => {
    await expect(pollGovernance(hash, { getTransaction: vi.fn().mockResolvedValue({ txHash: "cd".repeat(32), status: "SUCCESS", ledger: 1 }) })).rejects.toThrow("different transaction hash");
  });
});

describe("enumTag", () => {
  it("normalizes the shapes an enum unit variant can decode to", () => {
    expect(enumTag("Ready")).toBe("Ready");
    expect(enumTag(["Timelocked"])).toBe("Timelocked");
    expect(enumTag({ tag: "Expired" })).toBe("Expired");
    expect(enumTag(7)).toBeUndefined();
  });
});
