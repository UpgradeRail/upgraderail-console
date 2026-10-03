# Deployment

The Next.js app can use its standalone build output. API, indexer, and worker are long-running Go services and need persistent PostgreSQL access; the indexer is not a request-time serverless function.

Required server variables are in `.env.example`. `NEXT_PUBLIC_*` values are browser-visible and must never contain database URLs, session secrets, or private RPC credentials.

Before deployment, configure an absolute `ARTIFACT_LOCAL_DIR`, `UPGRADERAIL_ENGINE_BIN`, PostgreSQL, Stellar RPC, controller ID, `AUTH_DOMAIN`, and `WEB_ORIGIN`. No production URLs or infrastructure have been configured in this checkout. The current artifact implementation is filesystem only; an S3-compatible store has not been implemented. `SESSION_SECRET` is present in the example configuration but the current database-backed session implementation does not read it.

## Services

- Web: build with `pnpm --filter @upgraderail/web build`; set `NEXT_PUBLIC_API_BASE_URL`, `NEXT_PUBLIC_STELLAR_NETWORK`, `NEXT_PUBLIC_STELLAR_RPC_URL`, and `NEXT_PUBLIC_CONTROLLER_ID` before building. The standalone server needs `.next/static` copied into `.next/standalone/apps/web/.next/static` after the build; otherwise HTML loads without CSS or JavaScript. Run `PORT=3000 HOSTNAME=0.0.0.0 node apps/web/.next/standalone/apps/web/server.js` from the repository root.
- API: run `services/api/cmd/api`; set `DATABASE_URL`, `AUTH_DOMAIN`, and `WEB_ORIGIN`. `WEB_ORIGIN` is the exact browser origin including scheme and port, for example `https://console.example.com`. `AUTH_DOMAIN` must equal its hostname. API startup fails when either value is missing or mismatched. Use HTTPS and keep web and API on the same site so the `SameSite=Lax` session cookie can be sent.
- Indexer: compile and run `services/indexer/cmd/indexer` as a supervised, long-running process. Set `DATABASE_URL`, `STELLAR_NETWORK`, `STELLAR_NETWORK_PASSPHRASE`, `STELLAR_RPC_URL`, `UPGRADERAIL_CONTROLLER_ID`, and `UPGRADERAIL_START_LEDGER`. The start ledger must be a safe ledger from the checked deployment record and is used only before a checkpoint exists. `INDEXER_POLL_INTERVAL` defaults to 5 seconds. The process verifies the RPC network, replays its ordered journal, fetches from the saved RPC cursor, and commits events and checkpoint in one database transaction. Temporary RPC transport errors and HTTP 429/5xx responses are retried at the poll interval without advancing the checkpoint. It exits nonzero on configuration, invalid RPC responses, or batch errors so a supervisor can restart it.
- Worker: run `services/worker/cmd/worker`; set `DATABASE_URL`, `ARTIFACT_LOCAL_DIR`, `UPGRADERAIL_WORK_DIR`, `UPGRADERAIL_ENGINE_BIN`, and a timeout appropriate for Engine analysis jobs.

## Data Plane

- PostgreSQL must be migrated before API, indexer, or worker processes start.
- Run fresh bootstrap verification before first deployment with `DATABASE_URL=<empty database> make verify-fresh-migrations`.
- The initial migration creates schema only. Insert a `networks` row for each configured network before auth challenges, sessions, or analysis jobs can be created. For Testnet, use `INSERT INTO networks (id, passphrase, rpc_url, protocol_target) VALUES ('testnet', 'Test SDF Network ; September 2015', 'https://soroban-testnet.stellar.org', 28);` after migration. A missing row makes challenge creation fail with `challenge_unavailable`.
- Insert a `controllers` row for each controller before starting the indexer, using the network and contract ID checked against the deployment record. The indexer fails clearly if the configured controller is absent. Migration `000002_indexer_event_order.sql` stores RPC event IDs for deterministic restart replay. A pre-migration journal with missing RPC event IDs cannot be resumed safely; rebuild it from a verified safe start ledger in a fresh database, then switch service traffic after comparison.
- Artifact storage must be durable and shared between the API path that records artifact keys and the worker path that reads them.
- `ARTIFACT_LOCAL_DIR` is acceptable only when the API and worker share a persistent filesystem. Otherwise, add and verify an object-store implementation before deployment.

## Secrets

- `DATABASE_URL`, private RPC headers, and future object-store credentials are server-only secrets. Treat `SESSION_SECRET` as server-only if configured, even though this version does not use it.
- `NEXT_PUBLIC_*` values are embedded in browser output and must contain only public network metadata and public endpoints.
- Browser CSP `connect-src` permits only the configured API and Stellar RPC origins in production. API credentialed CORS and logout require the exact `WEB_ORIGIN`.
- Do not log signed transaction XDR, session tokens, database URLs, RPC credentials, or artifact contents.

## Health Checks

- API live check: `GET /health/live`.
- API readiness check: `GET /health/ready`; this must fail when PostgreSQL is unavailable.
- Web health check: `GET /api/health` for the Next.js runtime.
- Worker readiness requires database connectivity, artifact storage access, and `UPGRADERAIL_ENGINE_BIN version`.
- Indexer readiness: the supervised process must remain alive after logging `indexer ready`. Check its last successful batch and `SELECT cursor, ledger_sequence, updated_at FROM indexer_checkpoints WHERE controller_id = '<controller row ID>';`. The RPC cursor can advance on an empty page while `ledger_sequence` remains the last event ledger. Stop and restart the process to confirm `resuming=true` and a stable event count. A separate HTTP readiness endpoint is not implemented.

## Process Model

- Run at least one API process and one web process per environment.
- Run indexer as a single writer per controller checkpoint. Multiple writers for the same controller are not supported without advisory locking or partitioned ownership. On batch failure, the process exits and the checkpoint stays at the previous committed cursor.
- Run workers horizontally only after confirming `FOR UPDATE SKIP LOCKED` job claiming against the production database.
- Workers need a writable temporary directory for per-job Engine configuration and manifests.

## Rollback

1. Stop indexer and worker processes first to prevent new writes.
2. Roll the web and API back to the previous build.
3. If the database schema changed, use the database backup taken immediately before migration; this repository does not yet provide down migrations.
4. Restart the API, then indexer, then worker.
5. Confirm health checks and compare the indexer checkpoint with the pre-rollback value before resuming queued jobs.

## Current Status

No public production or staging web, API, indexer, worker, database, artifact store, or URLs have been provisioned from this repository. The local production-like pass is recorded in `docs/deployment-verification.md`. The indexer currently journals events and maintains a checkpoint; it does not yet populate the API's fleet and proposal read-model tables. This local pass does not establish public deployment readiness.
