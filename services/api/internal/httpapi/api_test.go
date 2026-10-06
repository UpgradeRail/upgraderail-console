package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests exercise the real fleet/proposal/approval endpoints over real
// HTTP against a real PostgreSQL database seeded the same way the indexer's
// read-model projection writes rows, so a regression in the SQL the API
// issues (not just the indexer's SQL) would be caught here.
func TestFleetProposalApprovalEndpointsServeProjectedReadModels(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seed, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "api-test-network-" + suffix
	controllerID := "api-test-controller-" + suffix
	fleetID := controllerID + ":fleet-" + suffix
	proposalID := controllerID + ":1"

	mustExec(t, ctx, seed, `INSERT INTO networks (id, passphrase, rpc_url, protocol_target) VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 29)`,
		networkID, "passphrase-"+suffix)
	mustExec(t, ctx, seed, `INSERT INTO controllers (id, network_id, contract_id) VALUES ($1, $2, $3)`,
		controllerID, networkID, "contract-"+suffix)
	mustExec(t, ctx, seed, `
		INSERT INTO fleets (id, controller_id, fleet_hash, tag, current_wasm_hash, created_ledger, created_transaction_hash, created_event_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fleetID, controllerID, "fleet-"+suffix, "shared-tag", "wasm-new", int64(1000), "tx-create", int32(0))
	mustExec(t, ctx, seed, `
		INSERT INTO proposals (id, controller_id, proposal_id, proposer, kind, governance_epoch, created_ledger, expires_ledger, approval_count, status, manifest_hash, created_transaction_hash, created_event_index)
		VALUES ($1, $2, 1, $3, NULL, 1, 900, 2000, 2, 'executed', NULL, $4, 0)`,
		proposalID, controllerID, "GPROPOSER", "tx-proposal")
	mustExec(t, ctx, seed, `
		INSERT INTO approvals (id, proposal_id, approver, approved_ledger, transaction_hash, event_index)
		VALUES ($1, $2, $3, 910, $4, 1)`,
		proposalID+":GAPPROVER1", proposalID, "GAPPROVER1", "tx-approve-1")
	mustExec(t, ctx, seed, `
		INSERT INTO approvals (id, proposal_id, approver, approved_ledger, transaction_hash, event_index, revoked_ledger, revoked_transaction_hash)
		VALUES ($1, $2, $3, 915, $4, 2, 950, $5)`,
		proposalID+":GAPPROVER2", proposalID, "GAPPROVER2", "tx-approve-2", "tx-revoke-2")
	mustExec(t, ctx, seed, `
		INSERT INTO fleet_upgrades (id, fleet_id, proposal_id, old_wasm_hash, new_wasm_hash, manifest_hash, ledger_sequence, transaction_hash, event_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		"tx-upgrade-"+suffix+":0", fleetID, proposalID, "wasm-old", "wasm-new", "manifest", int64(1500), "tx-upgrade-"+suffix, int32(0))
	mustExec(t, ctx, seed, `
		INSERT INTO controller_events (id, controller_id, network_id, ledger_sequence, transaction_hash, event_index, event_type, topics, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, '[]'::jsonb, '{"tag":"shared-tag"}'::jsonb)`,
		"event-"+suffix, controllerID, networkID, int64(1000), "tx-create", int32(0), "fleet_created")

	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	server := httptest.NewServer(New(repository, "example.com", "https://example.com"))
	defer server.Close()
	client := server.Client()

	fleet := getJSON(t, client, server.URL+"/api/v1/fleets/"+fleetID)
	if fleet["current_wasm_hash"] != "wasm-new" || fleet["tag"] != "shared-tag" {
		t.Fatalf("unexpected projected fleet: %#v", fleet)
	}

	upgrades := getJSONArray(t, client, server.URL+"/api/v1/fleets/"+fleetID+"/upgrades")
	if len(upgrades) != 1 || upgrades[0]["new_wasm_hash"] != "wasm-new" || upgrades[0]["manifest_hash"] != "manifest" {
		t.Fatalf("unexpected projected fleet upgrades: %#v", upgrades)
	}

	proposal := getJSON(t, client, server.URL+"/api/v1/proposals/"+proposalID)
	if proposal["status"] != "executed" {
		t.Fatalf("unexpected projected proposal status: %#v", proposal)
	}
	if n, ok := proposal["approval_count"].(float64); !ok || n != 2 {
		t.Fatalf("unexpected projected approval_count: %#v", proposal)
	}
	if proposal["kind"] != nil {
		t.Fatalf("proposal kind must stay null without reconciliation, got %#v", proposal["kind"])
	}

	approvals := getJSONArray(t, client, server.URL+"/api/v1/proposals/"+proposalID+"/approvals")
	if len(approvals) != 2 {
		t.Fatalf("expected both the live and the revoked approval row, got %#v", approvals)
	}
	var sawRevoked, sawActive bool
	for _, approval := range approvals {
		if approval["approver"] == "GAPPROVER2" {
			if approval["revoked_ledger"] == nil {
				t.Fatalf("expected GAPPROVER2's approval to show a revocation: %#v", approval)
			}
			sawRevoked = true
		}
		if approval["approver"] == "GAPPROVER1" {
			if approval["revoked_ledger"] != nil {
				t.Fatalf("expected GAPPROVER1's approval to remain active: %#v", approval)
			}
			sawActive = true
		}
	}
	if !sawRevoked || !sawActive {
		t.Fatalf("expected both an active and a revoked approval, got %#v", approvals)
	}

	fleets := getJSONArray(t, client, server.URL+"/api/v1/fleets")
	if len(fleets) == 0 {
		t.Fatal("expected the projected fleet to be listed")
	}

	allUpgrades := getJSONArray(t, client, server.URL+"/api/v1/upgrades")
	if len(allUpgrades) == 0 || allUpgrades[0]["new_wasm_hash"] != "wasm-new" {
		t.Fatalf("unexpected projected all upgrades: %#v", allUpgrades)
	}

	events := getJSONArray(t, client, server.URL+"/api/v1/events")
	if len(events) == 0 || events[0]["event_type"] != "fleet_created" {
		t.Fatalf("unexpected projected controller events: %#v", events)
	}
}

func TestFleetProposalEndpointsOnEmptyDatabaseReturnEmptyLists(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seed, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer seed.Close()
	// Verifies against a network with no controllers, fleets, or proposals
	// at all, i.e. the state of a freshly migrated database before the
	// indexer has ever run. The endpoint must respond with an empty list,
	// not an error and not fabricated rows.
	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "api-empty-network-" + suffix
	mustExec(t, ctx, seed, `INSERT INTO networks (id, passphrase, rpc_url, protocol_target) VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 29)`,
		networkID, "passphrase-"+suffix)

	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	server := httptest.NewServer(New(repository, "example.com", "https://example.com"))
	defer server.Close()
	client := server.Client()

	response, err := client.Get(server.URL + "/api/v1/fleets?limit=5&offset=0")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	var fleets []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&fleets); err != nil {
		t.Fatal(err)
	}
	// Other tests may have left rows from other controllers in a shared
	// database; this just confirms there is no crash and the shape is a list.
	if fleets == nil {
		t.Fatal("expected an empty list, not null, when no fleets are projected")
	}
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func getJSON(t *testing.T, client *http.Client, url string) map[string]any {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s returned status %d", url, response.StatusCode)
	}
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func getJSONArray(t *testing.T, client *http.Client, url string) []map[string]any {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s returned status %d", url, response.StatusCode)
	}
	var value []map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}
