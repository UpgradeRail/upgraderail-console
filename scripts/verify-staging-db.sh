#!/usr/bin/env bash
# verify-staging-db.sh is a read-only evidence report for a deployed staging
# database: schema objects, indexed data counts, the indexer checkpoint,
# stored manifest hashes versus their bytes, and whether Supabase's anon and
# authenticated roles can reach the tables. It never writes and never prints
# DATABASE_URL.
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 2
fi

psql_cmd=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -X -At -F ' | ')
section() { printf '\n== %s\n' "$1"; }

section "server"
"${psql_cmd[@]}" -c "SHOW server_version;"

section "tables in public"
"${psql_cmd[@]}" -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';"

section "indexes in public (name | table)"
"${psql_cmd[@]}" -c "SELECT indexname, tablename FROM pg_indexes WHERE schemaname='public' AND indexname NOT LIKE '%_pkey' ORDER BY tablename, indexname;"

section "constraints in public (type | count) p=primary u=unique f=foreign c=check"
"${psql_cmd[@]}" -c "SELECT contype, count(*) FROM pg_constraint WHERE connamespace='public'::regnamespace GROUP BY contype ORDER BY contype;"

section "indexed data"
"${psql_cmd[@]}" -c "SELECT 'controller_events', count(*) FROM controller_events
UNION ALL SELECT 'fleets', count(*) FROM fleets
UNION ALL SELECT 'proposals', count(*) FROM proposals
UNION ALL SELECT 'approvals', count(*) FROM approvals
UNION ALL SELECT 'fleet_upgrades', count(*) FROM fleet_upgrades;"
section "duplicate events (must be 0)"
"${psql_cmd[@]}" -c "SELECT count(*) FROM (SELECT transaction_hash, event_index FROM controller_events GROUP BY 1,2 HAVING count(*) > 1) d;"
section "indexer checkpoints (controller | ledger | updated)"
"${psql_cmd[@]}" -c "SELECT controller_id, ledger_sequence, updated_at FROM indexer_checkpoints;"

section "analysis jobs by status"
"${psql_cmd[@]}" -c "SELECT status, count(*) FROM analysis_jobs GROUP BY status ORDER BY status;"
section "reports and manifests"
"${psql_cmd[@]}" -c "SELECT 'reports', count(*) FROM analysis_reports UNION ALL SELECT 'manifests', count(*) FROM release_manifests;"
section "manifest hash mismatches (must be 0)"
mismatches="$("${psql_cmd[@]}" -c "SELECT count(*) FROM release_manifests WHERE sha256 <> encode(sha256(bytes), 'hex');")"
echo "$mismatches"
section "artifact hash/size sanity (rows with size <= 0, must be 0)"
"${psql_cmd[@]}" -c "SELECT count(*) FROM artifacts WHERE size_bytes <= 0;"

section "exposure: public tables readable by anon/authenticated (must be none)"
"${psql_cmd[@]}" -c "SELECT c.relname, r.rolname FROM pg_class c JOIN pg_roles r ON r.rolname IN ('anon','authenticated')
WHERE c.relnamespace='public'::regnamespace AND c.relkind='r' AND has_table_privilege(r.oid, c.oid, 'SELECT') ORDER BY 1,2;"

if [[ "$mismatches" != "0" ]]; then
  echo "FAIL: stored manifest bytes do not match their recorded hash" >&2
  exit 1
fi
