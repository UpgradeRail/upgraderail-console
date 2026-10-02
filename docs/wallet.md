# Wallet flow

The web app calls only documented Freighter APIs: `isConnected`, `getAddress`, `requestAccess`, `signMessage`, and `signTransaction`.

Private keys never leave Freighter. The Console uses the wallet address for display and sends an unsigned XDR transaction to Freighter only after a live controller read, simulation, and user confirmation are implemented. The browser code has signing helpers but no governance action currently exposes a signable transaction.

The server challenge uses a short-lived random nonce and verifies the Ed25519 signature against the Stellar public address. Browser-extension verification is not yet performed in this environment.
