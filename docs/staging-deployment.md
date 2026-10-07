# Staging Deployment Guide & Verification Record

Date: 2026-10-07  
Status: **Web Console Verified Live; Backend Services Unprovisioned on Public Cloud**

## 1. Staging Architecture

The planned staging topology consists of:
- **Web App**: Hosted on Vercel at `https://upgraderail-console.vercel.app`. Next.js 15 standalone application, monorepo root-aware build with frozen lockfile.
- **API Service**: Long-running Go service (`services/api/cmd/api`) requiring persistent PostgreSQL access, cookie sessions (`SameSite=Lax`, `Secure`, `HttpOnly`), and CORS restricted to `https://upgraderail-console.vercel.app`.
- **Worker Service**: Long-running Go worker (`services/worker/cmd/worker`) with real UpgradeRail Engine binary, processing analysis jobs from the database queue (`FOR UPDATE SKIP LOCKED`).
- **Indexer Service**: Supervised long-running Go process (`services/indexer/cmd/indexer`) polling Stellar Testnet RPC (`https://soroban-testnet.stellar.org`), recording events into `controller_events` journal and advancing `indexer_checkpoints`.
- **Database**: PostgreSQL 15+ with migrations 000001 through 000004 applied.
- **Artifact Storage**: Persistent filesystem (`ARTIFACT_LOCAL_DIR`) shared between API and Worker.

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
