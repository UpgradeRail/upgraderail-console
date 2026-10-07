# Indexing

The indexer projection core recognizes the exact event names emitted by `UpgradeController`:

```text
proposal_created, proposal_approved, approval_revoked, threshold_reached,
threshold_reset, proposal_cancelled, proposal_executed, fleet_created,
fleet_upgraded, policy_updated, controller_upgraded
```

Each event identity is `network + transaction hash + event index`. Reprocessing the same identity does not duplicate the fleet upgrade or mutate approval state again. A projection batch returns the previous state when it cannot decode an event, so a caller must commit the event journal, projections, and `indexer_checkpoints` cursor in one database transaction.

The RPC transport calls Stellar RPC `getEvents` with `xdrFormat: "json"` for the configured `UpgradeController` contract. It normalizes decoded `topicJson` and `valueJson` into the projection event model, preserving the contract event names and refusing unknown names instead of guessing their meaning.

The indexer store inserts `controller_events` idempotently with the `network_id + transaction_hash + event_index` uniqueness rule. It advances `indexer_checkpoints` only after projection succeeds and the event journal transaction commits.

## Read-model projection

Beyond the raw journal, the indexer projects each event into the API's read-model tables (`fleets`, `proposals`, `approvals`, `fleet_upgrades`, and the `controllers` policy/version columns) inside the same database transaction as the journal insert and the checkpoint update (`services/indexer/internal/store/readmodel.go`). If the projection update for any event in a batch fails, the whole transaction rolls back and the checkpoint cursor does not advance, so a later retry reprocesses the same batch.

Per-event field provenance, confirmed against real Testnet `getEvents` payloads rather than assumed from the generated TypeScript bindings alone:

| Event | Topics | Data map |
| --- | --- | --- |
| `fleet_created` | `fleet_id` (bytes) | `tag`, `wasm_hash`, `ledger`, `proposal_id`, `manifest_hash` |
| `fleet_upgraded` | `fleet_id` (bytes) | `proposal_id`, `old_wasm_hash`, `new_wasm_hash`, `manifest_hash` |
| `proposal_created` | `proposal_id` | `proposer`, `governance_epoch`, `expires_ledger` |
| `proposal_approved` / `approval_revoked` | `proposal_id`, `approver` | `approval_count` |
| `threshold_reached` | `proposal_id` | `approved_ledger`, `execute_after_ledger` |
| `threshold_reset` / `proposal_cancelled` / `proposal_executed` | `proposal_id` | (empty) |
| `policy_updated` | `proposal_id` | `governance_epoch` |
| `controller_upgraded` | `proposal_id` | `new_controller_version`, `new_wasm_hash`, `manifest_hash` |

The `approver` address was initially assumed to live in the data map; live Testnet events showed it is actually the event's third topic. This was caught by running the indexer against the real deployed controller, not by unit tests alone, and is documented here so the mistake is not repeated.

The `proposal_created` event does not carry the proposal's `kind` or manifest hash. The indexer fills them from the proposal's **execution** event (`fleet_created` → `CreateFleet`, `fleet_upgraded` → `UpgradeFleet`, `policy_updated` → `UpdatePolicy`, `controller_upgraded` → `UpgradeController`, each with the manifest hash it carries). A live Testnet run (2026-10-07) projected both executed proposals with a real `kind` and `manifest_hash`.

## Reconciliation

Proposals that were never executed (still pending, cancelled, or expired) have no execution event to fill `kind`/`manifest_hash` from. For these, the indexer reconciles against the live `UpgradeController` contract instead of leaving the gap unfilled forever.

`Client.GetProposalKind` (`services/indexer/internal/rpc/simulate.go`) calls the contract's `get_proposal` method via Soroban RPC's `simulateTransaction` — read-only: it builds an unsigned transaction with a freshly generated, never-used source keypair (so no private key is reused or persisted), simulates it, and never signs or submits it. It requests `xdrFormat: "json"` for the return value, the same JSON-XDR scVal encoding `getEvents` already uses, and decodes only the `kind` field: a `ProposalKind` enum, which Soroban encodes as a `Vec` of `[variant_name_symbol, variant_payload_struct]`. It reads the payload's `manifest_hash` field when present. An `UpdatePolicy` proposal's payload (`GovernancePolicy`) has no `manifest_hash` field at all — decoding returns the kind with a `nil` hash, which is "the contract genuinely has nothing here," not a failure. Any other unrecognized shape is a decode error, not a guess.

`services/indexer/internal/reconcile.Run` orchestrates this: each indexer poll cycle, after journaling whatever new events arrived, it selects up to 25 proposals (`reconcile.DefaultBatchSize`) whose `reconciled_at` column is still `NULL` — never attempted, or a previous attempt only hit a transient RPC failure — reads each from the contract, and writes back only the fields the read actually produced (`Store.ReconcileProposal` uses `COALESCE`, so it can never overwrite an existing value, including one this same pass is about to also learn, with `NULL`). A transport failure (`*rpc.TemporaryError`: timeout, 5xx, connection refused) leaves `reconciled_at` `NULL` so the next poll cycle retries that proposal; a non-retryable failure (the contract itself rejected the call, or the response could not be decoded) sets `reconciled_at` anyway, recording the error in `reconciliation_error`, so one bad proposal does not get retried forever. Reconciliation is a side process: it shares the indexer's database connection but never touches `indexer_checkpoints`, and a failure in it is logged and never stops the main event-journaling loop.

This also serves as the backfill path for proposals indexed before this feature existed: migration `000004_proposal_reconciliation.sql` added `reconciled_at`/`reconciliation_error` as nullable columns, so every pre-existing proposal row starts out `NULL` (pending) and is picked up by the very next poll cycle, bounded by the same batch size — no re-indexing from ledger zero required. The event-derived path (fleet/controller/policy events filling `kind`/`manifest_hash` directly) also now sets `reconciled_at`, so a proposal whose execution event already supplied the full picture is not needlessly re-read over RPC.

Verified live against the real, deployed Testnet controller (`CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`, see `docs/testnet-verification.md`): both of its real proposals (`CreateFleet` and `UpgradeFleet`, both executed) already get `kind`/`manifest_hash` from their execution events and so need no RPC round trip; a dedicated live test (`services/indexer/internal/reconcile/live_testnet_test.go`) deliberately clears proposal 2's fields back to the pre-reconciliation `NULL` state and confirms the real reconciliation pass restores them by reading the live contract. No live *unexecuted* proposal exists on this controller to demonstrate the RPC-filled path end-to-end against real chain state in a before/after read of a still-pending proposal; the fixture-based decode tests in `services/indexer/internal/rpc/simulate_test.go` cover `CreateFleet`, `UpgradeFleet`, `UpgradeController`, and `UpdatePolicy` shapes instead.

The web proposal page's own live `get_proposal` read (`apps/web/src/lib/governance-tx.ts`) is left in place as an additional freshness/verification layer rather than removed — it is still useful as independent confirmation that the API-served value matches the chain at page-view time — but the API's `kind`/`manifest_hash` are no longer expected to depend on it being present.

Idempotency: every projection statement is either an `INSERT ... ON CONFLICT DO NOTHING` on the row's natural unique key, or an `UPDATE ... SET column = <value from this event>` (never an increment or accumulation), so replaying the same event twice — a duplicate RPC page, or a batch retried after a transient failure — leaves the read models exactly as they were after applying it once. See `TestApplyControllerBatchAppliesControllerEventsIdempotently` and `TestApplyControllerBatchDoesNotAdvanceCheckpointOnReadModelFailure` in `services/indexer/internal/store/store_test.go`.

Live read-only Testnet transport verification uses the deployment record from `upgraderail-contracts/deployments/testnet.json`:

```text
STELLAR_RPC_LIVE=1 \
STELLAR_RPC_URL=https://soroban-testnet.stellar.org \
UPGRADERAIL_CONTROLLER_ID=CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
UPGRADERAIL_START_LEDGER=4969430 \
go test ./services/indexer/internal/rpc -run TestFetchControllerEventsLive -count=1 -v
```
