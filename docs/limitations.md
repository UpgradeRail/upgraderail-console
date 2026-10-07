# Known limitations

## Indexer read-model projection

The indexer now projects `UpgradeController` events into the `fleets`, `proposals`, `approvals`, and `fleet_upgrades` read-model tables, and the API serves those tables (see `docs/indexing.md`). What remains unproven or intentionally absent:

- **Proposal `kind` and `manifest_hash` are populated only for executed proposals.** The `proposal_created` event carries neither. The indexer sets them from the execution event, so pending, cancelled, and expired proposals stay `NULL` in the API. The UI shows the live `get_proposal` values on the proposal page, but the API read model itself is not reconciled for those proposals: `Store.ReconcileProposal` exists and is tested, yet nothing calls it, and the indexer does not issue `get_proposal` simulations.
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

Rust `1.99.0` is installed on this host and can build the checked-out Engine workspace, which requires Rust `1.98.1` or later. A database-backed worker job invoking the real Engine binary was verified locally with committed contracts WASM fixtures.

The verification does not prove live Testnet current-versus-candidate simulation. The Engine report still preserves `NOT PROVEN BY STATIC ANALYSIS` and `NOT TESTED` evidence states where the Engine cannot prove storage compatibility or runtime authorization behavior.

## Governance write flows

The SDK builders (`buildCreateProposal`, `buildApprove`, `buildRevokeApproval`, `buildCancelProposal`, `buildExecuteProposal`) and their UI panels are covered by unit tests with a mocked RPC server and by browser tests against a mocked API. **No create, approve, revoke, cancel, or execute transaction has been signed or submitted from the browser on Testnet.** Only the earlier `maintain_controller` write is verified on-chain (`docs/testnet-verification.md`). The builders live in `apps/web/src/lib/governance-tx.ts`, not in `packages/sdk`, which is empty.

Eligibility shown in the UI mirrors the controller's rules, but the contract is the authority: every action is simulated before signing and a rejected simulation returns no transaction.

## Analysis and artifacts

- The API has **no artifact upload endpoint**. `POST /api/v1/analyses` takes existing artifact ids, and nothing in this repository writes `artifacts` rows from the product. The web UI therefore lists and displays analyses; it cannot start one.
- Artifact storage is filesystem-only (`ARTIFACT_LOCAL_DIR`), readable by the worker. S3-compatible storage was **not** implemented: no v1 requirement for it was found in this repository, and with no upload path there is nothing to store. If the API and worker do not share a persistent filesystem, an object-store backend with SHA-256 verification must be added before deployment.
- Analyses are not linked to fleets in the data model, so the fleet analysis route shows the shared analysis list rather than a per-fleet filter.
- A proposal can only be drafted from an analysis whose Engine status is not `BLOCKED`, whose manifest is persisted, and whose current WASM hash equals the fleet's indexed hash.
- The Engine report carries no runtime simulation scenarios unless configured; the worker does not configure them, so runtime simulation shows `NOT CONFIGURED`.
- Upgrade history shows ledger and transaction but no block timestamp, and cannot link an upgrade to the analysis that produced its manifest hash. Both are displayed as not recorded.

## Browser test coverage

Playwright (Chromium) runs the production build against a **mocked** Console API: route smoke tests, light/dark themes, reduced motion, keyboard navigation, and axe (WCAG 2.0/2.1 A and AA) on the main routes. It is not evidence about a deployed API, indexer, or Freighter. The Freighter extension is not automated: wallet connect and signing remain manual observations (`docs/wallet.md`).
