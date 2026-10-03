import { describe, expect, it, vi } from "vitest";
import { Account, BASE_FEE, Contract, TransactionBuilder, rpc, xdr } from "@stellar/stellar-sdk";
import { TESTNET_PASSPHRASE, VERIFIED_CONTROLLER_ID, VERIFIED_CONTROLLER_WASM_HASH, type UnsignedMaintenance } from "./controller-maintenance";
import { pollMaintenance, submitMaintenance } from "./controller-maintenance-submit";

const address = "GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6";
function signedDraft() {
  const tx = new TransactionBuilder(new Account(address, "10"), { fee: BASE_FEE, networkPassphrase: TESTNET_PASSPHRASE })
    .addOperation(new Contract(VERIFIED_CONTROLLER_ID).call("maintain_controller"))
    .setTimeout(300).build();
  const draft: UnsignedMaintenance = {
    action: "maintain_controller", address, contractId: VERIFIED_CONTROLLER_ID,
    networkPassphrase: TESTNET_PASSPHRASE, rpcUrl: "https://soroban-testnet.stellar.org",
    unsignedXdr: tx.toXDR(), transactionHash: Buffer.from(tx.hash()).toString("hex"), feeStroops: tx.fee,
    snapshot: { ledger: 100, governanceEpoch: "1", controllerVersion: 1, accountSequence: "10", wasmHash: VERIFIED_CONTROLLER_WASM_HASH },
    builtAt: Date.now(),
  };
  tx.signatures.push(new xdr.DecoratedSignature({ hint: Buffer.alloc(4), signature: Buffer.alloc(64) }));
  return { draft, signedXdr: tx.toXDR() };
}

describe("Testnet maintenance submission lifecycle", () => {
  it("does not send an unsigned transaction", async () => {
    const { draft } = signedDraft();
    const sendTransaction = vi.fn();
    await expect(submitMaintenance(draft, draft.unsignedXdr, { sendTransaction } as unknown as rpc.Server)).rejects.toThrow("unsigned transaction");
    expect(sendTransaction).not.toHaveBeenCalled();
  });

  it("reports RPC rejection with its result code", async () => {
    const { draft, signedXdr } = signedDraft();
    const sendTransaction = vi.fn().mockResolvedValue({ status: "ERROR", hash: draft.transactionHash, errorResult: { result: { type: "txBadSeq" } } });
    await expect(submitMaintenance(draft, signedXdr, { sendTransaction } as unknown as rpc.Server)).resolves.toEqual({ kind: "failed", hash: draft.transactionHash, reason: "RPC rejected the transaction: txBadSeq" });
  });

  it("reports pending, confirmation, and failure by the same public hash", async () => {
    const { draft, signedXdr } = signedDraft();
    const sendTransaction = vi.fn().mockResolvedValue({ status: "PENDING", hash: draft.transactionHash });
    expect(await submitMaintenance(draft, signedXdr, { sendTransaction } as unknown as rpc.Server)).toEqual({ kind: "pending", hash: draft.transactionHash });
    const getTransaction = vi.fn()
      .mockResolvedValueOnce({ status: "NOT_FOUND", txHash: draft.transactionHash })
      .mockResolvedValueOnce({ status: "SUCCESS", txHash: draft.transactionHash, ledger: 101 })
      .mockResolvedValueOnce({ status: "FAILED", txHash: draft.transactionHash, ledger: 102, resultXdr: { result: { type: "txFailed" } } });
    const server = { getTransaction } as unknown as rpc.Server;
    expect(await pollMaintenance(draft.transactionHash, server)).toEqual({ kind: "pending", hash: draft.transactionHash });
    expect(await pollMaintenance(draft.transactionHash, server)).toEqual({ kind: "confirmed", hash: draft.transactionHash, ledger: 101 });
    expect(await pollMaintenance(draft.transactionHash, server)).toEqual({ kind: "failed", hash: draft.transactionHash, ledger: 102, reason: "Transaction failed: txFailed" });
  });
});
