# UpgradeRail Console

UpgradeRail Console is the public website and browser-facing operations console for governed Soroban upgrades. It reads controller activity from the indexer, presents UpgradeRail Engine reports, and keeps transaction signing in the connected wallet.

The Console does not implement governance rules or a second WASM analyzer. Those responsibilities remain in `upgraderail-contracts` and `upgraderail-engine`.

## Local development

1. Copy `.env.example` to `.env` and set local values.
2. Run `pnpm install --frozen-lockfile`.
3. Run `pnpm dev`.

The web app is available at `http://localhost:3000`. API, indexer, and worker services are designed to run independently.

## Status

The console UI, API, indexer, and worker are implemented and verified against Stellar Testnet, locally and on a public Testnet staging deployment. Without a configured API and indexer, pages show explicit unavailable or empty states rather than chain data.

- Repository implementation: **READY**
- Public Testnet staging: **READY WITH CAVEATS**
- Production/Mainnet launch: **NOT PERFORMED**

Public Testnet staging (Vercel web, Render API, indexer and worker, Supabase PostgreSQL and Storage):

- Web: https://upgraderail-console.vercel.app
- API: https://upgraderail-api.onrender.com/health/live
- Indexer: https://upgraderail-indexer.onrender.com/health/live
- Worker: https://upgraderail-worker.onrender.com/health/live

Staging runs on free-tier hosting with UptimeRobot keep-alive checks that were observed only briefly. A real Freighter sign-in on the public origin, multi-approver flows, and `UpdatePolicy`/`UpgradeController` proposals from the browser are not yet verified. Staging is Testnet only and is not evidence of production or Mainnet readiness. See `docs/staging-deployment.md` for the evidence and caveats.

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
