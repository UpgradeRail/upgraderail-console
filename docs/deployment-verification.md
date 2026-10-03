# Deployment verification report

Date: 2026-10-03. Starting HEAD: `c594c94`. Environment: **local-production-like**, on the workstation only. No public production or staging deployment URL was available. The local processes and database are disposable and are not durable deployment infrastructure.

## Endpoints

- Web: `http://127.0.0.1:13000` (Next.js standalone production build)
- API: `http://127.0.0.1:18080` (Go API process)
- PostgreSQL: local PostgreSQL 16 on port 55439
- Stellar RPC: `https://soroban-testnet.stellar.org`

These are verification addresses, not deployment URLs. The browser bundle was built with the local API URL.

## Results

| Area | Result | Evidence and limit |
| --- | --- | --- |
| Web production build | Passed locally | `pnpm build`; standalone server returned 200 on `/`, `/product`, `/security`, `/docs`, `/explore`, `/app`, `/app/fleets`, `/app/upgrades`, and `/app/settings`. The bundle needed `.next/static` copied into the standalone tree. Headless Chrome then loaded the styled `/app` page. |
| Themes and reduced motion | Passed locally | Chrome DevTools emulated `prefers-reduced-motion: reduce`; the page loaded, and Light and Dark controls changed `document.documentElement.dataset.theme`. |
| Web to API configuration | Passed locally | CSP named the configured local API and Testnet RPC origins only. The API allowed a credentialed preflight from the configured web origin and rejected a different scheme. Public network reachability remains unverified. |
| API health and database | Passed locally | `/health/ready` returned 200 against freshly migrated PostgreSQL. |
| Migrations and schema | Passed locally | `make verify-fresh-migrations` on an empty PostgreSQL 16 database verified expected tables, indexes, and constraints. A `testnet` network row had to be inserted before auth challenges worked. |
| Auth/session/logout | Passed locally | An ephemeral Stellar Ed25519 key signed a SEP-53 challenge; challenge 201, verify 200, session 200, logout 204, revoked session 401. Cookie had HttpOnly, Secure, and SameSite=Lax. This was programmatic HTTP verification, not a deployed browser wallet session. |
| Worker process and Engine | Passed locally | The long-running Go worker consumed a real queued job using the UpgradeRail Engine 0.1.0 binary, then persisted a `READY` analysis report and a 1,570-byte release manifest with a SHA-256 hash. The real Engine integration tests also passed. |
| Artifact storage | Passed on local filesystem only | Files were written under `/tmp/upgraderail-artifacts`, read by the worker, and matched their recorded SHA-256 hashes. This temporary path is not durable deployment storage. No artifact URLs or storage credentials were logged. |
| Stellar RPC | Passed read-only | `TestFetchControllerEventsLive` fetched Testnet controller events from start ledger 4,969,430. |
| Indexer service and checkpoint | **Failed deployment gate** | The indexer executable logs a message and exits with code 0. It does not connect to PostgreSQL, run continuously, or resume/persist a checkpoint. The standalone RPC test does not cover those behaviors. |
| Logs and secrets | Partial | Examined local API, worker, web, and indexer process output; no secret values appeared. There is no deployed log stream to audit. |
| Rollback/runbook | Documented | See [deployment.md](deployment.md) for build packaging, migration, process model, health checks, and rollback order. A deployed rollback rehearsal was not possible. |

## Validation

`pnpm install --frozen-lockfile`, `pnpm lint`, `pnpm typecheck`, `pnpm test` (24 tests), `pnpm build`, Go service tests, Go vet, `make verify-fresh-migrations`, and `scripts/verify-contract-bindings.sh` passed. The real Engine job and live indexer RPC test passed. GitHub CI must be checked on the final report commit.

## Readiness

**NOT READY.** The indexer process and checkpoint behavior are unimplemented. A public staging or production environment, durable artifact storage, public HTTPS origins, deployed health checks, browser wallet auth on the deployed origin, and deployed log/rollback verification remain unverified. Local production-like checks do not constitute production deployment.
