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

Live read-only Testnet transport verification uses the deployment record from `upgraderail-contracts/deployments/testnet.json`:

```text
STELLAR_RPC_LIVE=1 \
STELLAR_RPC_URL=https://soroban-testnet.stellar.org \
UPGRADERAIL_CONTROLLER_ID=CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
UPGRADERAIL_START_LEDGER=4969430 \
go test ./services/indexer/internal/rpc -run TestFetchControllerEventsLive -count=1 -v
```
