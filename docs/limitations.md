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

The SDK builders (`buildCreateProposal`, `buildApprove`, `buildRevokeApproval`, `buildCancelProposal`, `buildExecuteProposal`) and their UI panels are covered by unit tests with a mocked RPC server and by browser tests against a mocked API. The builders live in `apps/web/src/lib/governance-tx.ts`, not in `packages/sdk`, which is empty.

**All five actions (create, approve, revoke, cancel, execute) have now been signed in a real Freighter popup and submitted to Stellar Testnet**, confirmed on public RPC, and reflected in the indexer/API projection; see `docs/testnet-verification.md` for transaction hashes, ledgers, and the disposable controller used. This exercised a real bug in indexer reconciliation (a fresh, unapproved proposal's `Option<u32>` fields decode as the bare JSON string `"void"`, not a nested object) and a real guard in the web app (`VERIFIED_CONTROLLER_ID`/`VERIFIED_CONTROLLER_WASM_HASH` in `governance-tx.ts` previously hard-coded to the single shared evidence controller with no override); both are now fixed, documented below and in `docs/testnet-verification.md`.

Two things this pass did **not** prove:
- A multi-approver (`threshold > 1`) flow with two distinct Freighter-held keys signing independently. The disposable controller used has a single approver (`threshold: 1`); the shared evidence controller's two real approvers' keys are not available in this environment (see `docs/testnet-verification.md`). The contract logic for multi-approver threshold counting is exercised only by unit tests and by the pre-existing executed proposals on the shared controller (which were approved via CLI, not the browser).
- `UpdatePolicy` and `UpgradeController` proposal kinds from the browser. Only `CreateFleet` was exercised (the UI's create-proposal panel otherwise only drafts `UpgradeFleet`; see below). These two kinds share the same `create_proposal`/`approve`/`execute` code paths already verified, but have not themselves been built, signed, and submitted.

Eligibility shown in the UI mirrors the controller's rules, but the contract is the authority: every action is simulated before signing and a rejected simulation returns no transaction.

### The create-proposal UI only drafted `UpgradeFleet`, not `CreateFleet`

Before this pass, `CreateProposalPanel` (`apps/web/src/components/create-proposal-panel.tsx`) could only draft an `UpgradeFleet` proposal from an analysis, and showed a dead-end "No indexed fleets" message when a controller had none — there was no way to originate a fleet's first release from the browser at all. A `CreateFleetPanel`, a `buildCreateFleetKind` draft builder (`apps/web/src/lib/proposal-draft.ts`), and a fleet tag input were added so a controller with zero indexed fleets gets a working `CreateFleet` form instead. The fleet id is derived deterministically via SHA-256 of the tag. The fleets list is now also filtered by the configured controller (`FleetRecord.controller_id`), which a prior build did not do, so a controller with no fleets of its own previously still showed the shared controller's fleet and the `UpgradeFleet` path.

### Verified-controller guard

`configuredContractId()`/`requireVerifiedNetwork()` in `governance-tx.ts` used to hard-fail for any contract id other than the single shared evidence controller. This pass generalized it to a short allowlist (`VERIFIED_CONTROLLERS`) that still defaults to, and still only ever includes, pre-registered Testnet contract ids with their exact expected WASM hash — it does not accept an arbitrary configured id. The disposable controller added for this verification pass is explicitly commented as disposable-Testnet-only, never Mainnet or production.

### A `CreateFleet` proposal requires its initial WASM to already be uploaded on-chain

`execute_proposal` on a `CreateFleet` proposal failed simulation the first time with `HostError: Error(Storage, MissingValue)` / `"Wasm does not exist"`, because the analysis used to draft the proposal referenced a WASM hash that had never been uploaded to Testnet as contract code (only its bytes existed locally, hashed by the Engine). Uploading that WASM (`stellar contract upload`) resolved it. This is correct contract behavior, not a bug: a fleet's initial executable must actually exist on-chain before a `CreateFleet` proposal naming it can execute.

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

## Staging status (2026-10-08)

Public staging is **NOT READY**. The API monitor exists, but the indexer and worker UptimeRobot keep-alive monitors are missing, so on the free Render tier those two services sleep after 15 minutes without traffic and indexing and analysis stall until they are woken. See `docs/staging-deployment.md` section 3.1. The staging proof (artifact upload, a ready job, a blocked job, persisted report and manifest with matching hash) was run through the public API with a throwaway Testnet key; a real Freighter sign-in on the public origin is still untested. Go tests skip the live-Testnet-RPC and real-S3 cases unless `STELLAR_RPC_LIVE=1` or `UPGRADERAIL_TEST_S3_ENDPOINT` is set; these were not enabled in the 2026-10-08 run.
