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

## Governance writes verified on Testnet through Freighter (2026-10-07)

All five governance actions were performed from the real web console, signed in a real Freighter popup, and submitted to Stellar Testnet, using a **disposable** controller rather than the shared evidence controller above:

- controller: `CAJX4YE77N23K53MNJHYMCZIFXGMUEHSNZPHXAHDVZU5IYXUK4OQXTWS`
- policy: single approver `GAH42JRVEHJVL4LYKLIA7G52MX4AUP3DORK6DZGH52VOCYV2QJFUY4ZV`, threshold 1, timelock 1 ledger, proposal lifetime 200 ledgers
- deployed via CLI (not the browser) with a disposable local identity (`upgraderail-verify-deployer`); the approver is a Freighter-held account, unlocked by the human operator for each signature
- local stack: disposable PostgreSQL 16 (`upgraderail-freighter-verify-pg`), the compiled API, two indexer instances (one per controller), the compiled worker with the real Engine binary, and `next dev`, all on the workstation

This target was chosen because neither of the shared controller's two real approvers' secret keys (`GBPU3GJ7P72CP4JSY25WERJRTLYFGKOLENGDBRZKBRRLHCUEBDZ7FSDL`, `GBF5SV2NCEWJ36A6S2E5B6EDFDQKHG6LKYUYAXS2K5OKI2ZLLOR7PLAY`) are reachable from this environment (not in Freighter, not in the local Stellar CLI keystore), so no real signature could be produced against it for this pass.

The web app's `configuredContractId()`/`requireVerifiedNetwork()` (`apps/web/src/lib/governance-tx.ts`) previously hard-failed for any controller but the shared one; it was generalized to a small, explicit allowlist (`VERIFIED_CONTROLLERS`) that includes this disposable controller by exact id and WASM hash, commented as disposable-Testnet-only. See `docs/limitations.md` for the other fixes this pass made (indexer reconciliation decode bug, missing `CreateFleet` UI, controller-id fleet filtering).

| Action | Proposal | Tx hash | Ledger | Result |
| --- | --- | --- | --- | --- |
| Create (`CreateFleet`, tag `freighter-verify-2026-10-07`) | #1 | `8e6f0b8be12e34f50c703847b4f171a2d50abae4b1ead5b6cfbad1ef2f976dc9` | 5072636 | SUCCESS |
| Approve | #1 | `2ef3f316c63bac483bbe8d396ff8c5e695caa0bbaa1e0ca14bdfbf683398fbf1` | 5072749 | SUCCESS |
| Create (`CreateFleet`, tag `freighter-verify-revoke-cancel-01`) | #2 | `c3dfb43b612b51916eb3f82b9aae348ffec169a82d08cd86f7037ed8e7d55d48` | 5072787 | SUCCESS |
| Approve | #2 | `3c861ba929b7d7a2c9f7b0bbac9d23d33c6140c576adb9acea74714ec05f428b` | 5072817 | SUCCESS |
| Revoke approval | #2 | `c80b1b59858e8a549ef45ef1eddb810f7f4e9a7b3769f7387f355e9b267acf13` | 5072846 | SUCCESS |
| Cancel | #2 | `d6b10bdc7e79703a24007f60896230ece06c8170bb64d3247d2f39fcfe97606c` | 5072873 | SUCCESS |
| Create (`CreateFleet`, tag `freighter-verify-execute-01`) | #3 | `2e20f447a753d6885e94de84dbf8d07ff22f4fa2433587f9f34d2d1a55cc2df6` | 5072921 | SUCCESS |
| Approve | #3 | `8ad46bfcc5934fa0b0412ecb5daefedfe0ac2675f36610a1686102864ef0c168` | 5072944 | SUCCESS |
| Execute | #3 | `4b24d73c0e7b2a30a82f375f99cb7e46b8dd439abf90671b1ca6d1733de1224e` | 5073023 | SUCCESS |

Every hash and ledger above was independently confirmed with a direct `getTransaction` call to the public `https://soroban-testnet.stellar.org` RPC endpoint (`"status":"SUCCESS"`), not just read back from this project's own API. After execution, `GET /api/v1/fleets` showed the new fleet `freighter-verify-execute-01` (hash `0eef2058d074e1094099bd54d3b45784dfd413e86275ec9a73a92d08f367961e`, `current_wasm_hash` `a59283a3427b9355b6602c6b611508e0264a1c353b82a51c9883ab37f282bfaa`), and `GET /api/v1/proposals` showed proposals #1 `active` (expired before it could be executed — see below), #2 `cancelled`, and #3 `executed`, all with `kind: "CreateFleet"` populated.

Proposal #1 was approved but never executed: its 200-ledger lifetime (`expires_ledger`) elapsed while proposal #2's revoke/cancel sequence was being tested, and `execute_proposal` correctly refused it as expired (`"Unavailable: Proposal is expired."`). This is the contract working as designed, not a defect, and is why proposal #3 exists to prove execute.

Proposal #3's first `execute_proposal` simulation failed with `HostError: Error(Storage, MissingValue)` / `"Wasm does not exist"` for the candidate WASM hash, because that WASM's bytes had never been uploaded to Testnet as contract code — only hashed locally by the Engine. Uploading it (`stellar contract upload --wasm fleet_v2_compatible.wasm`, tx `d502db4bedeb188db6264380ac7ba2c3ce2a94c6c7011bd01963f6edaff41511`) resolved it; this is a real operational prerequisite for `CreateFleet`, not a console bug.

Driving this live, real proposals also surfaced a real indexer bug: `services/indexer/internal/rpc.GetProposalKind`'s decoder (`scMap` in `client.go`) failed on a fresh, unapproved proposal with `decode proposal struct: decode event value map: json: cannot unmarshal string into Go struct field .val of type rpc.scVal`, because stellar-rpc's `xdrFormat=json` renders an `Option<u32>` `None` (e.g. `approved_ledger`/`execute_after_ledger` before approval) as the bare JSON string `"void"`, not a nested `{"type": ...}` object — a shape never exercised before because every previously-reconciled proposal was already executed, with those fields populated. Fixed by giving `scVal` a custom `UnmarshalJSON` that accepts `"void"` as an empty value and still rejects any other unrecognized bare scalar; covered by a new regression test (`TestDecodeGetProposalResultToleratesVoidOptionFields`) built from this exact live response, plus direct `scVal` unit tests. All Go tests and `go vet` pass; Playwright (55 tests, including axe), `pnpm lint`, `tsc --noEmit`, `vitest` (131 tests), `pnpm build`, a fresh-schema migration check, `scripts/verify-contract-bindings.sh`, and `pnpm audit --prod` all pass on the commit that includes this fix.

Not verified in this pass: a multi-approver (`threshold > 1`) flow with two distinct Freighter-held signers (no second key was available to add as a second approver), and the `UpdatePolicy`/`UpgradeController` proposal kinds from the browser (same code paths as `CreateFleet`/`UpgradeFleet`, but not themselves built, signed, and submitted). See `docs/limitations.md`.

## Proposal reconciliation read (read-only, no write)

Also on 2026-10-07, a `get_proposal` simulation (Soroban RPC `simulateTransaction`, never signed or submitted) against proposal 2 on the same controller decoded to `kind: UpgradeFleet` and `manifest_hash: cd8b679e2215a53c3bda375de6112d0d3c0203d017dfcdcfa5e1a63ef6ef088a` — matching the value independently read earlier in this document via `stellar contract invoke`. This confirms the indexer's new reconciliation RPC call (`services/indexer/internal/rpc.Client.GetProposalKind`) decodes the real contract's response correctly, not just a hand-built fixture. See `docs/indexing.md` for how reconciliation uses this read, and `docs/deployment-verification.md` for the full before/after reconciliation run.
