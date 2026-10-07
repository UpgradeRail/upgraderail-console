# Deployment verification report

Date: 2026-10-03. Starting HEAD: `c594c94`. Environment: **local-production-like**, on the workstation only. No public production or staging deployment URL was available. The local processes and database are disposable and are not durable deployment infrastructure. The Vercel connector returned no teams and required a team ID to list projects; this checkout has no linked provider project or public service host.

## Endpoints

- Web: `http://127.0.0.1:13000` (Next.js standalone production build)
- API: `http://127.0.0.1:18080` (Go API process)
- PostgreSQL: local PostgreSQL 16 on port 55434, database `upgraderail_deploy_indexer_verify`
- Stellar RPC: `https://soroban-testnet.stellar.org`

These are verification addresses, not deployment URLs. The browser bundle was built with the local API URL.

## Results

| Area | Result | Evidence and limit |
| --- | --- | --- |
| Web production build | Passed locally | A separate worktree preserved the existing development server. `pnpm build` produced a standalone server with `.next/static` copied into its package. It returned 200 on `/`, `/product`, `/security`, `/docs`, `/explore`, `/app`, `/app/fleets`, `/app/upgrades`, `/app/settings`, and `/api/health`. |
| Themes and reduced motion | Passed locally | Headless Chrome loaded the production `/app` route with CSS and JavaScript. Light rendered `rgb(246, 245, 240)` and Dark rendered `rgb(16, 20, 19)` on the body; `prefers-reduced-motion: reduce` matched and the page stayed usable. No screenshot was taken. |
| Web to API configuration | Passed locally | Browser `fetch` from the production page to `http://127.0.0.1:18080/health/ready` returned 200. Production CSP `connect-src` included only self, that API origin, and `https://soroban-testnet.stellar.org`. The API allowed a credentialed preflight from `http://127.0.0.1:13000` and rejected `http://localhost:13000` with 403. Public network reachability remains unverified. |
| API health and database | Passed locally | `/health/ready` returned 200 against freshly migrated PostgreSQL. |
| Migrations and schema | Passed locally | `make verify-fresh-migrations` applied both migrations to two separate empty PostgreSQL 16 databases and verified tables, indexes, and constraints, including ordered RPC event IDs. A network and controller row were explicitly seeded before service startup. |
| Auth/session/logout | Partial on this stack | Challenge returned 201; an unauthenticated session returned 401; same-origin logout returned 204; foreign-origin logout returned 403. The earlier local production-like run completed SEP-53 challenge verification, session 200, logout, and revoked-session 401 with an ephemeral test identity at the same origins. A signed session was not repeated on this new database or a public origin. Session cookies are HttpOnly, Secure, and SameSite=Lax. |
| Worker process and Engine | Passed locally | The compiled long-running worker consumed `staging-engine-job` using the real UpgradeRail Engine 0.1.0 binary and real WASM fixtures. The job and report were `ready`/`READY`; a 1,582-byte release manifest was stored with SHA-256 `93293770edcf57d6491f834f44a8178504d644c5636a835b3fdeb4844e79482d`, independently recomputed from stored bytes. |
| Artifact storage | Passed on local filesystem only | The current and candidate WASM files were written under `/tmp/upgraderail-deploy-artifacts`, read by the worker, and matched their source SHA-256 hashes after readback. This temporary path is not durable or shared storage. No artifact URLs or storage credentials were logged. |
| Stellar RPC | Passed read-only | The compiled indexer verified the Testnet RPC passphrase and fetched real UpgradeController events beginning at ledger 4,969,430. No Testnet write was made during this deployment pass. |
| Indexer service and checkpoint | Passed locally, with limits | The compiled long-running indexer committed 12 real events with RPC order IDs and a checkpoint in PostgreSQL. After a clean stop and restart, it logged `resuming=true`, fetched from the saved cursor, and retained exactly 12 journal rows. An empty page advanced the cursor without lowering the last event ledger. A later Testnet DNS failure caused the original process to exit; the indexer now retries transport and HTTP 429/5xx failures without changing its checkpoint. The rebuilt process resumed from the saved cursor and continued polling successfully. Retry classification has automated coverage; recovery from another live DNS outage has not been observed. Missing `DATABASE_URL` exited nonzero with a clear error. The indexer still journals events only; it does not populate the API fleet/proposal read-model tables or expose an HTTP health endpoint. |
| Logs and secrets | Partial | Examined local API, worker, web, and indexer process output; no database password, session token, signed XDR, or artifact content appeared. There is no deployed log stream to audit. Temporary process launch commands contained the disposable local DB credential; production service managers must inject secrets without command-line arguments. |
| Rollback/runbook | Documented | See [deployment.md](deployment.md) for build packaging, migration, process model, health checks, and rollback order. A deployed rollback rehearsal was not possible. |

## Validation

`pnpm install --frozen-lockfile`, `pnpm lint`, `pnpm typecheck`, `pnpm test` (24 tests), `pnpm build`, Go service tests, Go vet, `make verify-fresh-migrations`, `scripts/verify-contract-bindings.sh`, and `pnpm audit --prod` passed. The compiled worker's real Engine job and the compiled indexer's live RPC/restart run passed. GitHub CI must be checked on the final report commit.

## Readiness

**NOT READY.** No public staging or production environment, durable shared artifact storage, or public HTTPS origin has been provisioned. Deployed browser wallet auth, deployed health/log checks, and rollback rehearsal remain unverified. The indexer journal and checkpoint now work, but API fleet/proposal read-model persistence is still absent. Local production-like checks do not constitute production deployment.

## Addendum: 2026-10-07 local re-verification

Local only, on a disposable PostgreSQL 16 container; still no public deployment. Go tests and vet pass, including the real-Engine worker tests (not skipped); a fresh-schema migration check and `scripts/verify-contract-bindings.sh` pass; `pnpm audit --prod` reports no known vulnerabilities; `pnpm lint`, `typecheck`, `test` (121 tests), `build`, and the Playwright suite (47 tests, mocked API) pass. A live run of the compiled indexer against Testnet produced the projection recorded in `docs/testnet-verification.md`, and the compiled API served those rows with `kind` and `manifest_hash` populated for both executed proposals. Public deployment, deployed logs, rollback rehearsal, durable shared artifact storage, and Testnet governance writes remain unverified.
