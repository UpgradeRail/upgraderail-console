# Known limitations

## Indexer read-model projection

The indexer now projects `UpgradeController` events into the `fleets`, `proposals`, `approvals`, and `fleet_upgrades` read-model tables, and the API serves those tables (see `docs/indexing.md`). What remains unproven or intentionally absent:

- **Proposal `kind` and `manifest_hash` are not derived from events.** The `proposal_created` event carries only `proposal_id`, `proposer`, `governance_epoch`, and `expires_ledger`. The proposal's kind (`CreateFleet`/`UpgradeFleet`/`UpdatePolicy`/`UpgradeController`) and its manifest hash live inside the `ProposalKind` variant returned by the contract's `get_proposal` read method, not in any event. `proposals.kind` and `proposals.manifest_hash` are `NULL` on every indexed proposal row for this reason. The API and UI must treat a null `kind` as "not yet known," not as "no kind."
- **No live read-only contract reconciliation was implemented in this pass.** A reconciliation step could fill the gap above by calling `get_proposal` read-only against the deployed controller. Building and testing a correct hand-rolled Soroban `simulateTransaction` invocation (XDR construction and decoding) in Go, with no existing dependency for it in this codebase, was judged too large and too easy to get subtly wrong to include safely in this pass. Every other projected field (fleets, controller policy/version, approvals, upgrades) is emitted directly by an event and does not need this.
- **Fleet membership and counts are never fabricated.** The `fleets` table only ever reflects fleets the indexer has actually observed a `fleet_created` event for, with `current_wasm_hash` only ever set from an observed `fleet_created`/`fleet_upgraded` event. There is no "total fleet count" derived from anything other than indexed rows.
- **Verified against real Testnet data, not fixtures.** The read-model projection was run end-to-end against the real deployed Testnet controller (`CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`) from a freshly migrated, empty local database. See `docs/deployment-verification.md` for the observed row counts.

## Live Testnet candidate simulation

There is no separately addressable live Testnet pair for a current pre-upgrade fleet and a candidate fleet. The contracts deployment record demonstrates governed upgrade evidence, including executed upgrade history, but it does not establish that distinct current/candidate fleet pair.

The Console must not call that evidence a live current-versus-candidate Testnet simulation. Engine runtime scenarios are shown only when they were configured and produced by the worker.

## Deployment state

This checkout does not have configured production API, worker, indexer, database, artifact-store, or public deployment URLs. Public routes show explicit unavailable states until an indexer-backed API is deployed.

## Browser wallet verification

The Freighter 5.48.0 popup was opened in the existing Chrome profile on 2026-10-03 and showed an unlocked account. A Chrome DevTools Protocol reproduction found the static CSP blocked Next.js bootstrap scripts, preventing client hydration. A nonce CSP and finite wallet detection states are now implemented. Connect, challenge signing, and transaction signing still require verification in the existing profile. The required browser automation executable is unavailable, and the full browser page and state matrix remains unverified.

## Worker Engine execution

Rust `1.99.0` is installed on this host and can build the checked-out Engine workspace, which requires Rust `1.98.1` or later. A database-backed worker job invoking the real Engine binary was verified locally with committed contracts WASM fixtures.

The verification does not prove live Testnet current-versus-candidate simulation. The Engine report still preserves `NOT PROVEN BY STATIC ANALYSIS` and `NOT TESTED` evidence states where the Engine cannot prove storage compatibility or runtime authorization behavior.
