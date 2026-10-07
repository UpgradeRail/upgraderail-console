# Staging Deployment Guide & Verification Record

Date: 2026-10-07  
Status: **Web console and backend services are publicly deployed and verified; overall status NOT READY (see section 6)**

## 1. Staging Architecture

Chosen architecture (Option A: managed services, Stellar Testnet only):

| Component | Provider | Region | Service name | Notes |
|---|---|---|---|---|
| Web | Vercel | global | `upgraderail-console` | Existing deployment, `https://upgraderail-console.vercel.app` |
| PostgreSQL | Supabase (Free) | West EU (Ireland), `eu-west-1` | `upgraderail-staging` (ref `drkywpdasiaqsnhphufq`) | Postgres 15+, reached through the **session pooler** (IPv4) |
| API | Render (Free web service, Docker) | Frankfurt | `upgraderail-api` | `services/api/Dockerfile` |
| Indexer | Render (Free web service, Docker) | Frankfurt | `upgraderail-indexer` | `services/indexer/Dockerfile`; long-running loop with a liveness port |
| Worker | Render (Free web service, Docker) | Frankfurt | `upgraderail-worker` | `services/worker/Dockerfile`; builds the Engine at the pinned commit |
| Artifact storage | Supabase Storage (S3 protocol) | `eu-west-1` | bucket `upgraderail-artifacts` (private) | `ARTIFACT_BACKEND=s3` |

Service definitions live in `render.yaml`. Public service URLs are recorded below only after they are deployed and verified.

### Environment variables

| Variable | Services | Source |
|---|---|---|
| `DATABASE_URL` | API, indexer, worker | Secret, entered in Render. Supabase session pooler URI, `sslmode=require` |
| `WEB_ORIGIN`, `AUTH_DOMAIN` | API | `https://upgraderail-console.vercel.app`, `upgraderail-console.vercel.app` |
| `API_ADDR` | API | `:10000` (Render's port) |
| `PORT` | indexer, worker | `10000`, serves `GET /health/live` |
| `ARTIFACT_BACKEND`, `ARTIFACT_S3_ENDPOINT`, `ARTIFACT_S3_REGION`, `ARTIFACT_S3_BUCKET` | API, worker | Public values in `render.yaml` |
| `ARTIFACT_S3_ACCESS_KEY_ID`, `ARTIFACT_S3_SECRET_ACCESS_KEY` | API, worker | Secret, entered in Render |
| `STELLAR_NETWORK`, `STELLAR_NETWORK_PASSPHRASE`, `STELLAR_RPC_URL` | indexer | `testnet`, `Test SDF Network ; September 2015`, `https://soroban-testnet.stellar.org` |
| `UPGRADERAIL_CONTROLLER_ID` | indexer | `CAJX4YE77N23K53MNJHYMCZIFXGMUEHSNZPHXAHDVZU5IYXUK4OQXTWS` (disposable verified Testnet controller) |
| `UPGRADERAIL_START_LEDGER` | indexer | `4969430` (ledger before the controller's first events; used only until a checkpoint exists) |
| `UPGRADERAIL_ENGINE_BIN`, `UPGRADERAIL_WORK_DIR` | worker | Set in the worker image |

### Free-tier limitations and behavior

- **Sleep:** Render free web services spin down after 15 minutes without inbound HTTP traffic and take about a minute to wake. The API wakes on request. The indexer and worker are polling loops that make no inbound requests, so **they sleep unless an external monitor pings `/health/live` at least every 14 minutes**. While asleep, the indexer falls behind (it resumes from its checkpoint and does not lose events) and queued analysis jobs wait.
- **This is a deliberate compromise.** Render has no free background workers. The indexer and worker are still long-running processes (not serverless handlers), run as web services only to satisfy the free tier's port requirement. A paid background worker removes the sleep behavior.
- **RPC history window:** Testnet RPC retains roughly 120k ledgers (about 7 days). `UPGRADERAIL_START_LEDGER=4969430` is only valid while it is inside that window (on 2026-10-07 the oldest served ledger was 4,955,089, leaving roughly 20 hours). After the indexer writes its first checkpoint the start ledger is no longer used. A fresh database after that window needs a newer controller or start ledger.
- **Restart behavior:** Render restarts a crashed container automatically. The indexer exits nonzero on configuration and invalid-response errors and resumes from `indexer_checkpoints`. The worker claims jobs with `FOR UPDATE SKIP LOCKED`.
- **Supabase free tier:** 500 MB database, 1 GB storage, 50 MB per-file limit, projects pause after a week of inactivity. Artifacts are capped at 8 MiB by the API.
- **Docker builds on the free tier:** the worker image compiles the Rust Engine, which can be slow on Render's free build resources.

## 2. Public Web Deployment Verification

The Web console has been deployed to Vercel and verified:
- **Live URL**: `https://upgraderail-console.vercel.app`
- **Deployment URL**: `https://upgraderail-console-o3ri6hpih-hollujays-projects.vercel.app`
- **Vercel Team**: `hollujays-projects`
- **Vercel Project**: `upgraderail-console`
- **Configuration**:
  - `NEXT_PUBLIC_STELLAR_NETWORK=testnet`
  - `NEXT_PUBLIC_STELLAR_NETWORK_PASSPHRASE="Test SDF Network ; September 2015"`
  - `NEXT_PUBLIC_STELLAR_RPC_URL="https://soroban-testnet.stellar.org"`
  - `NEXT_PUBLIC_CONTROLLER_ID="CAJX4YE77N23K53MNJHYMCZIFXGMUEHSNZPHXAHDVZU5IYXUK4OQXTWS"`
- **Verified Routes (HTTP 200 OK)**:
  - `GET https://upgraderail-console.vercel.app/`
  - `GET https://upgraderail-console.vercel.app/api/health` -> `{"status":"ok","service":"web"}`
  - `GET https://upgraderail-console.vercel.app/app`
  - `GET https://upgraderail-console.vercel.app/explore`
  - `GET https://upgraderail-console.vercel.app/how-it-works`
  - `GET https://upgraderail-console.vercel.app/security`
  - `GET https://upgraderail-console.vercel.app/developers`
  - `GET https://upgraderail-console.vercel.app/docs`
  - `GET https://upgraderail-console.vercel.app/app/fleets`
  - `GET https://upgraderail-console.vercel.app/app/upgrades`
  - `GET https://upgraderail-console.vercel.app/app/analyses`
- **Verified Security Headers**:
  - `Content-Security-Policy`: `default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https://soroban-testnet.stellar.org; frame-ancestors 'none';`
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`

## 3. Backend Deployment Record and Verification Evidence

Verified on 2026-10-07 against public URLs only, Stellar Testnet only. Repository HEAD at verification: `fac75e9` (CI green); later commits touch tests and docs only.

### Public services

| Service | URL | Deployed commit |
|---|---|---|
| Web (Vercel) | `https://upgraderail-console.vercel.app` | existing deployment, redeployed with `NEXT_PUBLIC_API_BASE_URL` |
| API (Render) | `https://upgraderail-api.onrender.com` | `cda225d` |
| Indexer (Render) | `https://upgraderail-indexer.onrender.com/health/live` | `5b58257` (redeployed after the schema existed) |
| Worker (Render) | `https://upgraderail-worker.onrender.com/health/live` | `cda225d` |
| Database | Supabase project `upgraderail-staging`, `eu-west-1`, PostgreSQL 17.11 | n/a |
| Artifact storage | Supabase Storage S3 endpoint, private bucket `upgraderail-artifacts` | n/a |

Render service IDs: API `srv-db3arl0m7kps73dlcoc0`, indexer `srv-db3arl0m7kps73dlcocg`, worker `srv-db3arl0m7kps73dlcobg`. Blueprint `upgraderail-staging` syncs `render.yaml` with auto-deploy off, so each deploy is a manual dashboard action.

### Health, CORS and auth (public API)

- `GET /health/live` and `GET /health/ready` return 200. Indexer and worker `/health/live` return 200.
- CORS preflight from `https://upgraderail-console.vercel.app` returns 204 with credentials allowed. A foreign origin returns 403.
- Sign-in, driven by a throwaway Testnet key: challenge issued (201); wrong signature rejected (401); SEP-53 signature accepted (200); the challenge cannot be replayed (401); the session is accepted with its cookie and invalid after logout. The cookie is `HttpOnly; Secure; SameSite=Lax`.
- Anonymous requests to session-protected upload return 401.

### Artifact upload and analysis (public API, real WASM fixtures)

- Uploading `fleet_v1`, `fleet_v2_compatible` and `fleet_v2_breaking` returned 201, and the server-computed SHA-256 matched the local SHA-256 for each. A non-WASM upload returned 400. An analysis naming a missing artifact returned 400.
- The worker, running the real Engine (`upgraderail 0.1.0`, protocol profile 28), read both artifacts from object storage and finished `fleet_v1` to `fleet_v2_compatible` as **READY** and `fleet_v1` to `fleet_v2_breaking` as **BLOCKED**. Reports and manifests are served by the API.
- Two earlier jobs failed with `object store unreachable` because the worker image lacked CA certificates. That was fixed in `cda225d` and they remain in the database as `failed`.

### Web to API

The deployed console issues `GET /api/v1/{controllers,fleets,proposals,upgrades,analyses}` to the public API from the Vercel origin (all 200), renders the indexed fleet, and logs no console errors. The CSP `connect-src` allows only `self`, the API origin and the Stellar Testnet RPC.

### Database evidence (`scripts/verify-staging-db.sh`, run by the operator)

- PostgreSQL 17.11; 14 tables in `public`; 20 non-primary-key indexes; constraints: 14 primary, 15 unique, 16 foreign, 5 check.
- Indexed data: 14 controller events, 1 fleet, 3 proposals, 3 approvals, 0 fleet upgrades. Duplicate events: 0.
- Checkpoint: `testnet-CAJX4YE7` at ledger 5073023.
- Analysis jobs: 1 ready, 1 blocked, 2 failed (the pre-fix jobs). Reports: 2. Manifests: 2. Manifest hash mismatches: 0. Artifact rows with non-positive size: 0.
- Public exposure: no `public` table is readable by `anon` or `authenticated`. The Supabase Data API shows 0 of 14 tables exposed, with automatic exposure of new tables off.

### Local validation

- `pnpm install --frozen-lockfile`, lint, typecheck, `pnpm test` (131 tests), `pnpm build` and `pnpm audit --prod` (no known vulnerabilities) pass.
- `pnpm --filter @upgraderail/web test:e2e` passes (55 tests) when run after `build:e2e`, as CI does. It runs against a mocked API and is not evidence about the deployed stack.
- `scripts/verify-fresh-migrations.sh` passes on an empty disposable database. It requires an empty database and must never be pointed at staging.
- `scripts/verify-contract-bindings.sh` regenerates the committed bindings with no diff.
- Go tests with a disposable PostgreSQL 16, the real Engine binary at the pinned commit and the contract fixtures: all pass. The live read-only Testnet tests pass against the public RPC. The only test still skipped is `TestS3AgainstRealServer`, which needs an S3 endpoint and credentials; the S3 signer is covered by the AWS documented signature vector and by the public staging upload and readback above.
- The Docker images were not built locally; Render's builds are the evidence for them.

## 4. Rollback Runbook

All services are managed from the Render dashboard (`dashboard.render.com`). Auto-deploy is off, so nothing changes unless an operator deploys.

### Web (Vercel)
1. Vercel project `upgraderail-console` -> Deployments -> pick the last good deployment -> Promote to Production (or `vercel rollback`).
2. To point the web app away from the API, remove or change `NEXT_PUBLIC_API_BASE_URL` in Settings -> Environment Variables and redeploy.

### API and worker (Render)
1. Service -> Deploys -> on the last good deploy choose Rollback. Render redeploys that image.
2. Verify `GET /health/ready` on the API and `GET /health/live` on the worker.
3. API and worker are stateless; rolling back does not change database rows or stored artifacts.

### Indexer (Render)
1. Pause safely: confirm no batch is mid-flight, then Service -> Settings -> Suspend. The indexer commits events and its checkpoint in one transaction, so stopping at any point leaves a consistent checkpoint.
2. Inspect progress: `SELECT * FROM indexer_checkpoints;`.
3. Resume: Resume the service (or roll back its deploy first). It continues from the saved cursor and does not duplicate events.
4. A fresh database needs `UPGRADERAIL_START_LEDGER` inside the RPC retention window (see section 1).

### Worker drain
1. Check `SELECT count(*) FROM analysis_jobs WHERE status = 'running';`. Wait until it is 0.
2. Suspend the worker. Queued jobs stay queued and run after resume.
3. Stopping during a running job may leave that job in `running`. Recovery of stale `running` jobs has not been verified; mark such a job failed by hand if needed.

### Database
- Migrations have no automated down path. The Supabase free plan has no managed backups, so take a manual dump before any schema change: `pg_dump "$DATABASE_URL" -Fc -f staging-pre-change.dump` (keep it outside the repository).
- Restore into a new Supabase project, apply nothing else, then update `DATABASE_URL` on the three Render services.
- `scripts/verify-fresh-migrations.sh` and `bootstrap-staging-db.sh` (without `SKIP_MIGRATIONS=1`) are for empty databases only.

### Artifact storage cleanup
- Objects are content-addressed under `<first two hex>/<sha256>` in bucket `upgraderail-artifacts`. Delete objects only after deleting the `analysis_jobs` and `artifacts` rows that reference them, otherwise analyses fail with a missing-artifact error.
- Supabase dashboard -> Storage -> `upgraderail-artifacts` to remove objects.

## 5. Security Audit
- No database credentials, keys or session secrets are in `NEXT_PUBLIC_*` variables or in the repository. Secret values are entered only in Render environment variables (`sync: false` in `render.yaml`).
- Cookies are `HttpOnly; Secure; SameSite=Lax`. HTTPS is used on every public endpoint. CORS allows only the web origin.
- S3 and transport errors never include the endpoint, object key or credentials, and they name the failure class.
- Log review: the Render application logs read during this pass contained no connection string or credential. A full log scan for the pre-rotation password has not been done (see section 6).

## 6. Known Limitations and Unverified Items

- **Overall status: NOT READY.**
- **Database password rotation not done.** The staging password was exposed in an operator terminal session during bootstrap. Rotate it in Supabase, update `DATABASE_URL` on the API, indexer and worker, then re-verify health, the indexer resuming from its checkpoint, worker job claiming, and that the old password is absent from logs.
- **Keep-alive monitors not created.** The indexer and worker are free web services and sleep after 15 minutes without inbound traffic (observed: the indexer took about 17 seconds to answer after sleeping, and its checkpoint stopped advancing). Create two 5-minute GET monitors on their `/health/live` URLs. Until then, indexing and analysis depend on someone waking the services.
- **Indexer restart behavior** was observed only through a Render redeploy that resumed indexing; a no-duplicate restart check beyond the zero-duplicate evidence above is not recorded.
- **Public-staging Freighter connect and a browser-signed governance write are untested.** Freighter was not available in the verification browser. Local Freighter governance writes were verified earlier (see `docs/testnet-verification.md`).
- **Mainnet:** not deployed and not performed.
- **Multi-approver browser verification and UpdatePolicy/UpgradeController browser verification:** not performed.
- **Free-tier limits:** Render free services sleep; Supabase free has no managed backups and pauses after a week of inactivity; Render free builds of the worker compile the Rust Engine.
