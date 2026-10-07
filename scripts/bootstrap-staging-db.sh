#!/usr/bin/env bash
# bootstrap-staging-db.sh applies migrations and seeds initial network and controller
# rows into a staging PostgreSQL database.
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 2
fi

network_id="${STELLAR_NETWORK:-testnet}"
network_passphrase="${STELLAR_NETWORK_PASSPHRASE:-Test SDF Network ; September 2015}"
network_rpc_url="${STELLAR_RPC_URL:-https://soroban-testnet.stellar.org}"
network_protocol_target="${STELLAR_PROTOCOL_TARGET:-28}"

controller_contract_id="${UPGRADERAIL_CONTROLLER_ID:-CAJX4YE77N23K53MNJHYMCZIFXGMUEHSNZPHXAHDVZU5IYXUK4OQXTWS}"
controller_start_ledger="${UPGRADERAIL_START_LEDGER:-5035000}"

psql_cmd=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)

echo "==> Applying migrations to database..."
for migration in database/migrations/*.sql; do
  echo "  Applying $migration..."
  "${psql_cmd[@]}" -f "$migration" >/dev/null
done

echo "==> Seeding network record ($network_id)..."
"${psql_cmd[@]}" -c "
INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
VALUES ('$network_id', '$network_passphrase', '$network_rpc_url', $network_protocol_target)
ON CONFLICT (id) DO UPDATE
SET passphrase = EXCLUDED.passphrase,
    rpc_url = EXCLUDED.rpc_url,
    protocol_target = EXCLUDED.protocol_target;
" >/dev/null

echo "==> Seeding staging controller record ($controller_contract_id)..."
"${psql_cmd[@]}" -c "
INSERT INTO controllers (id, network_id, contract_id, start_ledger)
VALUES ('${network_id}-${controller_contract_id:0:8}', '$network_id', '$controller_contract_id', $controller_start_ledger)
ON CONFLICT (network_id, contract_id) DO NOTHING;
" >/dev/null

echo "==> Verifying table presence..."
expected_tables=(
  networks controllers fleets proposals approvals controller_events fleet_upgrades
  artifacts analysis_jobs analysis_reports release_manifests indexer_checkpoints
  auth_challenges sessions
)

for table in "${expected_tables[@]}"; do
  exists="$("${psql_cmd[@]}" -At -c "SELECT to_regclass('public.${table}') IS NOT NULL;")"
  if [[ "$exists" != "t" ]]; then
    echo "ERROR: missing expected table: ${table}" >&2
    exit 1
  fi
done

echo "==> Staging database setup and bootstrap complete."
