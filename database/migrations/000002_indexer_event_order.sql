-- Preserve Stellar RPC event order for deterministic projection replay after restart.
-- Existing journal rows remain nullable; the indexer refuses to resume a legacy
-- checkpoint without ordered events and requires a fresh bootstrap instead.
ALTER TABLE controller_events ADD COLUMN rpc_event_id TEXT;
CREATE UNIQUE INDEX controller_events_rpc_event_id_idx
    ON controller_events (controller_id, rpc_event_id)
    WHERE rpc_event_id IS NOT NULL;
