import { describe, expect, it } from "vitest";
import { Account, BASE_FEE, Contract, TransactionBuilder, xdr } from "@stellar/stellar-sdk";
import { TESTNET_PASSPHRASE, VERIFIED_CONTROLLER_ID, VERIFIED_CONTROLLER_WASM_HASH, type UnsignedMaintenance } from "./controller-maintenance";
import { validateSignedMaintenance } from "./controller-maintenance-signing";

const address = "GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6";

function draft(): UnsignedMaintenance {
  const tx = new TransactionBuilder(new Account(address, "10"), { fee: BASE_FEE, networkPassphrase: TESTNET_PASSPHRASE })
    .addOperation(new Contract(VERIFIED_CONTROLLER_ID).call("maintain_controller"))
    .setTimeout(300)
    .build();
  return {
    action: "maintain_controller",
    address,
    contractId: VERIFIED_CONTROLLER_ID,
    networkPassphrase: TESTNET_PASSPHRASE,
    rpcUrl: "https://soroban-testnet.stellar.org",
    unsignedXdr: tx.toXDR(),
    transactionHash: Buffer.from(tx.hash()).toString("hex"),
    feeStroops: tx.fee,
    snapshot: { ledger: 100, governanceEpoch: "1", controllerVersion: 1, accountSequence: "10", wasmHash: VERIFIED_CONTROLLER_WASM_HASH },
    builtAt: Date.now(),
  };
}

describe("Freighter maintenance signing result", () => {
  it("accepts a signed copy of the exact unsigned transaction without a private key", () => {
    const built = draft();
    const signed = TransactionBuilder.fromXDR(built.unsignedXdr, TESTNET_PASSPHRASE);
    signed.signatures.push(new xdr.DecoratedSignature({ hint: Buffer.alloc(4), signature: Buffer.alloc(64) }));
    expect(() => validateSignedMaintenance(built, signed.toXDR())).not.toThrow();
  });

  it("rejects missing signatures and changed transaction contents", () => {
    const built = draft();
    expect(() => validateSignedMaintenance(built, built.unsignedXdr)).toThrow("unsigned transaction");
    const changed = draft();
    changed.transactionHash = "0".repeat(64);
    expect(() => validateSignedMaintenance(changed, built.unsignedXdr)).toThrow("different contents");
  });
});
