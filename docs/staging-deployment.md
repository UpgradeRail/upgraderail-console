# Staging Deployment Guide & Verification Record

Date: 2026-10-08 (originally recorded 2026-10-07)  
Status: **Web console and backend services are publicly deployed and verified after the `DATABASE_URL` rotation, and all three keep-alive monitors exist (section 3.1). Remaining blockers for READY are listed in section 6.**

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

### Post-rotation verification (2026-10-08)

The staging database password was rotated in Supabase and `DATABASE_URL` was updated on the API, indexer and worker (URL-encoded). Everything below was done through the public console and public API, with a throwaway Testnet key for sign-in; no local API or local database was used for the staging proof.

- **Service health.** API `/health/live` and `/health/ready` return 200; `/health/ready` performs a real database ping. Indexer and worker `/health/live` return 200 (these are static and do not touch the database). The API log history shows `password authentication failed (SQLSTATE 28P01)` startup failures before the final restart at 04:18 UTC (2026-10-08) and none after it. Those pre-fix lines include the Supabase username, project ref and pooler host, but no password.
- **Indexer.** Render logs show `indexer batch committed events=0` every 10 seconds for the controller after the rotation, and a search for `ERROR` over the last hour returned nothing. Public Testnet RPC had no controller events after ledger 5,073,023, so `events=0` is expected.
- **Artifact upload proof.** Public `POST /api/v1/artifacts` returned 201 for `fleet_v1` (2298 bytes), `fleet_v2_compatible` (2523 bytes) and `fleet_v2_breaking` (805 bytes). The server-computed SHA-256 matched the local SHA-256 for each. Re-uploads deduplicate to the same artifact id.
- **Ready analysis job proof.** `fleet_v1` to `fleet_v2_compatible`: `POST /api/v1/analyses` returned 202 `queued`; the worker claimed it and it reached **ready** in about one second (job `fc3e6905a44d1768f05d021b4207767cccd27b21e568fc08`, started 04:39:48 UTC, finished 04:39:49 UTC). Engine: `upgraderail 0.1.0`, protocol profile 28.
- **Blocked analysis job proof.** `fleet_v1` to `fleet_v2_breaking` reached **blocked** (job `75e0f8bd7878d299f292d8f44d68b428b38eb1fe962e9afa`, started 04:39:49 UTC, finished 04:39:49 UTC), also through the real Engine.
- **Worker public staging proof.** The worker claimed both jobs, read both artifacts from Supabase object storage, ran the real Engine, and persisted the report and manifest for each, with no manual intervention. This is the first job processed by the worker after the `DATABASE_URL` rotation. The worker emits no per-job log lines, so its logs contain only startup output and no artifact payloads or secrets.
- **Manifest hash verification.** For both jobs, `GET /api/v1/analyses/{id}/manifest` returned bytes whose SHA-256 equals the returned `sha256` (ready job `05ab525ffdb2c8c870fd4c8f976f047ebd836d9ccd36bccf8d4caa5e14d82bb9`; blocked job `e58fa67df23ece0d992c5991683981a66c2dd4790c637e7ea2af719417b49cd2`). Reports are served at `/report` (200).
- **UI proof.** `https://upgraderail-console.vercel.app/app/analyses/fc3e6905a44d1768f05d021b4207767cccd27b21e568fc08` shows "Engine status: READY", engine version, both WASM hashes, the same manifest hash, and two info findings (`FUNC004`, `SPEC010`). Storage compatibility, authorization behavior and runtime simulation are shown as not proven / not tested / not configured. The blocked job was not opened in the UI.
- **Not covered by this proof.** Public-staging Freighter sign-in was not exercised (a throwaway key signed the SEP-53 challenge, as before). A real Freighter sign-in on the public origin remains untested.

### 3.1 Keep-alive monitors (UptimeRobot)

Verified on 2026-10-08 in the UptimeRobot dashboard (free plan, one account) and from Render's logs.

| Monitor name | URL | Interval | Status at verification |
| --- | --- | --- | --- |
| `upgraderail-api` | `https://upgraderail-api.onrender.com/health/live` | 5 min | Up 55 min, 100%, 0 incidents |
| `upgraderail-indexer` | `https://upgraderail-indexer.onrender.com/health/live` | 5 min | Up 24 min after one incident, 99.65% |
| `upgraderail-worker` | `https://upgraderail-worker.onrender.com/health/live` | 5 min | Up 27 min, 100%, 0 incidents |

- **Names.** The API monitor was renamed from its URL to `upgraderail-api`.
- **Method is HEAD, not GET.** On the free plan the HTTP method selector is a premium feature and is locked to **HEAD** (confirmed by opening the API monitor's Advanced settings; nothing was saved). The other two monitors were created the same way and were not opened individually. The three `/health/live` handlers answer HEAD with 200 (checked with `curl -I`), and any inbound request counts as activity for Render, so HEAD is sufficient for keep-alive. A GET monitor needs a paid UptimeRobot plan.
- **Alerts** go to the configured UptimeRobot alert email (the UptimeRobot account email, shown on the API monitor); confirm it is the intended recipient. No SMS, voice, status page or integration is configured, and the test e-mail was not sent.
- **Indexer incident.** The indexer monitor's first check returned **502 for about 4 minutes**, then recovered by itself. This is consistent with the check waking a sleeping free instance (a manual curl at that time took 13 seconds). Render showed no failed instance or restart in that period.
- **Sleep observation.** After the monitors had run for about 20 minutes with no manual requests from the operator, all three stayed up, the indexer logged `batch committed` every 10 seconds with no gap on a single instance (`2s8mb`, 05:39 to 05:47 UTC, the part of the log read), and there was no new incident. That is about one and a half sleep windows, not a long soak: it shows the monitors keep the services awake over this window, not that they will for days.
- **Residual risk.** UptimeRobot's free tier has no SLA; if its checks stop, or its single-region checks are blocked, Render will sleep the indexer and worker again. Render's free instances also restart periodically and are subject to monthly free-instance hour limits.

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
- Re-run on 2026-10-08 against HEAD `07777ae` (CI is green on that commit):
  - `pnpm --filter @upgraderail/web test:e2e`: **55 of 55 passed**. The first `next build` stalled with no progress and was terminated; `apps/web/.next` was cleared and the single retry compiled and passed. It runs against a mocked API and is not evidence about the deployed stack.
  - Go, normal mode (no environment, as CI runs it): **57 passed, 0 failed, 26 skipped**. 21 skip because `DATABASE_URL` is not set (two of those also need `UPGRADERAIL_ENGINE_BIN` and `UPGRADERAIL_CONTRACT_FIXTURE_DIR`), 4 because `STELLAR_RPC_LIVE=1` is not set, and 1 (`TestS3AgainstRealServer`) because `UPGRADERAIL_TEST_S3_ENDPOINT` is not set.
  - Go with a disposable PostgreSQL 16, the Engine binary (reports `upgraderail 0.1.0`, protocol profile 28; the Engine checkout was at the pinned commit `78b385f`, but the binary's build date is older, so it was not rebuilt for this run) and the contract fixtures: **78 passed, 0 failed, 5 skipped**. The DB tests and the real worker/Engine tests ran. The remaining skips are the 4 live-Testnet-RPC tests (`STELLAR_RPC_LIVE` left unset) and `TestS3AgainstRealServer` (no S3 test endpoint).
  - `scripts/verify-fresh-migrations.sh`: passes on an empty disposable database. `scripts/verify-contract-bindings.sh`: passes with no diff.
  - The disposable container `ur-proof-pg` (127.0.0.1:55499) was stopped and removed afterwards; no other container or volume was touched.
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

### Rotating `DATABASE_URL`
1. Rotate the password in Supabase, then URL-encode it and set the new `DATABASE_URL` on the API, indexer and worker (Render -> service -> Environment). Deploys are manual.
2. Verify, in this order: API `/health/ready` returns 200 (a real database ping); indexer logs show `batch committed` lines and no `SQLSTATE 28P01`; submit one analysis through the public API and confirm it reaches `ready` or `blocked`. The indexer and worker `/health/live` endpoints do not touch the database, so a 200 from them does not prove the new credential works.
3. A bad credential makes the API exit at startup (`database startup failed ... SQLSTATE 28P01`) while Render keeps retrying; roll the environment value back and redeploy.
4. Never paste the connection string into chat, tickets, or this repository.

### Keep-alive monitors
- UptimeRobot monitors `upgraderail-api`, `upgraderail-indexer` and `upgraderail-worker` ping each `/health/live` every 5 minutes (section 3.1). If an alert fires for the indexer or worker, `curl -I` the URL (a cold start takes about 15 seconds and may return 502 first), then confirm indexer `batch committed` logs resume. If UptimeRobot itself is unavailable, wake them the same way by hand.
- Do not point monitors at write endpoints or add secrets to monitor URLs.

### Artifact storage cleanup
- Objects are content-addressed under `<first two hex>/<sha256>` in bucket `upgraderail-artifacts`. Delete objects only after deleting the `analysis_jobs` and `artifacts` rows that reference them, otherwise analyses fail with a missing-artifact error.
- Supabase dashboard -> Storage -> `upgraderail-artifacts` to remove objects.

## 5. Security Audit
- No database credentials, keys or session secrets are in `NEXT_PUBLIC_*` variables or in the repository. Secret values are entered only in Render environment variables (`sync: false` in `render.yaml`).
- Cookies are `HttpOnly; Secure; SameSite=Lax`. HTTPS is used on every public endpoint. CORS allows only the web origin.
- S3 and transport errors never include the endpoint, object key or credentials, and they name the failure class.
- Log review: the Render application logs read during this pass contained no connection string or credential. A full log scan for the pre-rotation password has not been done (see section 6).

## 6. Known Limitations and Unverified Items

- **Overall status: READY for Stellar Testnet staging use, with the caveats below. Not production and not Mainnet.** The earlier blockers (database password rotation, keep-alive monitors) are closed, but the keep-alive soak is short, the monitors use HEAD, and public-origin Freighter sign-in is untested.
- **Database password rotation done (2026-10-08).** Health, indexer resume and a worker job (ready and blocked) were verified after the rotation (section 3). A full log scan for the old password was not done; the Render log views read contained no credential.
- **Keep-alive monitors exist but are lightly verified.** All three 5-minute HEAD monitors are up (section 3.1). They use HEAD, not GET (GET is premium), alert to the configured UptimeRobot alert email (confirm it is intended), and were observed for only about 20 minutes. One 502 incident occurred on the indexer's first check. A multi-day soak has not been done.
- **Indexer restart behavior** was observed only through a Render redeploy that resumed indexing; a no-duplicate restart check beyond the zero-duplicate evidence above is not recorded.
- **Public-staging Freighter connect and a browser-signed governance write are untested.** Freighter was not available in the verification browser. Local Freighter governance writes were verified earlier (see `docs/testnet-verification.md`).
- **Mainnet:** not deployed and not performed.
- **Multi-approver browser verification and UpdatePolicy/UpgradeController browser verification:** not performed.
- **Free-tier limits:** Render free services sleep; Supabase free has no managed backups and pauses after a week of inactivity; Render free builds of the worker compile the Rust Engine.
