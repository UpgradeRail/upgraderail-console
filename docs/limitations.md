# Known limitations

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
