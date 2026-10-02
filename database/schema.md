# Database schema

Migrations in `database/migrations` are the schema source of truth. The API, indexer, and worker use parameterized SQL against PostgreSQL.

Chain-derived records retain their network, controller, ledger, transaction hash, and event index where applicable. The Console's UUID-like text IDs identify local rows only; they never replace a Soroban identity.

## Projection tables

`controller_events` is the idempotent event journal. `fleets`, `proposals`, `approvals`, and `fleet_upgrades` are projections rebuilt or reconciled from it. An indexer transaction inserts the event, applies projections, advances its checkpoint, and commits together.

## Artifact and report tables

WASM bytes live in an artifact store outside PostgreSQL. `artifacts` retains its immutable hash and storage metadata. Release manifests are intentionally stored as exact bytes because their SHA-256 commitment is later compared with the on-chain proposal value.

## Authentication tables

Challenges store a hash of a random nonce, expiry, and one-time use state. Sessions store only a hash of the browser token. Neither table contains a wallet private key.
