-- UpgradeRail Console's reviewable source-of-truth schema.
-- Chain identity remains network + contract ID + ledger + transaction hash.

CREATE TABLE networks (
    id TEXT PRIMARY KEY,
    passphrase TEXT NOT NULL UNIQUE,
    rpc_url TEXT NOT NULL,
    protocol_target INTEGER NOT NULL CHECK (protocol_target > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE controllers (
    id TEXT PRIMARY KEY,
    network_id TEXT NOT NULL REFERENCES networks(id),
    contract_id TEXT NOT NULL,
    wasm_hash TEXT,
    governance_epoch BIGINT,
    controller_version INTEGER,
    observed_ledger BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (network_id, contract_id)
);

CREATE TABLE fleets (
    id TEXT PRIMARY KEY,
    controller_id TEXT NOT NULL REFERENCES controllers(id),
    fleet_hash TEXT NOT NULL,
    tag TEXT NOT NULL,
    current_wasm_hash TEXT,
    created_ledger BIGINT NOT NULL,
    created_transaction_hash TEXT NOT NULL,
    created_event_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (controller_id, fleet_hash),
    UNIQUE (controller_id, tag)
);

CREATE TABLE proposals (
    id TEXT PRIMARY KEY,
    controller_id TEXT NOT NULL REFERENCES controllers(id),
    proposal_id BIGINT NOT NULL,
    proposer TEXT NOT NULL,
    kind TEXT NOT NULL,
    governance_epoch BIGINT NOT NULL,
    created_ledger BIGINT NOT NULL,
    expires_ledger BIGINT NOT NULL,
    approval_count INTEGER NOT NULL DEFAULT 0 CHECK (approval_count >= 0),
    approved_ledger BIGINT,
    execute_after_ledger BIGINT,
    status TEXT NOT NULL,
    manifest_hash TEXT,
    created_transaction_hash TEXT NOT NULL,
    created_event_index INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (controller_id, proposal_id)
);

CREATE TABLE approvals (
    id TEXT PRIMARY KEY,
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    approver TEXT NOT NULL,
    approved_ledger BIGINT NOT NULL,
    transaction_hash TEXT NOT NULL,
    event_index INTEGER NOT NULL,
    revoked_ledger BIGINT,
    revoked_transaction_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (proposal_id, approver)
);

CREATE TABLE controller_events (
    id TEXT PRIMARY KEY,
    controller_id TEXT NOT NULL REFERENCES controllers(id),
    network_id TEXT NOT NULL REFERENCES networks(id),
    ledger_sequence BIGINT NOT NULL,
    transaction_hash TEXT NOT NULL,
    event_index INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    topics JSONB NOT NULL,
    data JSONB NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (network_id, transaction_hash, event_index)
);

CREATE INDEX controller_events_controller_ledger_idx
    ON controller_events (controller_id, ledger_sequence DESC, event_index DESC);

CREATE TABLE fleet_upgrades (
    id TEXT PRIMARY KEY,
    fleet_id TEXT NOT NULL REFERENCES fleets(id),
    proposal_id TEXT NOT NULL REFERENCES proposals(id),
    old_wasm_hash TEXT NOT NULL,
    new_wasm_hash TEXT NOT NULL,
    manifest_hash TEXT NOT NULL,
    ledger_sequence BIGINT NOT NULL,
    transaction_hash TEXT NOT NULL,
    event_index INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (transaction_hash, event_index)
);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    sha256 TEXT NOT NULL UNIQUE,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    storage_key TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    uploader TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE analysis_jobs (
    id TEXT PRIMARY KEY,
    network_id TEXT NOT NULL REFERENCES networks(id),
    current_artifact_id TEXT REFERENCES artifacts(id),
    candidate_artifact_id TEXT NOT NULL REFERENCES artifacts(id),
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'ready', 'blocked', 'failed', 'cancelled')),
    engine_version TEXT,
    error_message TEXT,
    created_by TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX analysis_jobs_status_created_idx ON analysis_jobs (status, created_at);

CREATE TABLE analysis_reports (
    id TEXT PRIMARY KEY,
    analysis_job_id TEXT NOT NULL UNIQUE REFERENCES analysis_jobs(id),
    current_wasm_hash TEXT NOT NULL,
    candidate_wasm_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    findings JSONB NOT NULL,
    runtime_evidence JSONB NOT NULL,
    report JSONB NOT NULL,
    engine_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE release_manifests (
    id TEXT PRIMARY KEY,
    analysis_job_id TEXT NOT NULL UNIQUE REFERENCES analysis_jobs(id),
    sha256 TEXT NOT NULL UNIQUE,
    bytes BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE indexer_checkpoints (
    controller_id TEXT PRIMARY KEY REFERENCES controllers(id),
    cursor TEXT NOT NULL,
    ledger_sequence BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE auth_challenges (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL,
    network_id TEXT NOT NULL REFERENCES networks(id),
    public_address TEXT NOT NULL,
    nonce_hash TEXT NOT NULL UNIQUE,
    purpose TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expires_at > issued_at)
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    public_address TEXT NOT NULL,
    network_id TEXT NOT NULL REFERENCES networks(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_address_network_idx ON sessions (public_address, network_id);
