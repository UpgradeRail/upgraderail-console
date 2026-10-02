# Indexing

The indexer projection core recognizes the exact event names emitted by `UpgradeController`:

```text
proposal_created, proposal_approved, approval_revoked, threshold_reached,
threshold_reset, proposal_cancelled, proposal_executed, fleet_created,
fleet_upgraded, policy_updated, controller_upgraded
```

Each event identity is `network + transaction hash + event index`. Reprocessing the same identity does not duplicate the fleet upgrade or mutate approval state again. A projection batch returns the previous state when it cannot decode an event, so a caller must commit the event journal, projections, and `indexer_checkpoints` cursor in one database transaction.

The deployed RPC event transport and XDR-to-normalized-event adapter are still environment-specific work. The projection layer intentionally refuses unknown event names instead of guessing their meaning.
