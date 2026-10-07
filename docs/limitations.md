# Known limitations

## Indexer read-model projection

The indexer now projects `UpgradeController` events into the `fleets`, `proposals`, `approvals`, and `fleet_upgrades` read-model tables, and the API serves those tables (see `docs/indexing.md`). What remains unproven or intentionally absent:

- **Proposal `kind` and `manifest_hash` are filled from the execution event when there is one, and by read-only contract reconciliation otherwise.** The `proposal_created` event carries neither field. For an executed proposal the indexer fills them from the execution event (`fleet_created`/`fleet_upgraded`/`policy_updated`/`controller_upgraded`). For a pending, cancelled, or expired proposal — which has no execution event — the indexer's reconciliation pass (`services/indexer/internal/reconcile`, see `docs/indexing.md`) reads them from the live contract's `get_proposal` via a read-only `simulateTransaction` call, once per indexer poll cycle, in a bounded batch, with retry for transient RPC failures. This has been verified against the real Testnet controller (`docs/testnet-verification.md`), but that controller currently has no live unexecuted proposal, so the RPC-filled path is proven via a deliberate before/after test rather than an organic pending-proposal read. An `UpdatePolicy` proposal has no `manifest_hash` at all in the contract's own data — this stays `NULL` permanently, by design, not because reconciliation failed. The UI's own live `get_proposal` read on the proposal page remains in place as an additional freshness layer, not as a required substitute for the now-reconciled API field.
- **Fleet membership and counts are never fabricated.** The `fleets` table only ever reflects fleets the indexer has actually observed a `fleet_created` event for, with `current_wasm_hash` only ever set from an observed `fleet_created`/`fleet_upgraded` event. There is no "total fleet count" derived from anything other than indexed rows.
- **Verified against real Testnet data, not fixtures.** The read-model projection was run end-to-end against the real deployed Testnet controller (`CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`) from a freshly migrated, empty local database. See `docs/deployment-verification.md` for the observed row counts.

## Live Testnet candidate simulation

There is no separately addressable live Testnet pair for a current pre-upgrade fleet and a candidate fleet. The contracts deployment record demonstrates governed upgrade evidence, including executed upgrade history, but it does not establish that distinct current/candidate fleet pair.

The Console must not call that evidence a live current-versus-candidate Testnet simulation. Engine runtime scenarios are shown only when they were configured and produced by the worker.

## Deployment state

This checkout does not have configured production API, worker, indexer, database, artifact-store, or public deployment URLs. Public routes show explicit unavailable states until an indexer-backed API is deployed.

## Browser wallet verification

The Freighter 5.48.0 popup was opened in the existing Chrome profile on 2026-10-03 and showed an unlocked account. A Chrome DevTools Protocol reproduction found the static CSP blocked Next.js bootstrap scripts, preventing client hydration. A nonce CSP and finite wallet detection states are now implemented. Connect, challenge signing, and transaction signing still require verification in the existing profile. Playwright now covers the page and state matrix against a mocked API, but the Freighter extension itself is not automated.

## Worker Engine execution

Rust `1.99.0` is installed on this host and can build the checked-out Engine workspace, which requires Rust `1.98.1` or later. A database-backed worker job invoking the real Engine binary was verified locally with committed contracts WASM fixtures, including end to end from a real `POST /api/v1/artifacts` upload through to a persisted report and manifest (see `docs/api.md`).

The verification does not prove live Testnet current-versus-candidate simulation. The Engine report still preserves `NOT PROVEN BY STATIC ANALYSIS` and `NOT TESTED` evidence states where the Engine cannot prove storage compatibility or runtime authorization behavior.

## Governance write flows

The SDK builders (`buildCreateProposal`, `buildApprove`, `buildRevokeApproval`, `buildCancelProposal`, `buildExecuteProposal`) and their UI panels are covered by unit tests with a mocked RPC server and by browser tests against a mocked API. **No create, approve, revoke, cancel, or execute transaction has been signed or submitted from the browser on Testnet.** Only the earlier `maintain_controller` write is verified on-chain (`docs/testnet-verification.md`). The builders live in `apps/web/src/lib/governance-tx.ts`, not in `packages/sdk`, which is empty.

Eligibility shown in the UI mirrors the controller's rules, but the contract is the authority: every action is simulated before signing and a rejected simulation returns no transaction.

## Analysis and artifacts

- The API now has an artifact upload endpoint (`POST /api/v1/artifacts`, see `docs/api.md`) and the web UI can upload a current/candidate WASM pair and create an analysis job from them (`/app/analyses/new`). This closes the previous gap where nothing in the product could write an `artifacts` row.
- Artifact storage is still filesystem-only (`ARTIFACT_LOCAL_DIR`), shared by the API (which writes uploads) and the worker (which reads them to run the Engine) via `services/shared/artifactstore`. S3-compatible storage was **not** implemented: no v1 requirement for it was found in this repository. If the API and worker do not run on the same filesystem (e.g. separate hosts/containers without a shared volume), the worker cannot read what the API wrote, and a job will fail with a clear "artifact" error rather than silently hanging — but this is still not production-ready storage. An object-store backend with SHA-256 verification should be added before a real multi-host deployment.
- The upload endpoint's WASM check is a minimal magic-byte check (`\0asm`), not full module validation; a well-formed-looking but invalid module would still be accepted at upload time and would only fail once the worker runs the Engine on it.
- Analyses are not linked to fleets in the data model, so the fleet analysis route shows the shared analysis list rather than a per-fleet filter.
- A proposal can only be drafted from an analysis whose Engine status is not `BLOCKED`, whose manifest is persisted, and whose current WASM hash equals the fleet's indexed hash.
- The Engine report carries no runtime simulation scenarios unless configured; the worker does not configure them, so runtime simulation shows `NOT CONFIGURED`.
- Upgrade history shows ledger and transaction but no block timestamp, and cannot link an upgrade to the analysis that produced its manifest hash. Both are displayed as not recorded.

## Dependency audit

`pnpm audit --prod` reports no known vulnerabilities in production dependencies. A full `pnpm audit` (including devDependencies) reports one high-severity advisory: `braces` (GHSA-vfj7-8cjw-p6xm, stack-exhaustion denial of service via deeply nested patterns), pulled in transitively through `eslint-config-next` → `@next/eslint-plugin-next` → `fast-glob` → `micromatch` → `braces`. No patched version exists upstream as of this check. This is a dev-only lint/tooling dependency, not reachable from any production runtime path (API, worker, indexer, or the built web app); it is not resolved, and this should not be reported as zero advisories — only as zero in the production dependency graph.

## Browser test coverage

Playwright (Chromium) runs the production build against a **mocked** Console API: route smoke tests, light/dark themes, reduced motion, keyboard navigation, and axe (WCAG 2.0/2.1 A and AA) on the main routes. It is not evidence about a deployed API, indexer, or Freighter. The Freighter extension is not automated: wallet connect and signing remain manual observations (`docs/wallet.md`).
