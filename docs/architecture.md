# Architecture

```text
Browser → Next.js web app → Console API → PostgreSQL
                                  ↑             ↑
Stellar RPC → indexer ────────────┘             │
                                                │
Artifact store → worker → UpgradeRail Engine ───┘
```

`UpgradeController` remains the source of governance authority. The schema reserves an idempotent event journal and projection tables for the indexer. The API read layer exposes those projected models under `/api/v1`; it does not infer chain events by polling state.

The web app uses Server Components for static product and unavailable states. Browser-only theme controls are isolated in a Client Component. Wallet transaction signing and live-contract reconciliation are not implemented yet and must not be represented as available.
