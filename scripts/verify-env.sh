#!/usr/bin/env bash
set -euo pipefail

node --version
pnpm --version
go version
if command -v psql >/dev/null 2>&1; then psql --version; fi
if command -v stellar >/dev/null 2>&1; then stellar version; fi
