import {
  Account,
  BASE_FEE,
  Contract,
  TransactionBuilder,
  rpc,
  scValToNative,
} from "@stellar/stellar-sdk";
import { Spec } from "@stellar/stellar-sdk/contract";

export const TESTNET_PASSPHRASE = "Test SDF Network ; September 2015" as const;
export const VERIFIED_CONTROLLER_ID =
  "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3";
export const VERIFIED_CONTROLLER_WASM_HASH =
  "0ac2626c26b43f5330bf89cfdd952ecdd9940817a703ec04c327f478f5f20458";

export const CONTROLLER_SPEC_ENTRIES = [
  "AAAAAAAAAAAAAAAHYXBwcm92ZQAAAAACAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAAAAAAhhcHByb3ZlcgAAABMAAAABAAAD6QAAAAIAAAfQAAAADUNvbnRyYWN0RXJyb3IAAAA=",
  "AAAAAAAAAAAAAAAJZ2V0X2ZsZWV0AAAAAAAAAQAAAAAAAAAIZmxlZXRfaWQAAAPuAAAAIAAAAAEAAAPpAAAH0AAAAAVGbGVldAAAAAAAB9AAAAANQ29udHJhY3RFcnJvcgAAAA==",
  "AAAAAAAAAAAAAAAKZ2V0X3BvbGljeQAAAAAAAAAAAAEAAAfQAAAAEEdvdmVybmFuY2VQb2xpY3k=",
  "AAAAAAAAAAAAAAAMZ2V0X3Byb3Bvc2FsAAAAAQAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAEAAAPpAAAH0AAAAAhQcm9wb3NhbAAAB9AAAAANQ29udHJhY3RFcnJvcgAAAA==",
  "AAAAAAAAAAAAAAAMaGFzX2FwcHJvdmVkAAAAAgAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAAAAAAIYXBwcm92ZXIAAAATAAAAAQAAAAE=",
  "AAAAAAAAAAAAAAANX19jb25zdHJ1Y3RvcgAAAAAAAAEAAAAAAAAABnBvbGljeQAAAAAH0AAAABBHb3Zlcm5hbmNlUG9saWN5AAAAAQAAA+kAAAACAAAH0AAAAA1Db250cmFjdEVycm9yAAAA",
  "AAAAAAAAAAAAAAAPY2FuY2VsX3Byb3Bvc2FsAAAAAAIAAAAAAAAAC3Byb3Bvc2FsX2lkAAAAAAYAAAAAAAAACHByb3Bvc2VyAAAAEwAAAAEAAAPpAAAAAgAAB9AAAAANQ29udHJhY3RFcnJvcgAAAA==",
  "AAAAAAAAAAAAAAAPY3JlYXRlX3Byb3Bvc2FsAAAAAAIAAAAAAAAACHByb3Bvc2VyAAAAEwAAAAAAAAAEa2luZAAAB9AAAAAMUHJvcG9zYWxLaW5kAAAAAQAAA+kAAAAGAAAH0AAAAA1Db250cmFjdEVycm9yAAAA",
  "AAAAAAAAAAAAAAAPcmV2b2tlX2FwcHJvdmFsAAAAAAIAAAAAAAAAC3Byb3Bvc2FsX2lkAAAAAAYAAAAAAAAACGFwcHJvdmVyAAAAEwAAAAEAAAPpAAAAAgAAB9AAAAANQ29udHJhY3RFcnJvcgAAAA==",
  "AAAAAAAAAAAAAAAQZXhlY3V0ZV9wcm9wb3NhbAAAAAEAAAAAAAAAC3Byb3Bvc2FsX2lkAAAAAAYAAAABAAAD6QAAAAIAAAfQAAAADUNvbnRyYWN0RXJyb3IAAAA=",
  "AAAAAAAAAAAAAAAQZ2V0X2N1cnJlbnRfd2FzbQAAAAEAAAAAAAAACGZsZWV0X2lkAAAD7gAAACAAAAABAAAD6QAAA+4AAAAgAAAH0AAAAA1Db250cmFjdEVycm9yAAAA",
  "AAAAAAAAAAAAAAASZ2V0X3Byb3Bvc2FsX3N0YXRlAAAAAAABAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAQAAA+kAAAfQAAAADVByb3Bvc2FsU3RhdGUAAAAAAAfQAAAADUNvbnRyYWN0RXJyb3IAAAA=",
  "AAAAAAAAAAAAAAAUZ2V0X2dvdmVybmFuY2VfZXBvY2gAAAAAAAAAAQAAAAY=",
  "AAAAAAAAAAAAAAAWZ2V0X2NvbnRyb2xsZXJfdmVyc2lvbgAAAAAAAAAAAAEAAAAE",
  "AAAAAQAAAAAAAAAAAAAABUZsZWV0AAAAAAAAAwAAAAAAAAAOY3JlYXRlZF9sZWRnZXIAAAAAAAQAAAAAAAAAAmlkAAAAAAPuAAAAIAAAAAAAAAADdGFnAAAAABA=",
  "AAAAAQAAAAAAAAAAAAAACFByb3Bvc2FsAAAACgAAAAAAAAAOYXBwcm92YWxfY291bnQAAAAAAAQAAAAAAAAAD2FwcHJvdmVkX2xlZGdlcgAAAAPoAAAABAAAAAAAAAAOY3JlYXRlZF9sZWRnZXIAAAAAAAQAAAAAAAAAFGV4ZWN1dGVfYWZ0ZXJfbGVkZ2VyAAAD6AAAAAQAAAAAAAAADmV4cGlyZXNfbGVkZ2VyAAAAAAAEAAAAAAAAABBnb3Zlcm5hbmNlX2Vwb2NoAAAABgAAAAAAAAACaWQAAAAAAAYAAAAAAAAABGtpbmQAAAfQAAAADFByb3Bvc2FsS2luZAAAAAAAAAAIcHJvcG9zZXIAAAATAAAAAAAAAAZzdGF0dXMAAAAAB9AAAAAUU3RvcmVkUHJvcG9zYWxTdGF0dXM=",
  "AAAAAgAAAAAAAAAAAAAADFByb3Bvc2FsS2luZAAAAAQAAAABAAAAAAAAAAtDcmVhdGVGbGVldAAAAAABAAAH0AAAABNDcmVhdGVGbGVldFByb3Bvc2FsAAAAAAEAAAAAAAAADFVwZ3JhZGVGbGVldAAAAAEAAAfQAAAAFFVwZ3JhZGVGbGVldFByb3Bvc2FsAAAAAQAAAAAAAAAMVXBkYXRlUG9saWN5AAAAAQAAB9AAAAAUVXBkYXRlUG9saWN5UHJvcG9zYWwAAAABAAAAAAAAABFVcGdyYWRlQ29udHJvbGxlcgAAAAAAAAEAAAfQAAAAGVVwZ3JhZGVDb250cm9sbGVyUHJvcG9zYWwAAAA=",
  "AAAAAgAAAAAAAAAAAAAADVByb3Bvc2FsU3RhdGUAAAAAAAAHAAAAAAAAAAAAAAARQXdhaXRpbmdBcHByb3ZhbHMAAAAAAAAAAAAAAAAAAApUaW1lbG9ja2VkAAAAAAAAAAAAAAAAAAVSZWFkeQAAAAAAAAAAAAAAAAAAB0V4cGlyZWQAAAAAAAAAAAAAAAAFU3RhbGUAAAAAAAAAAAAAAAAAAAhFeGVjdXRlZAAAAAAAAAAAAAAACUNhbmNlbGxlZAAAAA==",
  "AAAAAQAAAAAAAAAAAAAAEEdvdmVybmFuY2VQb2xpY3kAAAAEAAAAAAAAAAlhcHByb3ZlcnMAAAAAAAPqAAAAEwAAAAAAAAAZcHJvcG9zYWxfbGlmZXRpbWVfbGVkZ2VycwAAAAAAAAQAAAAAAAAACXRocmVzaG9sZAAAAAAAAAQAAAAAAAAAEHRpbWVsb2NrX2xlZGdlcnMAAAAE",
  "AAAAAQAAAAAAAAAAAAAAE0NyZWF0ZUZsZWV0UHJvcG9zYWwAAAAABAAAAAAAAAAIZmxlZXRfaWQAAAPuAAAAIAAAAAAAAAARaW5pdGlhbF93YXNtX2hhc2gAAAAAAAPuAAAAIAAAAAAAAAANbWFuaWZlc3RfaGFzaAAAAAAAA+4AAAAgAAAAAAAAAAN0YWcAAAAAEA==",
  "AAAAAgAAAAAAAAAAAAAAFFN0b3JlZFByb3Bvc2FsU3RhdHVzAAAAAwAAAAAAAAAAAAAABkFjdGl2ZQAAAAAAAAAAAAAAAAAIRXhlY3V0ZWQAAAAAAAAAAAAAAAlDYW5jZWxsZWQAAAA=",
  "AAAAAQAAAAAAAAAAAAAAFFVwZGF0ZVBvbGljeVByb3Bvc2FsAAAAAQAAAAAAAAAGcG9saWN5AAAAAAfQAAAAEEdvdmVybmFuY2VQb2xpY3k=",
  "AAAAAQAAAAAAAAAAAAAAFFVwZ3JhZGVGbGVldFByb3Bvc2FsAAAABAAAAAAAAAASZXhwZWN0ZWRfd2FzbV9oYXNoAAAAAAPuAAAAIAAAAAAAAAAIZmxlZXRfaWQAAAPuAAAAIAAAAAAAAAANbWFuaWZlc3RfaGFzaAAAAAAAA+4AAAAgAAAAAAAAAA1uZXdfd2FzbV9oYXNoAAAAAAAD7gAAACA=",
  "AAAAAQAAAAAAAAAAAAAAGVVwZ3JhZGVDb250cm9sbGVyUHJvcG9zYWwAAAAAAAAEAAAAAAAAABtleHBlY3RlZF9jb250cm9sbGVyX3ZlcnNpb24AAAAABAAAAAAAAAANbWFuaWZlc3RfaGFzaAAAAAAAA+4AAAAgAAAAAAAAABZuZXdfY29udHJvbGxlcl92ZXJzaW9uAAAAAAAEAAAAAAAAAA1uZXdfd2FzbV9oYXNoAAAAAAAD7gAAACA=",
];

export const controllerSpec = new Spec(CONTROLLER_SPEC_ENTRIES);

export type GovernanceSnapshot = {
  ledger: number;
  governanceEpoch: string;
  controllerVersion: number;
  accountSequence: string;
  wasmHash: string;
};

export type UnsignedGovernanceTx = {
  action:
    | "create_proposal"
    | "approve"
    | "revoke_approval"
    | "cancel_proposal"
    | "execute_proposal";
  address: string;
  contractId: string;
  networkPassphrase: typeof TESTNET_PASSPHRASE;
  rpcUrl: string;
  unsignedXdr: string;
  transactionHash: string;
  feeStroops: string;
  snapshot: GovernanceSnapshot;
  builtAt: number;
};

export type CreateFleetProposalPayload = {
  fleet_id: Buffer | Uint8Array;
  initial_wasm_hash: Buffer | Uint8Array;
  manifest_hash: Buffer | Uint8Array;
  tag: string;
};

export type UpgradeFleetProposalPayload = {
  fleet_id: Buffer | Uint8Array;
  expected_wasm_hash: Buffer | Uint8Array;
  new_wasm_hash: Buffer | Uint8Array;
  manifest_hash: Buffer | Uint8Array;
};

export type UpdatePolicyProposalPayload = {
  policy: {
    approvers: string[];
    threshold: number;
    timelock_ledgers: number;
    proposal_lifetime_ledgers: number;
  };
};

export type UpgradeControllerProposalPayload = {
  expected_controller_version: number;
  new_controller_version: number;
  new_wasm_hash: Buffer | Uint8Array;
  manifest_hash: Buffer | Uint8Array;
};

export type ProposalKindInput =
  | { tag: "CreateFleet"; values: [CreateFleetProposalPayload] }
  | { tag: "UpgradeFleet"; values: [UpgradeFleetProposalPayload] }
  | { tag: "UpdatePolicy"; values: [UpdatePolicyProposalPayload] }
  | { tag: "UpgradeController"; values: [UpgradeControllerProposalPayload] };

export function defaultRpc(): rpc.Server {
  const endpoint = process.env.NEXT_PUBLIC_STELLAR_RPC_URL?.trim();
  if (!endpoint || new URL(endpoint).protocol !== "https:") {
    throw new Error("Configure an HTTPS Testnet RPC endpoint before building a transaction.");
  }
  return new rpc.Server(endpoint);
}

export function configuredContractId(): string {
  const configured =
    process.env.NEXT_PUBLIC_CONTROLLER_ID?.trim() || VERIFIED_CONTROLLER_ID;
  if (configured !== VERIFIED_CONTROLLER_ID) {
    throw new Error(
      "The configured controller does not match the verified Testnet deployment record."
    );
  }
  return configured;
}

export function requireVerifiedNetwork(
  rpcPassphrase: string,
  wasmHash: string
): void {
  const envNetwork = process.env.NEXT_PUBLIC_STELLAR_NETWORK?.toLowerCase();
  if (
    (envNetwork && envNetwork !== "testnet") ||
    rpcPassphrase !== TESTNET_PASSPHRASE
  ) {
    throw new Error(
      "Wrong network: governance transactions are available only on Stellar Testnet."
    );
  }
  if (wasmHash !== VERIFIED_CONTROLLER_WASM_HASH) {
    throw new Error(
      "The live controller WASM differs from the verified deployment record."
    );
  }
}

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function simulateRead(
  server: rpc.Server,
  source: Awaited<ReturnType<rpc.Server["getAccount"]>>,
  method: "get_governance_epoch" | "get_controller_version",
  contractId: string
): Promise<bigint> {
  const simulationSource = new Account(
    source.accountId(),
    source.sequenceNumber()
  );
  const tx = new TransactionBuilder(simulationSource, {
    fee: BASE_FEE,
    networkPassphrase: TESTNET_PASSPHRASE,
  })
    .addOperation(new Contract(contractId).call(method))
    .setTimeout(60)
    .build();
  const result = await server.simulateTransaction(tx);
  if (!rpc.Api.isSimulationSuccess(result) || !result.result) {
    throw new Error(
      `Live controller ${method} simulation failed: ${
        "error" in result ? result.error : "no result returned"
      }`
    );
  }
  const value: unknown = scValToNative(result.result.retval);
  if (
    typeof value !== "bigint" &&
    (typeof value !== "number" || !Number.isSafeInteger(value))
  ) {
    throw new Error(`Live controller ${method} returned an invalid value.`);
  }
  return BigInt(value);
}

export async function readLiveGovernanceState(
  server: rpc.Server,
  address: string
) {
  const contractId = configuredContractId();
  const [network, ledger, instance, account] = await Promise.all([
    server.getNetwork(),
    server.getLatestLedger(),
    server.getContractInstance(contractId),
    server.getAccount(address),
  ]);
  const executable = instance.executable;
  const wasmHash =
    executable.type === "contractExecutableWasm"
      ? toHex(executable.value.value)
      : "unsupported";
  requireVerifiedNetwork(network.passphrase, wasmHash);
  const [epoch, version] = await Promise.all([
    simulateRead(server, account, "get_governance_epoch", contractId),
    simulateRead(server, account, "get_controller_version", contractId),
  ]);
  if (version < 0 || version > 0xffffffffn) {
    throw new Error("The live controller version is out of range.");
  }
  return {
    account,
    contractId,
    networkPassphrase: TESTNET_PASSPHRASE,
    snapshot: {
      ledger: ledger.sequence,
      governanceEpoch: epoch.toString(),
      controllerVersion: Number(version),
      accountSequence: account.sequenceNumber(),
      wasmHash,
    } satisfies GovernanceSnapshot,
  };
}

export function assertStateFresh(
  previous: GovernanceSnapshot,
  current: GovernanceSnapshot
): void {
  if (
    current.ledger < previous.ledger ||
    current.ledger - previous.ledger > 20 ||
    current.wasmHash !== previous.wasmHash ||
    current.governanceEpoch !== previous.governanceEpoch ||
    current.controllerVersion !== previous.controllerVersion ||
    current.accountSequence !== previous.accountSequence
  ) {
    throw new Error(
      "Controller or account state changed since simulation. Build a new unsigned transaction."
    );
  }
}

/**
 * Builds an unsigned create_proposal governance transaction.
 * Reads fresh live state, converts proposal arguments via contract spec,
 * simulates on RPC, and returns assembled unsigned XDR.
 */
export async function buildCreateProposal(
  proposerAddress: string,
  kind: ProposalKindInput,
  server = defaultRpc()
): Promise<UnsignedGovernanceTx> {
  const { account, contractId, networkPassphrase, snapshot } =
    await readLiveGovernanceState(server, proposerAddress);

  const scVals = controllerSpec.funcArgsToScVals("create_proposal", {
    proposer: proposerAddress,
    kind,
  });

  const unsigned = new TransactionBuilder(account, {
    fee: BASE_FEE,
    networkPassphrase,
  })
    .addOperation(new Contract(contractId).call("create_proposal", ...scVals))
    .setTimeout(300)
    .build();

  const simulation = await server.simulateTransaction(unsigned);
  if (!rpc.Api.isSimulationSuccess(simulation) || !simulation.result) {
    throw new Error(
      `Create proposal simulation failed: ${
        "error" in simulation ? simulation.error : "no result returned"
      }`
    );
  }

  const prepared = rpc.assembleTransaction(unsigned, simulation).build();
  return {
    action: "create_proposal",
    address: proposerAddress,
    contractId,
    networkPassphrase,
    rpcUrl: server.serverURL.toString(),
    unsignedXdr: prepared.toXDR(),
    transactionHash: toHex(prepared.hash()),
    feeStroops: prepared.fee,
    snapshot,
    builtAt: Date.now(),
  };
}

/**
 * Builds an unsigned approve governance transaction.
 * Simulates on RPC, verifies live controller state, and returns assembled unsigned XDR.
 */
export async function buildApprove(
  approverAddress: string,
  proposalId: bigint | number | string,
  server = defaultRpc()
): Promise<UnsignedGovernanceTx> {
  const { account, contractId, networkPassphrase, snapshot } =
    await readLiveGovernanceState(server, approverAddress);

  const scVals = controllerSpec.funcArgsToScVals("approve", {
    proposal_id: BigInt(proposalId),
    approver: approverAddress,
  });

  const unsigned = new TransactionBuilder(account, {
    fee: BASE_FEE,
    networkPassphrase,
  })
    .addOperation(new Contract(contractId).call("approve", ...scVals))
    .setTimeout(300)
    .build();

  const simulation = await server.simulateTransaction(unsigned);
  if (!rpc.Api.isSimulationSuccess(simulation) || !simulation.result) {
    throw new Error(
      `Approve simulation failed: ${
        "error" in simulation ? simulation.error : "no result returned"
      }`
    );
  }

  const prepared = rpc.assembleTransaction(unsigned, simulation).build();
  return {
    action: "approve",
    address: approverAddress,
    contractId,
    networkPassphrase,
    rpcUrl: server.serverURL.toString(),
    unsignedXdr: prepared.toXDR(),
    transactionHash: toHex(prepared.hash()),
    feeStroops: prepared.fee,
    snapshot,
    builtAt: Date.now(),
  };
}

/**
 * Builds an unsigned revoke_approval governance transaction.
 * Simulates on RPC, verifies live controller state, and returns assembled unsigned XDR.
 */
export async function buildRevokeApproval(
  approverAddress: string,
  proposalId: bigint | number | string,
  server = defaultRpc()
): Promise<UnsignedGovernanceTx> {
  const { account, contractId, networkPassphrase, snapshot } =
    await readLiveGovernanceState(server, approverAddress);

  const scVals = controllerSpec.funcArgsToScVals("revoke_approval", {
    proposal_id: BigInt(proposalId),
    approver: approverAddress,
  });

  const unsigned = new TransactionBuilder(account, {
    fee: BASE_FEE,
    networkPassphrase,
  })
    .addOperation(new Contract(contractId).call("revoke_approval", ...scVals))
    .setTimeout(300)
    .build();

  const simulation = await server.simulateTransaction(unsigned);
  if (!rpc.Api.isSimulationSuccess(simulation) || !simulation.result) {
    throw new Error(
      `Revoke approval simulation failed: ${
        "error" in simulation ? simulation.error : "no result returned"
      }`
    );
  }

  const prepared = rpc.assembleTransaction(unsigned, simulation).build();
  return {
    action: "revoke_approval",
    address: approverAddress,
    contractId,
    networkPassphrase,
    rpcUrl: server.serverURL.toString(),
    unsignedXdr: prepared.toXDR(),
    transactionHash: toHex(prepared.hash()),
    feeStroops: prepared.fee,
    snapshot,
    builtAt: Date.now(),
  };
}

async function buildSimulatedGovernanceTx(
  action: UnsignedGovernanceTx["action"],
  label: string,
  address: string,
  args: Record<string, unknown>,
  server: rpc.Server
): Promise<UnsignedGovernanceTx> {
  const { account, contractId, networkPassphrase, snapshot } =
    await readLiveGovernanceState(server, address);

  const scVals = controllerSpec.funcArgsToScVals(action, args);
  const unsigned = new TransactionBuilder(account, {
    fee: BASE_FEE,
    networkPassphrase,
  })
    .addOperation(new Contract(contractId).call(action, ...scVals))
    .setTimeout(300)
    .build();

  const simulation = await server.simulateTransaction(unsigned);
  if (!rpc.Api.isSimulationSuccess(simulation) || !simulation.result) {
    throw new Error(
      `${label} simulation failed: ${
        "error" in simulation ? simulation.error : "no result returned"
      }`
    );
  }

  const prepared = rpc.assembleTransaction(unsigned, simulation).build();
  return {
    action,
    address,
    contractId,
    networkPassphrase,
    rpcUrl: server.serverURL.toString(),
    unsignedXdr: prepared.toXDR(),
    transactionHash: toHex(prepared.hash()),
    feeStroops: prepared.fee,
    snapshot,
    builtAt: Date.now(),
  };
}

/**
 * Builds an unsigned cancel_proposal transaction. The contract rejects
 * cancellation by anyone other than the proposer; simulation surfaces that.
 */
export function buildCancelProposal(
  proposerAddress: string,
  proposalId: bigint | number | string,
  server = defaultRpc()
): Promise<UnsignedGovernanceTx> {
  return buildSimulatedGovernanceTx(
    "cancel_proposal",
    "Cancel proposal",
    proposerAddress,
    { proposal_id: BigInt(proposalId), proposer: proposerAddress },
    server
  );
}

/**
 * Builds an unsigned execute_proposal transaction. Simulation fails with the
 * contract error while the proposal is still timelocked, stale, or expired.
 */
export function buildExecuteProposal(
  executorAddress: string,
  proposalId: bigint | number | string,
  server = defaultRpc()
): Promise<UnsignedGovernanceTx> {
  return buildSimulatedGovernanceTx(
    "execute_proposal",
    "Execute proposal",
    executorAddress,
    { proposal_id: BigInt(proposalId) },
    server
  );
}
