# UpgradeRail Console

UpgradeRail Console is the public website and browser-facing operations console for governed Soroban upgrades. It reads controller activity from the indexer, presents UpgradeRail Engine reports, and keeps transaction signing in the connected wallet.

The Console does not implement governance rules or a second WASM analyzer. Those responsibilities remain in `upgraderail-contracts` and `upgraderail-engine`.

## Local development

1. Copy `.env.example` to `.env` and set local values.
2. Run `pnpm install --frozen-lockfile`.
3. Run `pnpm dev`.

The web app is available at `http://localhost:3000`. API, indexer, and worker services are designed to run independently.

## Status

This repository is being implemented. Until an indexer is configured, public pages show explicit unavailable states rather than chain data.

## Related repositories

- `upgraderail-contracts`: on-chain UpgradeController governance.
- `upgraderail-engine`: Protocol 28 WASM analysis and manifest tooling.

See `docs/` for architecture, deployment, engine integration, and limitations.
