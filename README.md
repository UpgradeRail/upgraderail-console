# UpgradeRail Console

UpgradeRail Console is the public website and browser-facing operations console for governed Soroban upgrades. It reads controller activity from the indexer, presents UpgradeRail Engine reports, and keeps transaction signing in the connected wallet.

The Console does not implement governance rules or a second WASM analyzer. Those responsibilities remain in `upgraderail-contracts` and `upgraderail-engine`.

## Local development

1. Copy `.env.example` to `.env` and set local values.
2. Run `pnpm install --frozen-lockfile`.
3. Run `pnpm dev`.

The web app is available at `http://localhost:3000`. API, indexer, and worker services are designed to run independently.

## Status

The console UI, API, indexer, and worker are implemented and verified locally against Stellar Testnet. No public staging or production deployment exists, and nothing here is evidence of one. Without a configured API and indexer, pages show explicit unavailable or empty states rather than chain data.

What works (locally verified; see `docs/deployment-verification.md` and `docs/testnet-verification.md`):

- Freighter connect, SEP-53 challenge signing, session creation, and transaction signing.
- Read-model pages (overview, activity, fleets, proposals, upgrade history) backed by the indexer projection.
- Preflight analysis pages that show Engine findings, a compatibility diff, and exact `NOT PROVEN` / `NOT TESTED` evidence states.
- Uploading a current/candidate WASM pair (`POST /api/v1/artifacts`, `/app/analyses/new`) and creating an analysis job from the resulting artifact IDs, which the worker then runs through the real Engine binary.
- Governance transaction builders and UI flows for create, approve, revoke, cancel, and execute. They simulate before signing and require explicit confirmation before a Testnet submit.

What is not proven or remains external: see `docs/limitations.md`.

## Related repositories

- `upgraderail-contracts`: on-chain UpgradeController governance.
- `upgraderail-engine`: Protocol 28 WASM analysis and manifest tooling.

See `docs/` for architecture, deployment, engine integration, and limitations.
