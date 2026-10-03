# Wallet flow

The web app calls only documented Freighter APIs: `isConnected`, `getAddress`, `requestAccess`, `signMessage`, and `signTransaction`.

Private keys never leave Freighter. The Console uses the wallet address for display and sends an unsigned XDR transaction to Freighter only after a live controller read, simulation, and user confirmation are implemented. The browser code has signing helpers but no governance action currently exposes a signable transaction.

The server challenge uses a short-lived random nonce and verifies the Ed25519 signature against the Stellar public address using SEP-53: SHA-256 of `Stellar Signed Message:\n` followed by the challenge text. This matches Freighter's `signMessage` behavior and rejects signatures over the raw challenge text.

## Manual browser observation

On 2026-10-03, the Freighter 5.48.0 extension popup was opened in the existing Chrome profile while `http://127.0.0.1:3000/app` was active. The popup displayed an unlocked account. The console remained at “Checking wallet…” and “Network unavailable”; it did not open a connect, challenge-signing, or transaction-signing request. This verifies extension presence and popup access only. Network selection, connection, challenge/session flow, and transaction signing remain unverified.

The client failure was reproduced in Chrome DevTools Protocol against the same local origin. The browser console reported that Next.js inline bootstrap scripts were blocked by the static `script-src 'self'` policy, followed by a Next.js invariant error because `self.__next_r` was never initialized. That prevented client hydration, so the wallet effect never ran and the server-rendered “Checking wallet…” state never changed. The app now supplies a per-request CSP nonce through Next.js `src/proxy.ts`. Wallet detection also has a finite timeout and does not request account access during initial detection. The development CSP also permits HTTP API calls to local services; production API origins must use HTTPS.

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

Full manual Freighter browser-extension verification still requires the console wallet flow to become available, then:

1. Connect Freighter on Testnet.
2. Sign the server challenge and confirm the authenticated session cookie is set.
3. Build an unsigned controller transaction from fresh indexed state.
4. Open the Freighter sign flow and reject it; confirm the UI remains recoverable.
5. Repeat signing and confirm a signed transaction XDR is returned.
6. Submit only a safe Testnet transaction, then verify pending, confirmed, and failed polling states.
