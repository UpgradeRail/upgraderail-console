# API

The Console API uses JSON and versioned routes under `/api/v1`.

## Public reads

| Method | Route | Source |
| --- | --- | --- |
| GET | `/api/v1/health` | API process |
| GET | `/api/v1/controllers` | indexer projection |
| GET | `/api/v1/controllers/{id}` | indexer projection |
| GET | `/api/v1/fleets` | indexer projection |
| GET | `/api/v1/fleets/{id}` | indexer projection |
| GET | `/api/v1/fleets/{id}/upgrades` | indexer projection |
| GET | `/api/v1/proposals` | indexer projection |
| GET | `/api/v1/proposals/{id}` | indexer projection |
| GET | `/api/v1/proposals/{id}/approvals` | indexer projection |
| GET | `/api/v1/upgrades` | indexer projection |
| GET | `/api/v1/events` | indexer event journal |
| GET | `/api/v1/analyses` | analysis jobs, newest first |
| GET | `/api/v1/analyses/{id}` | analysis job |
| GET | `/api/v1/analyses/{id}/report` | Engine report |
| GET | `/api/v1/analyses/{id}/manifest` | manifest SHA-256 and bytes (base64 in JSON) |

List endpoints accept `limit` from 1 through 100 and nonnegative `offset` values.

The `indexer projection` rows above are now populated by the indexer's read-model projection (previously these tables existed but were always empty, because only the raw event journal was written). All JSON response fields are `snake_case` (e.g. `current_wasm_hash`, `approval_count`, `expires_ledger`), matching the column names in `database/migrations/000001_initial.sql`; this is enforced with explicit `json` struct tags on `services/api/internal/store.{Controller,Fleet,Proposal,Approval,FleetUpgrade}` and covered by `services/api/internal/httpapi/api_test.go`, which asserts on the real JSON keys served over real HTTP. `proposals.kind` and `proposals.manifest_hash` are filled from the proposal's execution event, so they are real for executed proposals and `null` for proposals that are pending, cancelled, or expired. `null` means "not known to the indexer", never "none". See `docs/indexing.md`. Analysis job, report, and manifest responses are also `snake_case`.

## Wallet session

`POST /api/v1/auth/challenge` creates a five-minute random challenge for a network, public address, domain, and purpose. `POST /api/v1/auth/verify` receives the challenge ID, nonce, and Freighter message signature. It verifies the Stellar Ed25519 key, consumes the challenge once, and returns an HttpOnly, Secure, SameSite=Lax session cookie. `POST /api/v1/auth/logout` requires a matching Origin and revokes that session.

## Analysis jobs

`POST /api/v1/analyses` requires the session cookie and accepts `network`, `current_artifact_id`, and `candidate_artifact_id`. Both artifacts are required. The worker later changes the job from `queued` to `running`, then to `ready`, `blocked`, or `failed` based on the actual Engine result.
