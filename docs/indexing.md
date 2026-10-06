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

The `proposal_created` event does not carry the proposal's `kind` (`CreateFleet`/`UpgradeFleet`/`UpdatePolicy`/`UpgradeController`) or its manifest hash — those live inside the `ProposalKind` variant returned by the contract's `get_proposal` read method, not in the event payload. The indexer stores `kind` and `manifest_hash` as `NULL` on the proposal row rather than fabricating a value (`database/migrations/000003_proposal_kind_nullable.sql` makes `kind` nullable for this reason). Read-only reconciliation against the live contract to fill this specific gap has not been implemented; see `docs/limitations.md`.

Idempotency: every projection statement is either an `INSERT ... ON CONFLICT DO NOTHING` on the row's natural unique key, or an `UPDATE ... SET column = <value from this event>` (never an increment or accumulation), so replaying the same event twice — a duplicate RPC page, or a batch retried after a transient failure — leaves the read models exactly as they were after applying it once. See `TestApplyControllerBatchAppliesControllerEventsIdempotently` and `TestApplyControllerBatchDoesNotAdvanceCheckpointOnReadModelFailure` in `services/indexer/internal/store/store_test.go`.

Live read-only Testnet transport verification uses the deployment record from `upgraderail-contracts/deployments/testnet.json`:

```text
STELLAR_RPC_LIVE=1 \
STELLAR_RPC_URL=https://soroban-testnet.stellar.org \
UPGRADERAIL_CONTROLLER_ID=CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
UPGRADERAIL_START_LEDGER=4969430 \
go test ./services/indexer/internal/rpc -run TestFetchControllerEventsLive -count=1 -v
```
