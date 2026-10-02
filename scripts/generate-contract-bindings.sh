#!/usr/bin/env bash
set -euo pipefail

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
wasm="$repo_root/../upgraderail-contracts/fixtures/wasm/upgrade_controller_v1.wasm"
output="$repo_root/packages/contracts/generated"

if [[ ! -f "$wasm" ]]; then
  printf 'UpgradeController WASM was not found at %s\n' "$wasm" >&2
  exit 1
fi

stellar contract bindings typescript --wasm "$wasm" --output-dir "$output" --overwrite
