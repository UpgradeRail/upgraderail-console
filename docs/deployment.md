# Deployment

The Next.js app can use its standalone build output. API, indexer, and worker are long-running Go services and need persistent PostgreSQL access; the indexer is not a request-time serverless function.

Required server variables are in `.env.example`. `NEXT_PUBLIC_*` values are browser-visible and must never contain database URLs, session secrets, or private RPC credentials.

Before deployment, configure an absolute `ARTIFACT_LOCAL_DIR`, `UPGRADERAIL_ENGINE_BIN`, PostgreSQL, Stellar RPC, controller ID, `SESSION_SECRET`, `AUTH_DOMAIN`, and `WEB_ORIGIN`. No production URLs or infrastructure have been configured in this checkout. The current artifact implementation is filesystem only; an S3-compatible store has not been implemented.

## Services

- Web: build with `pnpm --filter @upgraderail/web build`; set `NEXT_PUBLIC_API_BASE_URL`, `NEXT_PUBLIC_STELLAR_NETWORK`, `NEXT_PUBLIC_STELLAR_RPC_URL`, and `NEXT_PUBLIC_CONTROLLER_ID` before building. The standalone server needs `.next/static` copied into `.next/standalone/apps/web/.next/static` after the build; otherwise HTML loads without CSS or JavaScript. Run `PORT=3000 HOSTNAME=0.0.0.0 node apps/web/.next/standalone/apps/web/server.js` from the repository root.
- API: run `services/api/cmd/api`; set `DATABASE_URL`, `SESSION_SECRET`, `AUTH_DOMAIN`, and `WEB_ORIGIN`. `WEB_ORIGIN` is the exact browser origin including scheme and port, for example `https://console.example.com`. API startup fails when it is missing or malformed. Use HTTPS for deployed browser and API origins.
- Indexer: run `services/indexer/cmd/indexer`; set `DATABASE_URL`, `STELLAR_RPC_URL`, network passphrase, controller ID, and a safe start ledger or cursor.
- Worker: run `services/worker/cmd/worker`; set `DATABASE_URL`, `ARTIFACT_LOCAL_DIR`, `UPGRADERAIL_WORK_DIR`, `UPGRADERAIL_ENGINE_BIN`, and a timeout appropriate for Engine analysis jobs.

## Data Plane

- PostgreSQL must be migrated before API, indexer, or worker processes start.
- Run fresh bootstrap verification before first deployment with `DATABASE_URL=<empty database> make verify-fresh-migrations`.
- The initial migration creates schema only. Insert a `networks` row for each configured network before auth challenges, sessions, or analysis jobs can be created. For Testnet, use `INSERT INTO networks (id, passphrase, rpc_url, protocol_target) VALUES ('testnet', 'Test SDF Network ; September 2015', 'https://soroban-testnet.stellar.org', 28);` after migration. A missing row makes challenge creation fail with `challenge_unavailable`.
- Artifact storage must be durable and shared between the API path that records artifact keys and the worker path that reads them.
- `ARTIFACT_LOCAL_DIR` is acceptable only when the API and worker share a persistent filesystem. Otherwise, add and verify an object-store implementation before deployment.

## Secrets

- `DATABASE_URL`, `SESSION_SECRET`, private RPC headers, and object-store credentials are server-only secrets.
- `NEXT_PUBLIC_*` values are embedded in browser output and must contain only public network metadata and public endpoints.
- Browser CSP `connect-src` permits only the configured API and Stellar RPC origins in production. API credentialed CORS and logout require the exact `WEB_ORIGIN`.
- Do not log signed transaction XDR, session tokens, database URLs, RPC credentials, or artifact contents.

## Health Checks

- API live check: `GET /health/live`.
- API readiness check: `GET /health/ready`; this must fail when PostgreSQL is unavailable.
- Web health check: `GET /api/health` for the Next.js runtime.
- Worker readiness requires database connectivity, artifact storage access, and `UPGRADERAIL_ENGINE_BIN version`.
- Indexer readiness is currently unavailable: the indexer executable only logs a message and exits. The live RPC test verifies read-only `getEvents`, but does not verify process startup, database checkpoint resume, or continuous indexing.

## Process Model

- Run at least one API process and one web process per environment.
- Run indexer as a single writer per controller checkpoint unless advisory locking or partitioned controller ownership is added.
- Run workers horizontally only after confirming `FOR UPDATE SKIP LOCKED` job claiming against the production database.
- Workers need a writable temporary directory for per-job Engine configuration and manifests.

## Rollback

1. Stop indexer and worker processes first to prevent new writes.
2. Roll the web and API back to the previous build.
3. If the database schema changed, use the database backup taken immediately before migration; this repository does not yet provide down migrations.
4. Restart the API, then indexer, then worker.
5. Confirm health checks and compare the indexer checkpoint with the pre-rollback value before resuming queued jobs.

## Current Status

No public production or staging web, API, indexer, worker, database, artifact store, or URLs have been provisioned from this repository. The local production-like pass is recorded in `docs/deployment-verification.md`. It does not establish deployment readiness.
