# Wallet flow

The web app calls only documented Freighter APIs: `isConnected`, `getAddress`, `requestAccess`, `signMessage`, and `signTransaction`.

Private keys never leave Freighter. The Console uses the wallet address for display and sends an unsigned XDR transaction to Freighter only after a live controller read, simulation, and user confirmation are implemented. The browser code has signing helpers but no governance action currently exposes a signable transaction.

The server challenge uses a short-lived random nonce and verifies the Ed25519 signature against the Stellar public address using SEP-53: SHA-256 of `Stellar Signed Message:\n` followed by the challenge text. This matches Freighter's `signMessage` behavior and rejects signatures over the raw challenge text.

## Manual browser observation

On 2026-10-03, Freighter 5.48.0 was used in the existing Chrome profile at `http://127.0.0.1:3000/app`, with the local API at `http://127.0.0.1:8080` and a disposable migrated PostgreSQL database. The extension was unlocked on Testnet. The console detected Freighter, displayed “Freighter available · TESTNET”, and offered Connect. After approval, Freighter returned a SEP-53 challenge signature. The API verified it, consumed the challenge, created a session, and the browser's authenticated `GET /api/v1/auth/session` succeeded before the UI displayed “Connected · TESTNET”. The database showed one consumed challenge and one active session after the first successful attempt; a second approval also completed successfully.

A separate challenge-signing request was opened in Freighter and its popup was visually checked: it showed the UpgradeRail authentication text, the selected Testnet account, and Test Net. Clicking Cancel left that challenge unused and changed the console to “Request declined: The user rejected this request” with a “Try again” control. No private key, full signature, session cookie, or challenge nonce was recorded in this document. The transaction signing prompt, signed transaction return, and Testnet submit/poll lifecycle were not exercised because the console does not yet expose a controller transaction builder.

The client failure was reproduced in Chrome DevTools Protocol against the same local origin. The browser console reported that Next.js inline bootstrap scripts were blocked by the static `script-src 'self'` policy, followed by a Next.js invariant error because `self.__next_r` was never initialized. That prevented client hydration, so the wallet effect never ran and the server-rendered “Checking wallet…” state never changed. The app now supplies a per-request CSP nonce through Next.js `src/proxy.ts`. Wallet detection also has a finite timeout and does not request account access during initial detection. The development CSP permits HTTP API calls to local services; production API origins must use HTTPS. The API verifier originally checked a raw Ed25519 signature instead of SEP-53's prefixed hash. A second failure occurred because issuing and reloading the challenge formatted the same timestamp in different timezones; challenge text is now canonicalized to UTC.

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

Remaining browser verification:

1. Build an unsigned controller transaction from fresh indexed state.
2. Open the Freighter transaction sign flow and reject it; confirm the UI remains recoverable.
3. Repeat transaction signing and confirm a signed transaction XDR is returned.
4. Submit only a safe Testnet transaction, then verify pending, confirmed, and failed polling states.
