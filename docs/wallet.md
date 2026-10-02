# Wallet flow

The web app calls only documented Freighter APIs: `isConnected`, `getAddress`, `requestAccess`, `signMessage`, and `signTransaction`.

Private keys never leave Freighter. The Console uses the wallet address for display and sends an unsigned XDR transaction to Freighter only after a live controller read, simulation, and user confirmation are implemented. The browser code has signing helpers but no governance action currently exposes a signable transaction.

The server challenge uses a short-lived random nonce and verifies the Ed25519 signature against the Stellar public address. Browser-extension verification is not yet performed in this environment.

The wallet wrapper has unit coverage for:

- wallet missing
- wrong network
- user rejects access or signing
- challenge signing
- transaction signing
- submit/signing failure
- pending transaction state
- confirmed transaction state
- failed transaction state

Manual Freighter browser-extension verification still requires a browser profile with Freighter installed:

1. Connect Freighter on Testnet.
2. Sign the server challenge and confirm the authenticated session cookie is set.
3. Build an unsigned controller transaction from fresh indexed state.
4. Open the Freighter sign flow and reject it; confirm the UI remains recoverable.
5. Repeat signing and confirm a signed transaction XDR is returned.
6. Submit only a safe Testnet transaction, then verify pending, confirmed, and failed polling states.
