# Staging Deployment Guide & Verification Record

Date: 2026-10-07  
Status: **Web Console Verified Live; Backend Services Unprovisioned on Public Cloud**

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

## 3. Backend Deployment Status & Constraints

The backend Go services (API, Worker, Indexer) and PostgreSQL database require persistent hosting (e.g. Render, Fly.io, Railway, or managed VM) and managed database hosting (e.g. Neon, Supabase).

- **Current Environment State**: No active credentials or tokens exist on this machine for external cloud platforms (Render, Fly, Railway, Supabase, Neon).
- **Rule Adherence**: Per project rules, fake deployment URLs are never fabricated. Only verified endpoints are reported.
- **Staging Automation**:
  - `scripts/verify-staging-env.sh`: Validates staging environment configurations, HTTPS origins, domain matching, and prevents secret leakage.
  - `scripts/bootstrap-staging-db.sh`: Automates migration execution and seed data injection for empty staging PostgreSQL instances.

## 4. Rollback Runbook

### Web Rollback
- Revert instantly via Vercel CLI:
  ```bash
  vercel alias set <previous-deployment-url> upgraderail-console.vercel.app
  ```
  or trigger rollback from the Vercel project dashboard.

### API & Worker Rollback
1. Stop running service container/process.
2. Deploy the previous binary or container image tag.
3. Restart the service and verify `/health/live` and `/health/ready`.

### Indexer Rollback
1. Send `SIGTERM` or `SIGINT` to allow graceful shutdown of the current batch.
2. Inspect `indexer_checkpoints` to confirm the last committed cursor.
3. Roll back the indexer binary.
4. Restart the indexer. The indexer will resume from `indexer_checkpoints` without duplicating journal rows.

### Database Rollback
- Migrations do not implement automated down migrations.
- Rollback requires restoring a pre-migration database snapshot/backup taken prior to running migrations.

## 5. Security Audit
- No database credentials, private keys, or session secrets are exposed in `NEXT_PUBLIC_*` variables.
- Deployed Vercel logs confirmed absence of sensitive tokens.
- Deployed web origin enforces strict CSP restricting connections to self and Stellar Testnet RPC.
