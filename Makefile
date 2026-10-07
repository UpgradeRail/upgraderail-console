.PHONY: install dev build lint test go-test vet verify-env verify-staging-env verify-fresh-migrations smoke-test

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
	go test ./services/api/... ./services/indexer/... ./services/shared/artifactstore/... ./services/worker/...

vet:
	go vet ./services/api/... ./services/indexer/... ./services/shared/artifactstore/... ./services/worker/...

verify-env:
	./scripts/verify-env.sh

verify-staging-env:
	./scripts/verify-staging-env.sh

verify-fresh-migrations:
	./scripts/verify-fresh-migrations.sh

smoke-test:
	./scripts/smoke-test.sh
