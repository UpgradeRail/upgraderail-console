# Engine integration

The worker invokes the configured `UPGRADERAIL_ENGINE_BIN` directly. It does not use a shell or interpolate user input into a command string.

For comparison jobs it runs:

```text
upgraderail compare --current <absolute artifact path> --candidate <absolute artifact path> --format json
```

The worker runner requires caller-provided absolute paths, applies a bounded timeout, passes a minimal environment containing `PATH`, and captures at most 4 MiB of stdout or stderr. Job claiming and temporary-directory lifecycle are not implemented yet. Exit code `1` is accepted only when valid Engine JSON states `BLOCKED`; the report is evidence, not a successful release decision.

The persisted report retains Engine status, findings, storage compatibility, and authorization behavior without rewriting evidence states. In particular, `NOT PROVEN BY STATIC ANALYSIS` and `NOT TESTED` remain distinct from passing results.

The source integration point is UpgradeRail Engine commit `78b385fd78e7dde034145f680a15f21e7071eda4`, whose CLI reports a Protocol 28 analysis profile.
