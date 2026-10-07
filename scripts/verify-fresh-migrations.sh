#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 2
fi

psql_cmd=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)

table_count="$("${psql_cmd[@]}" -At -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';")"
if [[ "$table_count" != "0" ]]; then
  echo "expected an empty public schema, found ${table_count} tables" >&2
  exit 1
fi

for migration in database/migrations/*.sql; do
  "${psql_cmd[@]}" -f "$migration" >/dev/null
done

expected_tables=(
  networks controllers fleets proposals approvals controller_events fleet_upgrades
  artifacts analysis_jobs analysis_reports release_manifests indexer_checkpoints
  auth_challenges sessions
)

for table in "${expected_tables[@]}"; do
  exists="$("${psql_cmd[@]}" -At -c "SELECT to_regclass('public.${table}') IS NOT NULL;")"
  if [[ "$exists" != "t" ]]; then
    echo "missing table: ${table}" >&2
    exit 1
  fi
done

expected_indexes=(
  analysis_jobs_status_created_idx
  controller_events_controller_ledger_idx
  controller_events_rpc_event_id_idx
  sessions_address_network_idx
  proposals_needs_reconciliation_idx
)

for index in "${expected_indexes[@]}"; do
  exists="$("${psql_cmd[@]}" -At -c "SELECT to_regclass('public.${index}') IS NOT NULL;")"
  if [[ "$exists" != "t" ]]; then
    echo "missing index: ${index}" >&2
    exit 1
  fi
done

expected_constraints=(
  analysis_jobs_status_check
  artifacts_size_bytes_check
  auth_challenges_check
  controller_events_network_id_transaction_hash_event_index_key
  controllers_network_id_contract_id_key
  fleets_controller_id_fleet_hash_key
  fleets_controller_id_tag_key
  proposals_approval_count_check
  proposals_controller_id_proposal_id_key
  release_manifests_sha256_key
  sessions_token_hash_key
)

for constraint in "${expected_constraints[@]}"; do
  exists="$("${psql_cmd[@]}" -At -c "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND conname = '${constraint}');")"
  if [[ "$exists" != "t" ]]; then
    echo "missing constraint: ${constraint}" >&2
    exit 1
  fi
done

echo "fresh migrations verified"
