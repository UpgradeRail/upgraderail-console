# Wallet flow

The web app calls only documented Freighter APIs: `isConnected`, `getAddress`, `requestAccess`, `signMessage`, and `signTransaction`.

Private keys never leave Freighter. The Console builds and simulates an unsigned `maintain_controller` transaction from live Testnet state, then sends its XDR to Freighter. It requires a separate explicit disposable-account confirmation before submission.

The server challenge uses a short-lived random nonce and verifies the Ed25519 signature against the Stellar public address using SEP-53: SHA-256 of `Stellar Signed Message:\n` followed by the challenge text. This matches Freighter's `signMessage` behavior and rejects signatures over the raw challenge text.

## Manual browser observation

On 2026-10-03, Freighter 5.48.0 was used in the existing Chrome profile at `http://127.0.0.1:3000/app`, with the local API at `http://127.0.0.1:8080` and a disposable migrated PostgreSQL database. The extension was unlocked on Testnet. The console detected Freighter, displayed “Freighter available · TESTNET”, and offered Connect. After approval, Freighter returned a SEP-53 challenge signature. The API verified it, consumed the challenge, created a session, and the browser's authenticated `GET /api/v1/auth/session` succeeded before the UI displayed “Connected · TESTNET”. The database showed one consumed challenge and one active session after the first successful attempt; a second approval also completed successfully.

A separate challenge-signing request was opened in Freighter and its popup was visually checked: it showed the UpgradeRail authentication text, the selected Testnet account, and Test Net. Clicking Cancel left that challenge unused and changed the console to “Request declined: The user rejected this request” with a “Try again” control. No private key, full signature, session cookie, or challenge nonce was recorded in this document.

On 2026-10-03 the same Chrome profile at `http://127.0.0.1:3000/app/settings` connected again and completed authenticated sign-in against the local API. The settings page built and simulated a real `maintain_controller` transaction against Stellar Testnet. Freighter displayed a **Confirm Transaction** prompt naming the Testnet wallet, network, and 2.2633237 XLM maximum fee. Cancel returned “Request declined: The user rejected this request. No transaction was submitted.” A fresh draft was built from ledger 4,999,619; approving the Freighter prompt returned signed XDR, and the console displayed “Signed XDR returned by Freighter and checked against the unsigned transaction. It has not been submitted.” Its transaction hash was `3db06ccb9ab61c8c26c4b8281ab7e88243a5a5045e584d176519183cd8c47c42`. This is a hash of an **unsubmitted** signed transaction, not an on-chain transaction ID. Submit was left disabled because the disposable-account checkbox was not selected. No Testnet write, polling, ledger confirmation, or transaction receipt was verified.

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

On 2026-10-03 the browser then exercised the submit path with the same Testnet account. The first submission was rejected before inclusion with `txBadSeq`; the console displayed that RPC diagnostic and public RPC returned `NOT_FOUND` for the rejected hash. Investigation found that the two read-only simulations had advanced the shared in-memory `Account` sequence via Stellar SDK `TransactionBuilder.build()`. Read simulations now use copies of the account, with a regression test for the sequence values.

After that fix, the browser rebuilt, signed, submitted, and polled a fresh maintenance transaction. The console displayed **Confirmed on Stellar Testnet** in ledger **4,999,846**. Public Stellar RPC returned `SUCCESS` for hash `f7bda7274e21e1cebe6ca939218ce92bc47961fcdccd46c9082f5d96c1c371e0`. The decoded envelope contained one `maintain_controller` invocation of `CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3`. The browser account was confirmed as disposable through the submit control. This verifies the real browser build, Freighter sign, Testnet submit, polling, and confirmed display path. A live on-chain failed transaction display remains untested; the pre-inclusion RPC rejection display was observed.

## Governance transactions

The governance panels (create, approve, revoke, cancel, execute) use the same Freighter path as the maintenance flow: build and simulate an unsigned transaction from live Testnet state, re-read that state and check the draft is under two minutes old, sign in Freighter, verify the returned transaction hash matches the simulated one, then require an explicit confirmation checkbox before submitting and polling the result. Pending, confirmed, failed, and user-rejected states are shown. Eligibility comes from a live read-only simulation of the controller, not from the indexed projection. These panels have not been exercised against Freighter on Testnet; see `docs/limitations.md`.
