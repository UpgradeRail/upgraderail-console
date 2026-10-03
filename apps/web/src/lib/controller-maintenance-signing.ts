import { Transaction, TransactionBuilder } from "@stellar/stellar-sdk";
import { TESTNET_PASSPHRASE, type UnsignedMaintenance } from "./controller-maintenance";

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export function validateSignedMaintenance(draft: UnsignedMaintenance, signedXdr: string): void {
  const unsigned = TransactionBuilder.fromXDR(draft.unsignedXdr, TESTNET_PASSPHRASE);
  const signed = TransactionBuilder.fromXDR(signedXdr, TESTNET_PASSPHRASE);
  if (!(unsigned instanceof Transaction) || !(signed instanceof Transaction)) {
    throw new Error("Freighter returned a fee-bump transaction instead of controller maintenance.");
  }
  if (signed.source !== draft.address || signed.sequence !== unsigned.sequence ||
      toHex(unsigned.hash()) !== draft.transactionHash ||
      toHex(signed.hash()) !== draft.transactionHash) {
    throw new Error("Freighter returned a transaction with different contents. Rebuild before signing.");
  }
  if (signed.signatures.length === 0) {
    throw new Error("Freighter returned an unsigned transaction.");
  }
}
