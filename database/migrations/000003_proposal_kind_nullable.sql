-- The UpgradeController's proposal_created event carries proposal_id, proposer,
-- governance_epoch, and expires_ledger only; it does not emit the proposal kind
-- or manifest hash (those live inside the ProposalKind variant returned by the
-- contract's get_proposal read method, not in the event payload). Until the
-- indexer adds read-only contract reconciliation for that field, the indexer
-- persists proposals with kind left NULL rather than fabricating a value.
ALTER TABLE proposals ALTER COLUMN kind DROP NOT NULL;
