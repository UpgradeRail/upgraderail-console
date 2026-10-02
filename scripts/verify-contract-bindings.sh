#!/usr/bin/env bash
set -euo pipefail

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
wasm="$repo_root/../upgraderail-contracts/fixtures/wasm/upgrade_controller_v1.wasm"
committed="$repo_root/packages/contracts/generated"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT

if [[ ! -f "$wasm" ]]; then
  printf 'UpgradeController WASM was not found at %s\n' "$wasm" >&2
  exit 1
fi

stellar contract bindings typescript --wasm "$wasm" --output-dir "$temporary/generated"
diff --recursive --unified "$committed" "$temporary/generated"
