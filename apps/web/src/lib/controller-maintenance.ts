import { Account, BASE_FEE, Contract, TransactionBuilder, rpc, scValToNative } from "@stellar/stellar-sdk";

export const TESTNET_PASSPHRASE = "Test SDF Network ; September 2015";

// Checked against upgraderail-contracts/deployments/testnet.json and the live
// Testnet contract interface/hash on 2026-10-03. Fail closed after an upgrade.
export const VERIFIED_CONTROLLER_ID = "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3";
export const VERIFIED_CONTROLLER_WASM_HASH = "0ac2626c26b43f5330bf89cfdd952ecdd9940817a703ec04c327f478f5f20458";

export type MaintenanceSnapshot = {
  ledger: number;
  governanceEpoch: string;
  controllerVersion: number;
  accountSequence: string;
  wasmHash: string;
};

export type UnsignedMaintenance = {
  action: "maintain_controller";
  address: string;
  contractId: string;
  networkPassphrase: typeof TESTNET_PASSPHRASE;
  rpcUrl: string;
  unsignedXdr: string;
  transactionHash: string;
  feeStroops: string;
  snapshot: MaintenanceSnapshot;
  builtAt: number;
};

export function maintenanceRpc(): rpc.Server {
  const endpoint = process.env.NEXT_PUBLIC_STELLAR_RPC_URL?.trim();
  if (!endpoint || new URL(endpoint).protocol !== "https:") {
    throw new Error("Configure an HTTPS Testnet RPC endpoint before building a transaction.");
  }
  return new rpc.Server(endpoint);
}

function configuredContractId(): string {
  const configured = process.env.NEXT_PUBLIC_CONTROLLER_ID?.trim();
  if (configured !== VERIFIED_CONTROLLER_ID) {
    throw new Error("The configured controller does not match the verified Testnet deployment record.");
  }
  return configured;
}

export function requireVerifiedTestnet(rpcPassphrase: string, wasmHash: string): void {
  if (process.env.NEXT_PUBLIC_STELLAR_NETWORK?.toLowerCase() !== "testnet" ||
      process.env.NEXT_PUBLIC_STELLAR_NETWORK_PASSPHRASE !== TESTNET_PASSPHRASE ||
      rpcPassphrase !== TESTNET_PASSPHRASE) {
    throw new Error("Wrong network: this verification flow is available only on Stellar Testnet.");
  }
  if (wasmHash !== VERIFIED_CONTROLLER_WASM_HASH) {
    throw new Error("The live controller WASM differs from the verified deployment record. Refresh the record before building.");
  }
}

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function simulateRead(server: rpc.Server, source: Awaited<ReturnType<rpc.Server["getAccount"]>>, method: "get_governance_epoch" | "get_controller_version", contractId: string): Promise<bigint> {
  // TransactionBuilder.build mutates its Account source sequence. Read-only
  // simulations must not consume the source used by the write transaction.
  const simulationSource = new Account(source.accountId(), source.sequenceNumber());
  const tx = new TransactionBuilder(simulationSource, { fee: BASE_FEE, networkPassphrase: TESTNET_PASSPHRASE })
    .addOperation(new Contract(contractId).call(method))
    .setTimeout(60)
    .build();
  const result = await server.simulateTransaction(tx);
  if (!rpc.Api.isSimulationSuccess(result) || !result.result) {
    throw new Error(`Live controller ${method} simulation failed: ${"error" in result ? result.error : "no result returned"}`);
  }
  const value: unknown = scValToNative(result.result.retval);
  if (typeof value !== "bigint" && (typeof value !== "number" || !Number.isSafeInteger(value))) {
    throw new Error(`Live controller ${method} returned an invalid value.`);
  }
  return BigInt(value);
}

async function liveState(server: rpc.Server, address: string) {
  const contractId = configuredContractId();
  const [network, ledger, instance, account] = await Promise.all([
    server.getNetwork(),
    server.getLatestLedger(),
    server.getContractInstance(contractId),
    server.getAccount(address),
  ]);
  const executable = instance.executable;
  const wasmHash = executable.type === "contractExecutableWasm" ? toHex(executable.value.value) : "unsupported";
  requireVerifiedTestnet(network.passphrase, wasmHash);
  const [epoch, version] = await Promise.all([
    simulateRead(server, account, "get_governance_epoch", contractId),
    simulateRead(server, account, "get_controller_version", contractId),
  ]);
  if (version < 0 || version > 0xffffffffn) throw new Error("The live controller version is out of range.");
  return {
    account,
    snapshot: {
      ledger: ledger.sequence,
      governanceEpoch: epoch.toString(),
      controllerVersion: Number(version),
      accountSequence: account.sequenceNumber(),
      wasmHash,
    } satisfies MaintenanceSnapshot,
  };
}

export function assertMaintenanceFresh(previous: MaintenanceSnapshot, current: MaintenanceSnapshot): void {
  if (current.ledger < previous.ledger || current.ledger - previous.ledger > 20 ||
      current.wasmHash !== previous.wasmHash ||
      current.governanceEpoch !== previous.governanceEpoch ||
      current.controllerVersion !== previous.controllerVersion ||
      current.accountSequence !== previous.accountSequence) {
    throw new Error("Controller or account state changed since simulation. Build a new unsigned transaction.");
  }
}

export async function buildMaintenanceTransaction(address: string, server = maintenanceRpc()): Promise<UnsignedMaintenance> {
  const { account, snapshot } = await liveState(server, address);
  const contractId = configuredContractId();
  // The pinned deployed contract exposes maintain_controller. The checked-in
  // generated v1 binding predates this method, so invoke only this verified name.
  const unsigned = new TransactionBuilder(account, { fee: BASE_FEE, networkPassphrase: TESTNET_PASSPHRASE })
    .addOperation(new Contract(contractId).call("maintain_controller"))
    .setTimeout(300)
    .build();
  const simulation = await server.simulateTransaction(unsigned);
  if (!rpc.Api.isSimulationSuccess(simulation) || !simulation.result) {
    throw new Error(`Controller maintenance simulation failed: ${"error" in simulation ? simulation.error : "no result returned"}`);
  }
  if (simulation.result.auth.length !== 0) {
    throw new Error("Controller maintenance unexpectedly requires additional authorization.");
  }
  const prepared = rpc.assembleTransaction(unsigned, simulation).build();
  return {
    action: "maintain_controller",
    address,
    contractId,
    networkPassphrase: TESTNET_PASSPHRASE,
    rpcUrl: process.env.NEXT_PUBLIC_STELLAR_RPC_URL!,
    unsignedXdr: prepared.toXDR(),
    transactionHash: toHex(prepared.hash()),
    feeStroops: prepared.fee,
    snapshot,
    builtAt: Date.now(),
  };
}

export async function refreshMaintenanceState(draft: UnsignedMaintenance, server = maintenanceRpc()): Promise<void> {
  if (Date.now() - draft.builtAt > 120_000) {
    throw new Error("The unsigned transaction is older than two minutes. Build a new one.");
  }
  const { snapshot } = await liveState(server, draft.address);
  assertMaintenanceFresh(draft.snapshot, snapshot);
}
