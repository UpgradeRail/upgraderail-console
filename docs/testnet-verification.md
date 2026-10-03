# Testnet read verification

Read-only verification ran on 2026-10-02 against the controller recorded by `upgraderail-contracts/deployments/testnet.json`.

- controller: `CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`
- RPC: `https://soroban-testnet.stellar.org`
- controller policy: two approvers, threshold 2, timelock 12 ledgers, proposal lifetime 720 ledgers
- governance epoch: 1
- controller version: 1
- fleet: `8863397c695ebe1dd9f1922afd511aee68999753d13cd61345767487564dc820` (`upgraderail-testnet-shared`), created at ledger 4,969,465
- current fleet WASM hash: `67a657dd6c64f4255f5e928782a058aaeff30c14c7ca530d2604c554d30d76c7`
- proposal 2: `Executed`, with two approvals; its manifest commitment is `cd8b679e2215a53c3bda375de6112d0d3c0203d017dfcdcfa5e1a63ef6ef088a`

The deployment record reports observed Testnet Protocol 29. These reads do not alter UpgradeRail Engine’s Protocol 28 analysis profile or the Mainnet Protocol 28 production target.

Verification commands used read-only RPC calls and did not submit transactions:

```text
stellar contract invoke --rpc-url https://soroban-testnet.stellar.org \
  --network-passphrase "Test SDF Network ; September 2015" \
  --contract-id CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
  --source-account GB6NGKUWJFXWAVE5K3UNLGTPTBGD3TDVOZAA3ITOBIMUR25SGMLGKRA6 \
  --send no -- get_policy
```

The indexer read the same controller events from Stellar RPC with `xdrFormat: "json"` and applied them to a local migrated PostgreSQL database:

```text
DATABASE_URL=postgresql://upgraderail:upgraderail@127.0.0.1:55433/upgraderail_migration_verify \
STELLAR_RPC_LIVE=1 \
STELLAR_RPC_URL=https://soroban-testnet.stellar.org \
UPGRADERAIL_CONTROLLER_ID=CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3 \
UPGRADERAIL_START_LEDGER=4969430 \
go test ./services/indexer/internal/store -run TestLiveReadOnlyTestnetControllerProjection -count=1 -v
```

No wallet connection, write transaction, or Testnet current/candidate fleet-pair simulation was performed. The Freighter popup was opened in Chrome on 2026-10-03, but the console stayed at “Checking wallet…” and “Network unavailable”; no wallet connection or signing was verified.
