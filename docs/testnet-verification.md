# Testnet read verification

Read-only verification ran on 2026-10-02 against the controller recorded by `upgraderail-contracts/deployments/testnet.json`.

- controller: `CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`
- RPC: `https://soroban-testnet.stellar.org`
- controller policy: two approvers, threshold 2, timelock 12 ledgers, proposal lifetime 720 ledgers
- governance epoch: 1
- controller version: 1
- fleet: `8863397c695ebe1dd9f1922afd511aee68999753d13cd61345767487564dc820` (`upgraderail-testnet-shared`), created at ledger 4,969,465
- current fleet WASM hash: `67a657dd6c64f4255f5e928782a058aaeff30c14c7ca530d2604c554d30d76c7`
- proposal 2: `Executed`, with two approvals; its manifest commitment is `cd8b679e2215a53c3bda375de6112d0d3c0203d017dfcdcfa5e1a63ef6ef088a`

The deployment record reports observed Testnet Protocol 29. These reads do not alter UpgradeRail Engine’s Protocol 28 analysis profile or the Mainnet Protocol 28 production target.

Verification commands used read-only RPC calls and did not submit transactions:

```text
stellar contract invoke --rpc-url https://soroban-testnet.stellar.org \
  --network-passphrase "Test SDF Network ; September 2015" \
  --contract-id CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
  --source-account GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6 \
  --send no -- get_policy
```

The indexer read the same controller events from Stellar RPC with `xdrFormat: "json"` and applied them to a local migrated PostgreSQL database:

```text
DATABASE_URL=postgresql://upgraderail:upgraderail@127.0.0.1:55433/upgraderail_migration_verify \
STELLAR_RPC_LIVE=1 \
STELLAR_RPC_URL=https://soroban-testnet.stellar.org \
UPGRADERAIL_CONTROLLER_ID=CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
UPGRADERAIL_START_LEDGER=4969430 \
go test ./services/indexer/internal/store -run TestLiveReadOnlyTestnetControllerProjection -count=1 -v
```

The same Chrome profile subsequently connected to Freighter on Testnet, completed challenge signing and authenticated session creation, and signed a simulated `maintain_controller` transaction. See `docs/wallet.md`. No write transaction or Testnet current/candidate fleet-pair simulation was performed.

## Maintenance flow and write limitation

The selected verification call is `maintain_controller`, which extends the deployed controller's instance storage lifetime without changing governance policy, proposals, fleets, approvals, or controller version. The console pins the Testnet controller ID and live WASM hash from `upgraderail-contracts/deployments/testnet.json`, verifies the RPC network passphrase, reads live epoch/version/account state, simulates the call, and rejects stale state before signing and submission. The checked-in generated binding predates this deployed method, so this single verified method is invoked through the Stellar SDK `Contract.call` API after checking the live contract hash and interface. The live simulation required no additional authorization and estimated a maximum fee near 2.27 Testnet XLM.

The initial browser sign-only run used the Testnet account displayed as `GBWM5…2UPB`; it stopped before submission. A subsequent browser run used the disposable-account confirmation control and submitted a fresh transaction. The first attempt was rejected before inclusion with `txBadSeq` because read-only simulation had mutated the in-memory source account sequence. The sequence handling was fixed and covered by a regression test.

The retry was **confirmed on Stellar Testnet**:

- network: Stellar Testnet (`Test SDF Network ; September 2015`)
- controller: `CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`
- operation: `maintain_controller`
- transaction hash: `f7bda7274e21e1cebe6ca939218ce92bc47961fcdccd46c9082f5d96c1c371e0`
- ledger: `4999846`
- RPC result: `SUCCESS`

The console displayed the confirmed state. Public `getTransaction` returned the same hash, ledger, and status. The decoded envelope contained one invocation of the named operation on the recorded controller. No Mainnet transaction was made. This is verification of one maintenance write, not of governance proposal or fleet write flows.

## Governance writes are not yet verified on Testnet

The only verified Testnet write is `maintain_controller`. The create, approve, revoke, cancel, and execute governance flows have not been submitted from the browser. On 2026-10-07 the indexer was re-run live against the controller above from ledger 4,969,430 into a fresh database. It produced 12 events, 1 fleet, 2 proposals (`CreateFleet` and `UpgradeFleet`, both executed, with manifest hashes), 4 approvals, and 1 upgrade, and the API served them.

## Proposal reconciliation read (read-only, no write)

Also on 2026-10-07, a `get_proposal` simulation (Soroban RPC `simulateTransaction`, never signed or submitted) against proposal 2 on the same controller decoded to `kind: UpgradeFleet` and `manifest_hash: cd8b679e2215a53c3bda375de6112d0d3c0203d017dfcdcfa5e1a63ef6ef088a` — matching the value independently read earlier in this document via `stellar contract invoke`. This confirms the indexer's new reconciliation RPC call (`services/indexer/internal/rpc.Client.GetProposalKind`) decodes the real contract's response correctly, not just a hand-built fixture. See `docs/indexing.md` for how reconciliation uses this read, and `docs/deployment-verification.md` for the full before/after reconciliation run.
