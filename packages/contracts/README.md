# UpgradeController bindings

`generated/` is produced from the `UpgradeController` WASM in the adjacent `upgraderail-contracts` checkout.

Generation source:

- contracts commit: `3a3c0dea136147c7fac02fc30ecc57e71ed8947d`
- file: `fixtures/wasm/upgrade_controller_v1.wasm`
- SHA-256: `ea773fd9e77c5b53b46922c1661f6479ac5561c381665d64f1cee1f786a109b3`
- generator: Stellar CLI `28.1.0`

This is a reproducible built-contract specification source. It is not a claim that the fixture hash is the current Testnet controller deployment hash. Deployment data remains in `upgraderail-contracts/deployments/testnet.json`.

Regenerate with `./scripts/generate-contract-bindings.sh`; verify freshness with `./scripts/verify-contract-bindings.sh`.
