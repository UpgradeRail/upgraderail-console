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

## Artifact upload

`POST /api/v1/artifacts` requires the session cookie and a `multipart/form-data` body with the WASM file in a `file` field. The response is `201` with the artifact row: `id`, `sha256`, `size_bytes`, `content_type`, `uploader`, `created_at`.

- **Size limit**: 8 MiB per upload (`services/api/internal/httpapi.MaxArtifactUploadBytes`), enforced both at the HTTP layer (`http.MaxBytesReader`) and by the storage backend's own `MaxBytes`. An oversized or truncated-by-the-limit upload gets `413` with code `artifact_too_large`.
- **Validation**: an empty file is rejected (`400 empty_artifact`); a file whose first four bytes are not the WASM magic header (`\0asm`) is rejected (`400 invalid_wasm`) before anything is hashed or stored. This is a minimal sanity check, not proof the module is well-formed WASM — the Engine is what actually parses it. The client's declared `Content-Type` is never trusted for this check.
- **Hashing**: SHA-256 is computed server-side from the bytes actually written to storage (`services/shared/artifactstore.Filesystem.Put`), never from a client-supplied value.
- **Dedup**: artifacts are content-addressed; re-uploading identical bytes returns the existing artifact row (same `id`) instead of creating a duplicate, enforced by the `artifacts.sha256` unique constraint.
- **Storage backend**: filesystem only, under `ARTIFACT_LOCAL_DIR` (default `./artifacts`), shared by the API and the worker. There is no S3-compatible or other object-store backend. If the API and worker processes do not share that filesystem (e.g. separate hosts/containers without a shared volume), the worker cannot read what the API wrote. See `docs/limitations.md`.
- Never log the uploaded bytes or any derived secret; only metadata (id, hash, size) is ever logged or returned.

## Analysis jobs

`POST /api/v1/analyses` requires the session cookie and accepts `network`, `current_artifact_id`, and `candidate_artifact_id` (snake_case; both artifacts are required). Before queuing the job, the API confirms both artifact IDs exist and re-hashes their stored bytes against the recorded SHA-256 (`verifyArtifactExists`) — this re-read is an integrity check against disk corruption, not re-establishing trust in a client value, since the hash itself was computed server-side at upload time. Unknown or hash-mismatched artifacts get `400 artifact_not_found`. The worker later changes the job from `queued` to `running`, then to `ready`, `blocked`, or `failed` based on the actual Engine result; status, report, and manifest are read back through the existing `GET /api/v1/analyses/{id}`, `/report`, and `/manifest` endpoints — there is no separate upload-specific status endpoint.
