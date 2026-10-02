import { Buffer } from "buffer";
import { Address } from "@stellar/stellar-sdk";
import {
  AssembledTransaction,
  Client as ContractClient,
  ClientOptions as ContractClientOptions,
  MethodOptions,
  Result,
  Spec as ContractSpec,
} from "@stellar/stellar-sdk/contract";
import type {
  u32,
  i32,
  u64,
  i64,
  u128,
  i128,
  u256,
  i256,
  Option,
  Timepoint,
  Duration,
} from "@stellar/stellar-sdk/contract";
export * from "@stellar/stellar-sdk";
export * as contract from "@stellar/stellar-sdk/contract";
export * as rpc from "@stellar/stellar-sdk/rpc";

if (typeof window !== "undefined") {
  //@ts-ignore Buffer exists
  window.Buffer = window.Buffer || Buffer;
}





export interface Fleet {
  created_ledger: u32;
  id: Buffer;
  tag: string;
}


export interface Proposal {
  approval_count: u32;
  approved_ledger: Option<u32>;
  created_ledger: u32;
  execute_after_ledger: Option<u32>;
  expires_ledger: u32;
  governance_epoch: u64;
  id: u64;
  kind: ProposalKind;
  proposer: string;
  status: StoredProposalStatus;
}

export type ProposalKind = {tag: "CreateFleet", values: readonly [CreateFleetProposal]} | {tag: "UpgradeFleet", values: readonly [UpgradeFleetProposal]} | {tag: "UpdatePolicy", values: readonly [UpdatePolicyProposal]} | {tag: "UpgradeController", values: readonly [UpgradeControllerProposal]};

export type ProposalState = {tag: "AwaitingApprovals", values: void} | {tag: "Timelocked", values: void} | {tag: "Ready", values: void} | {tag: "Expired", values: void} | {tag: "Stale", values: void} | {tag: "Executed", values: void} | {tag: "Cancelled", values: void};


export interface GovernancePolicy {
  approvers: Array<string>;
  proposal_lifetime_ledgers: u32;
  threshold: u32;
  timelock_ledgers: u32;
}


export interface CreateFleetProposal {
  fleet_id: Buffer;
  initial_wasm_hash: Buffer;
  manifest_hash: Buffer;
  tag: string;
}

export type StoredProposalStatus = {tag: "Active", values: void} | {tag: "Executed", values: void} | {tag: "Cancelled", values: void};


export interface UpdatePolicyProposal {
  policy: GovernancePolicy;
}


export interface UpgradeFleetProposal {
  expected_wasm_hash: Buffer;
  fleet_id: Buffer;
  manifest_hash: Buffer;
  new_wasm_hash: Buffer;
}


export interface UpgradeControllerProposal {
  expected_controller_version: u32;
  manifest_hash: Buffer;
  new_controller_version: u32;
  new_wasm_hash: Buffer;
}

export const ContractError = {
  1: {message:"InvalidPolicy"},
  2: {message:"NoApprovers"},
  3: {message:"TooManyApprovers"},
  4: {message:"DuplicateApprover"},
  5: {message:"InvalidThreshold"},
  6: {message:"InvalidTimelock"},
  7: {message:"InvalidProposalLifetime"},
  20: {message:"NotApprover"},
  30: {message:"FleetExists"},
  31: {message:"FleetNotFound"},
  32: {message:"TagAlreadyUsed"},
  33: {message:"ExecutableRefMissing"},
  34: {message:"CurrentWasmMismatch"},
  35: {message:"CandidateMatchesCurrent"},
  36: {message:"InvalidFleetTag"},
  37: {message:"InvalidWasmHash"},
  38: {message:"InvalidManifestHash"},
  50: {message:"ProposalNotFound"},
  51: {message:"ProposalNotActive"},
  52: {message:"ProposalExpired"},
  53: {message:"ProposalStale"},
  54: {message:"AlreadyApproved"},
  55: {message:"ApprovalNotFound"},
  56: {message:"ThresholdNotMet"},
  57: {message:"TimelockNotStarted"},
  58: {message:"TimelockNotElapsed"},
  59: {message:"CannotCancelAfterThreshold"},
  70: {message:"ControllerVersionMismatch"},
  71: {message:"InvalidControllerVersion"},
  80: {message:"ArithmeticOverflow"},
  81: {message:"TtlConfigurationInvalid"}
}












export interface Client {
  /**
   * Construct and simulate a approve transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  approve: ({proposal_id, approver}: {proposal_id: u64, approver: string}, options?: MethodOptions) => Promise<AssembledTransaction<Result<void>>>

  /**
   * Construct and simulate a get_fleet transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_fleet: ({fleet_id}: {fleet_id: Buffer}, options?: MethodOptions) => Promise<AssembledTransaction<Result<Fleet>>>

  /**
   * Construct and simulate a get_policy transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_policy: (options?: MethodOptions) => Promise<AssembledTransaction<GovernancePolicy>>

  /**
   * Construct and simulate a get_proposal transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_proposal: ({proposal_id}: {proposal_id: u64}, options?: MethodOptions) => Promise<AssembledTransaction<Result<Proposal>>>

  /**
   * Construct and simulate a has_approved transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  has_approved: ({proposal_id, approver}: {proposal_id: u64, approver: string}, options?: MethodOptions) => Promise<AssembledTransaction<boolean>>

  /**
   * Construct and simulate a cancel_proposal transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  cancel_proposal: ({proposal_id, proposer}: {proposal_id: u64, proposer: string}, options?: MethodOptions) => Promise<AssembledTransaction<Result<void>>>

  /**
   * Construct and simulate a create_proposal transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  create_proposal: ({proposer, kind}: {proposer: string, kind: ProposalKind}, options?: MethodOptions) => Promise<AssembledTransaction<Result<u64>>>

  /**
   * Construct and simulate a revoke_approval transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  revoke_approval: ({proposal_id, approver}: {proposal_id: u64, approver: string}, options?: MethodOptions) => Promise<AssembledTransaction<Result<void>>>

  /**
   * Construct and simulate a execute_proposal transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  execute_proposal: ({proposal_id}: {proposal_id: u64}, options?: MethodOptions) => Promise<AssembledTransaction<Result<void>>>

  /**
   * Construct and simulate a get_current_wasm transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_current_wasm: ({fleet_id}: {fleet_id: Buffer}, options?: MethodOptions) => Promise<AssembledTransaction<Result<Buffer>>>

  /**
   * Construct and simulate a get_proposal_state transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_proposal_state: ({proposal_id}: {proposal_id: u64}, options?: MethodOptions) => Promise<AssembledTransaction<Result<ProposalState>>>

  /**
   * Construct and simulate a get_governance_epoch transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_governance_epoch: (options?: MethodOptions) => Promise<AssembledTransaction<u64>>

  /**
   * Construct and simulate a get_controller_version transaction. Returns an `AssembledTransaction` object which will have a `result` field containing the result of the simulation. If this transaction changes contract state, you will need to call `signAndSend()` on the returned object.
   */
  get_controller_version: (options?: MethodOptions) => Promise<AssembledTransaction<u32>>

}
export class Client extends ContractClient {
  static async deploy<T = Client>(
        /** Constructor/Initialization Args for the contract's `__constructor` method */
        {policy}: {policy: GovernancePolicy},
    /** Options for initializing a Client as well as for calling a method, with extras specific to deploying. */
    options: MethodOptions &
      Omit<ContractClientOptions, "contractId"> & {
        /** The hash of the Wasm blob, which must already be installed on-chain. */
        wasmHash: Buffer | string;
        /** Salt used to generate the contract's ID. Passed through to {@link Operation.createCustomContract}. Default: random. */
        salt?: Buffer | Uint8Array;
        /** The format used to decode `wasmHash`, if it's provided as a string. */
        format?: "hex" | "base64";
      }
  ): Promise<AssembledTransaction<T>> {
    return ContractClient.deploy({policy}, options)
  }
  constructor(public readonly options: ContractClientOptions) {
    super(
      new ContractSpec([ "AAAAAAAAAAAAAAAHYXBwcm92ZQAAAAACAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAAAAAAhhcHByb3ZlcgAAABMAAAABAAAD6QAAAAIAAAfQAAAADUNvbnRyYWN0RXJyb3IAAAA=",
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
        "AAAABAAAAAAAAAAAAAAADUNvbnRyYWN0RXJyb3IAAAAAAAAfAAAAAAAAAA1JbnZhbGlkUG9saWN5AAAAAAAAAQAAAAAAAAALTm9BcHByb3ZlcnMAAAAAAgAAAAAAAAAQVG9vTWFueUFwcHJvdmVycwAAAAMAAAAAAAAAEUR1cGxpY2F0ZUFwcHJvdmVyAAAAAAAABAAAAAAAAAAQSW52YWxpZFRocmVzaG9sZAAAAAUAAAAAAAAAD0ludmFsaWRUaW1lbG9jawAAAAAGAAAAAAAAABdJbnZhbGlkUHJvcG9zYWxMaWZldGltZQAAAAAHAAAAAAAAAAtOb3RBcHByb3ZlcgAAAAAUAAAAAAAAAAtGbGVldEV4aXN0cwAAAAAeAAAAAAAAAA1GbGVldE5vdEZvdW5kAAAAAAAAHwAAAAAAAAAOVGFnQWxyZWFkeVVzZWQAAAAAACAAAAAAAAAAFEV4ZWN1dGFibGVSZWZNaXNzaW5nAAAAIQAAAAAAAAATQ3VycmVudFdhc21NaXNtYXRjaAAAAAAiAAAAAAAAABdDYW5kaWRhdGVNYXRjaGVzQ3VycmVudAAAAAAjAAAAAAAAAA9JbnZhbGlkRmxlZXRUYWcAAAAAJAAAAAAAAAAPSW52YWxpZFdhc21IYXNoAAAAACUAAAAAAAAAE0ludmFsaWRNYW5pZmVzdEhhc2gAAAAAJgAAAAAAAAAQUHJvcG9zYWxOb3RGb3VuZAAAADIAAAAAAAAAEVByb3Bvc2FsTm90QWN0aXZlAAAAAAAAMwAAAAAAAAAPUHJvcG9zYWxFeHBpcmVkAAAAADQAAAAAAAAADVByb3Bvc2FsU3RhbGUAAAAAAAA1AAAAAAAAAA9BbHJlYWR5QXBwcm92ZWQAAAAANgAAAAAAAAAQQXBwcm92YWxOb3RGb3VuZAAAADcAAAAAAAAAD1RocmVzaG9sZE5vdE1ldAAAAAA4AAAAAAAAABJUaW1lbG9ja05vdFN0YXJ0ZWQAAAAAADkAAAAAAAAAElRpbWVsb2NrTm90RWxhcHNlZAAAAAAAOgAAAAAAAAAaQ2Fubm90Q2FuY2VsQWZ0ZXJUaHJlc2hvbGQAAAAAADsAAAAAAAAAGUNvbnRyb2xsZXJWZXJzaW9uTWlzbWF0Y2gAAAAAAABGAAAAAAAAABhJbnZhbGlkQ29udHJvbGxlclZlcnNpb24AAABHAAAAAAAAABJBcml0aG1ldGljT3ZlcmZsb3cAAAAAAFAAAAAAAAAAF1R0bENvbmZpZ3VyYXRpb25JbnZhbGlkAAAAAFE=",
        "AAAABQAAAAAAAAAAAAAADEZsZWV0Q3JlYXRlZAAAAAEAAAANZmxlZXRfY3JlYXRlZAAAAAAAAAYAAAAAAAAACGZsZWV0X2lkAAAD7gAAACAAAAABAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAAAAAAAAAAADdGFnAAAAABAAAAAAAAAAAAAAAAl3YXNtX2hhc2gAAAAAAAPuAAAAIAAAAAAAAAAAAAAADW1hbmlmZXN0X2hhc2gAAAAAAAPuAAAAIAAAAAAAAAAAAAAABmxlZGdlcgAAAAAABAAAAAAAAAAC",
        "AAAABQAAAAAAAAAAAAAADUZsZWV0VXBncmFkZWQAAAAAAAABAAAADmZsZWV0X3VwZ3JhZGVkAAAAAAAFAAAAAAAAAAhmbGVldF9pZAAAA+4AAAAgAAAAAQAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAAAAAAAAAAADW9sZF93YXNtX2hhc2gAAAAAAAPuAAAAIAAAAAAAAAAAAAAADW5ld193YXNtX2hhc2gAAAAAAAPuAAAAIAAAAAAAAAAAAAAADW1hbmlmZXN0X2hhc2gAAAAAAAPuAAAAIAAAAAAAAAAC",
        "AAAABQAAAAAAAAAAAAAADVBvbGljeVVwZGF0ZWQAAAAAAAABAAAADnBvbGljeV91cGRhdGVkAAAAAAACAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAQAAAAAAAAAQZ292ZXJuYW5jZV9lcG9jaAAAAAYAAAAAAAAAAg==",
        "AAAABQAAAAAAAAAAAAAADlRocmVzaG9sZFJlc2V0AAAAAAABAAAAD3RocmVzaG9sZF9yZXNldAAAAAABAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAQAAAAI=",
        "AAAABQAAAAAAAAAAAAAAD0FwcHJvdmFsUmV2b2tlZAAAAAABAAAAEGFwcHJvdmFsX3Jldm9rZWQAAAADAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAQAAAAAAAAAIYXBwcm92ZXIAAAATAAAAAQAAAAAAAAAOYXBwcm92YWxfY291bnQAAAAAAAQAAAAAAAAAAg==",
        "AAAABQAAAAAAAAAAAAAAD1Byb3Bvc2FsQ3JlYXRlZAAAAAABAAAAEHByb3Bvc2FsX2NyZWF0ZWQAAAAEAAAAAAAAAAtwcm9wb3NhbF9pZAAAAAAGAAAAAQAAAAAAAAAIcHJvcG9zZXIAAAATAAAAAAAAAAAAAAAQZ292ZXJuYW5jZV9lcG9jaAAAAAYAAAAAAAAAAAAAAA5leHBpcmVzX2xlZGdlcgAAAAAABAAAAAAAAAAC",
        "AAAABQAAAAAAAAAAAAAAEFByb3Bvc2FsQXBwcm92ZWQAAAABAAAAEXByb3Bvc2FsX2FwcHJvdmVkAAAAAAAAAwAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAEAAAAAAAAACGFwcHJvdmVyAAAAEwAAAAEAAAAAAAAADmFwcHJvdmFsX2NvdW50AAAAAAAEAAAAAAAAAAI=",
        "AAAABQAAAAAAAAAAAAAAEFByb3Bvc2FsRXhlY3V0ZWQAAAABAAAAEXByb3Bvc2FsX2V4ZWN1dGVkAAAAAAAAAQAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAEAAAAC",
        "AAAABQAAAAAAAAAAAAAAEFRocmVzaG9sZFJlYWNoZWQAAAABAAAAEXRocmVzaG9sZF9yZWFjaGVkAAAAAAAAAwAAAAAAAAALcHJvcG9zYWxfaWQAAAAABgAAAAEAAAAAAAAAD2FwcHJvdmVkX2xlZGdlcgAAAAAEAAAAAAAAAAAAAAAUZXhlY3V0ZV9hZnRlcl9sZWRnZXIAAAAEAAAAAAAAAAI=",
        "AAAABQAAAAAAAAAAAAAAEVByb3Bvc2FsQ2FuY2VsbGVkAAAAAAAAAQAAABJwcm9wb3NhbF9jYW5jZWxsZWQAAAAAAAIAAAAAAAAAC3Byb3Bvc2FsX2lkAAAAAAYAAAABAAAAAAAAAAhwcm9wb3NlcgAAABMAAAAAAAAAAg==",
        "AAAABQAAAAAAAAAAAAAAEkNvbnRyb2xsZXJVcGdyYWRlZAAAAAAAAQAAABNjb250cm9sbGVyX3VwZ3JhZGVkAAAAAAQAAAAAAAAAC3Byb3Bvc2FsX2lkAAAAAAYAAAABAAAAAAAAABZuZXdfY29udHJvbGxlcl92ZXJzaW9uAAAAAAAEAAAAAAAAAAAAAAANbmV3X3dhc21faGFzaAAAAAAAA+4AAAAgAAAAAAAAAAAAAAANbWFuaWZlc3RfaGFzaAAAAAAAA+4AAAAgAAAAAAAAAAI=" ]),
      options
    )
  }
  public readonly fromJSON = {
    approve: this.txFromJSON<Result<void>>,
        get_fleet: this.txFromJSON<Result<Fleet>>,
        get_policy: this.txFromJSON<GovernancePolicy>,
        get_proposal: this.txFromJSON<Result<Proposal>>,
        has_approved: this.txFromJSON<boolean>,
        cancel_proposal: this.txFromJSON<Result<void>>,
        create_proposal: this.txFromJSON<Result<u64>>,
        revoke_approval: this.txFromJSON<Result<void>>,
        execute_proposal: this.txFromJSON<Result<void>>,
        get_current_wasm: this.txFromJSON<Result<Buffer>>,
        get_proposal_state: this.txFromJSON<Result<ProposalState>>,
        get_governance_epoch: this.txFromJSON<u64>,
        get_controller_version: this.txFromJSON<u32>
  }
}