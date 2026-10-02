# UI system

The web app uses semantic CSS tokens for surface, raised surface, border, primary text, muted text, accent, and danger values. Light, dark, and system theme choices are persisted locally. The theme control has an accessible pressed state.

The landing page uses the UpgradeRail process diagram to show contracts moving through analysis, simulation, approval, and readiness. Console routes distinguish loading, API error, empty, and indexed-data table states. Motion is limited and disabled for `prefers-reduced-motion`.

`agent-browser` is not installed in this environment, so browser verification used Chrome headless and HTTP route checks instead.

Verified on 2026-10-02:

- desktop home screenshot: `/tmp/upgraderail-home.png`
- desktop console shell screenshot: `/tmp/upgraderail-console.png`
- mobile explorer screenshot: `/tmp/upgraderail-explore.png`
- dark preference screenshot: `/tmp/upgraderail-dark.png`
- reduced motion preference screenshot: `/tmp/upgraderail-reduced-motion.png`
- HTTP 200 route checks for `/`, `/explore`, `/app`, `/app/fleets`, `/app/upgrades`, `/app/activity`, and `/app/settings`

The verified routes rendered public product content, explorer loading state, console empty state, fleet/proposal loading states, wallet checking state, network unavailable state, and theme controls. Freighter extension interaction was not verified.
