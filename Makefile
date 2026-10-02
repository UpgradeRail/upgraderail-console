.PHONY: install dev build lint test go-test vet verify-env smoke-test

install:
	pnpm install --frozen-lockfile

dev:
	pnpm dev

build:
	pnpm build

lint:
	pnpm lint

test:
	pnpm test

go-test:
	go test ./services/api/... ./services/indexer/... ./services/worker/...

vet:
	go vet ./services/api/... ./services/indexer/... ./services/worker/...

verify-env:
	./scripts/verify-env.sh

smoke-test:
	./scripts/smoke-test.sh
