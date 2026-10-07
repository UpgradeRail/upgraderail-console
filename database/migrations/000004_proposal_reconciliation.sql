-- Tracks the outcome of reconciling a proposal's kind/manifest_hash against
-- the live UpgradeController contract (a read-only simulateTransaction call,
-- never a chain mutation). reconciled_at is set only once a reconciliation
-- attempt reaches a definitive outcome (success, or a non-retryable error);
-- it stays NULL while a proposal has never been reconciled or only hit a
-- transient RPC failure, so the indexer knows to retry it. reconciliation_error
-- holds the last attempt's error message, if any, and is cleared on success.
ALTER TABLE proposals ADD COLUMN reconciled_at TIMESTAMPTZ;
ALTER TABLE proposals ADD COLUMN reconciliation_error TEXT;

CREATE INDEX proposals_needs_reconciliation_idx
    ON proposals (controller_id, proposal_id)
    WHERE reconciled_at IS NULL;
