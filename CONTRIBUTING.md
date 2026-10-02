# Contributing

Use Node 24.21.0 and pnpm 12.8.2 for the web workspace. Use the Go version declared in `go.work` for services.

Run `pnpm lint`, `pnpm typecheck`, `pnpm test`, and `pnpm build` before submitting web changes. Run `go test ./...` and `go vet ./...` inside each changed Go service.

Do not add fabricated chain data, wallet addresses, or engine reports to product pages. Development fixtures must be clearly marked.
