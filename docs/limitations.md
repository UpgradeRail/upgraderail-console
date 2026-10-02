# Known limitations

## Live Testnet candidate simulation

There is no separately addressable live Testnet pair for a current pre-upgrade fleet and a candidate fleet. The contracts deployment record demonstrates governed upgrade evidence, including executed upgrade history, but it does not establish that distinct current/candidate fleet pair.

The Console must not call that evidence a live current-versus-candidate Testnet simulation. Engine runtime scenarios are shown only when they were configured and produced by the worker.

## Deployment state

This checkout does not have configured production API, worker, indexer, database, artifact-store, or public deployment URLs. Public routes show explicit unavailable states until an indexer-backed API is deployed.

## Browser wallet verification

Freighter browser-extension verification has not run in this environment. The required browser automation executable is also unavailable, so visual browser verification remains unverified despite a successful local HTTP response and production build.

## Local worker Engine execution

This host provides Rust 1.97.1. The checked-out Engine workspace requires Rust 1.98.1 or later, and the supplied Engine verification evidence uses Rust 1.99.0. The worker integration code compiles and validates Engine JSON, but a database-backed worker job invoking the local Engine binary could not run on this host.
