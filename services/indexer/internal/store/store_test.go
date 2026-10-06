package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
)

func TestApplyControllerBatchIsIdempotentAndAdvancesCheckpoint(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	networkID, controllerID := seedController(t, ctx, store, "store-ok")
	events := []projection.Event{
		event(networkID, projection.ProposalCreated, `{"proposal_id":2,"proposer":"GABC"}`, 0),
		event(networkID, projection.FleetCreated, `{"fleet_id":"fleet","tag":"shared","wasm_hash":"old","ledger":10}`, 1),
		event(networkID, projection.FleetUpgraded, `{"fleet_id":"fleet","proposal_id":2,"old_wasm_hash":"old","new_wasm_hash":"new","manifest_hash":"manifest"}`, 2),
	}
	state, err := store.ApplyControllerBatch(ctx, projection.NewState(), controllerID, events, "cursor-1")
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.ApplyControllerBatch(ctx, state, controllerID, events, "cursor-2")
	if err != nil {
		t.Fatal(err)
	}
	if state.Cursor != "cursor-2" || len(state.Upgrades) != 1 {
		t.Fatalf("unexpected state %#v", state)
	}
	var eventCount int
	var cursor string
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM controller_events WHERE controller_id = $1`, controllerID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT cursor FROM indexer_checkpoints WHERE controller_id = $1`, controllerID).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if eventCount != 3 || cursor != "cursor-2" {
		t.Fatalf("unexpected journal count %d cursor %q", eventCount, cursor)
	}
	reloaded, err := store.LoadControllerState(ctx, controllerID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Cursor != "cursor-2" || reloaded.Proposals[2].Status != "active" || reloaded.Fleets["fleet"].WASMHash != "new" || len(reloaded.Upgrades) != 1 {
		t.Fatalf("restart did not replay journal and checkpoint: %#v", reloaded)
	}
}

func TestApplyControllerBatchDoesNotAdvanceCheckpointOnProjectionFailure(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	networkID, controllerID := seedController(t, ctx, store, "store-failure")
	_, err = store.ApplyControllerBatch(ctx, projection.NewState(), controllerID, []projection.Event{
		event(networkID, "invented", `{}`, 0),
	}, "new")
	if err == nil {
		t.Fatal("expected projection failure")
	}
	var count int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM indexer_checkpoints WHERE controller_id = $1`, controllerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("checkpoint advanced after failed batch")
	}
}

func TestApplyControllerBatchProjectsFleetAndProposalReadModels(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	networkID, controllerID := seedController(t, ctx, store, "readmodel")
	tx := "tx-" + controllerID

	events := []projection.Event{
		eventWithTx(networkID, tx, projection.ProposalCreated, `{"proposal_id":7,"proposer":"GPROPOSER","governance_epoch":1,"expires_ledger":5000000}`, 0),
		eventWithTx(networkID, tx, projection.ProposalApproved, `{"proposal_id":7,"approver":"GAPPROVER1","approval_count":1}`, 1),
		eventWithTx(networkID, tx, projection.ProposalApproved, `{"proposal_id":7,"approver":"GAPPROVER2","approval_count":2}`, 2),
		eventWithTx(networkID, tx, projection.ThresholdReached, `{"proposal_id":7,"approved_ledger":100,"execute_after_ledger":112}`, 3),
		eventWithTx(networkID, tx, projection.ApprovalRevoked, `{"proposal_id":7,"approver":"GAPPROVER2","approval_count":1}`, 4),
		eventWithTx(networkID, tx, projection.ThresholdReset, `{"proposal_id":7}`, 5),
		eventWithTx(networkID, tx, projection.FleetCreated, `{"fleet_id":"fleetA","tag":"shared","wasm_hash":"wasmA","ledger":10}`, 6),
		eventWithTx(networkID, tx, projection.FleetUpgraded, `{"fleet_id":"fleetA","proposal_id":7,"old_wasm_hash":"wasmA","new_wasm_hash":"wasmB","manifest_hash":"manifest"}`, 7),
		eventWithTx(networkID, tx, projection.ProposalExecuted, `{"proposal_id":7}`, 8),
	}
	if _, err := store.ApplyControllerBatch(ctx, projection.NewState(), controllerID, events, "cursor-1"); err != nil {
		t.Fatal(err)
	}

	var status string
	var approvalCount int32
	var approvedLedger *int64
	var kind, manifestHash *string
	if err := store.pool.QueryRow(ctx, `SELECT status, approval_count, approved_ledger, kind, manifest_hash FROM proposals WHERE controller_id = $1 AND proposal_id = 7`, controllerID).
		Scan(&status, &approvalCount, &approvedLedger, &kind, &manifestHash); err != nil {
		t.Fatal(err)
	}
	if status != "executed" || approvalCount != 1 || approvedLedger != nil {
		t.Fatalf("unexpected proposal projection: status=%s count=%d approvedLedger=%v", status, approvalCount, approvedLedger)
	}
	if kind != nil || manifestHash != nil {
		t.Fatalf("proposal kind/manifest_hash must stay NULL without reconciliation, got kind=%v manifest_hash=%v", kind, manifestHash)
	}

	var approverCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM approvals WHERE proposal_id = $1`, proposalRowID(controllerID, 7)).Scan(&approverCount); err != nil {
		t.Fatal(err)
	}
	if approverCount != 2 {
		t.Fatalf("expected one approval row per approver even after revocation, got %d", approverCount)
	}
	var revokedLedger *int64
	if err := store.pool.QueryRow(ctx, `SELECT revoked_ledger FROM approvals WHERE proposal_id = $1 AND approver = 'GAPPROVER2'`, proposalRowID(controllerID, 7)).Scan(&revokedLedger); err != nil {
		t.Fatal(err)
	}
	if revokedLedger == nil {
		t.Fatal("expected revoked approval to record a revoked ledger")
	}

	var currentWASMHash string
	if err := store.pool.QueryRow(ctx, `SELECT current_wasm_hash FROM fleets WHERE controller_id = $1 AND fleet_hash = 'fleetA'`, controllerID).Scan(&currentWASMHash); err != nil {
		t.Fatal(err)
	}
	if currentWASMHash != "wasmB" {
		t.Fatalf("expected fleet to reflect the upgraded wasm hash, got %s", currentWASMHash)
	}

	var upgradeCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM fleet_upgrades WHERE fleet_id = $1`, fleetRowID(controllerID, "fleetA")).Scan(&upgradeCount); err != nil {
		t.Fatal(err)
	}
	if upgradeCount != 1 {
		t.Fatalf("expected exactly one recorded fleet upgrade, got %d", upgradeCount)
	}
}

func TestApplyControllerBatchAppliesControllerEventsIdempotently(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	networkID, controllerID := seedController(t, ctx, store, "idempotent")

	events := []projection.Event{
		event(networkID, projection.ProposalCreated, `{"proposal_id":9,"proposer":"GPROPOSER","governance_epoch":1,"expires_ledger":5000000}`, 0),
		event(networkID, projection.ProposalApproved, `{"proposal_id":9,"approver":"GAPPROVER1","approval_count":1}`, 1),
	}
	state, err := store.ApplyControllerBatch(ctx, projection.NewState(), controllerID, events, "cursor-1")
	if err != nil {
		t.Fatal(err)
	}
	// Re-deliver the exact same batch, as a duplicate RPC page or restart replay would.
	if _, err := store.ApplyControllerBatch(ctx, state, controllerID, events, "cursor-1-again"); err != nil {
		t.Fatal(err)
	}

	var proposalRows, approvalRows int
	var approvalCount int32
	if err := store.pool.QueryRow(ctx, `SELECT count(*), max(approval_count) FROM proposals WHERE controller_id = $1 AND proposal_id = 9`, controllerID).Scan(&proposalRows, &approvalCount); err != nil {
		t.Fatal(err)
	}
	if proposalRows != 1 || approvalCount != 1 {
		t.Fatalf("duplicate delivery must not double-count: rows=%d approval_count=%d", proposalRows, approvalCount)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM approvals WHERE proposal_id = $1`, proposalRowID(controllerID, 9)).Scan(&approvalRows); err != nil {
		t.Fatal(err)
	}
	if approvalRows != 1 {
		t.Fatalf("duplicate delivery must not create a second approval row, got %d", approvalRows)
	}
}

func TestApplyControllerBatchDoesNotAdvanceCheckpointOnReadModelFailure(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	networkID, controllerID := seedController(t, ctx, store, "readmodel-failure")

	// The in-memory projection state already believes fleet "ghost" exists
	// (as it would after a process restart that replayed an older journal),
	// but no row for it was ever written to this fresh database. The read
	// model update must fail, and the checkpoint must not advance.
	state := projection.NewState()
	state.Fleets["ghost"] = projection.Fleet{ID: "ghost", WASMHash: "old"}

	_, err = store.ApplyControllerBatch(ctx, state, controllerID, []projection.Event{
		eventWithTx(networkID, "tx-"+controllerID, projection.FleetUpgraded, `{"fleet_id":"ghost","proposal_id":1,"old_wasm_hash":"old","new_wasm_hash":"new","manifest_hash":"manifest"}`, 0),
	}, "cursor-should-not-apply")
	if err == nil {
		t.Fatal("expected the read-model update to fail for a fleet that was never persisted")
	}
	var count int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM indexer_checkpoints WHERE controller_id = $1`, controllerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("checkpoint advanced after a failed read-model projection")
	}
}

func seedController(t *testing.T, ctx context.Context, store *Store, prefix string) (string, string) {
	t.Helper()
	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := prefix + "-network-" + suffix
	controllerID := prefix + "-controller-" + suffix
	_, err := store.pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 28)
	`, networkID, prefix+"-passphrase-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `
		INSERT INTO controllers (id, network_id, contract_id)
		VALUES ($1, $2, $3)
	`, controllerID, networkID, "controller")
	if err != nil {
		t.Fatal(err)
	}
	return networkID, controllerID
}

func event(network, kind, data string, index uint32) projection.Event {
	return projection.Event{Network: network, Controller: "controller", TransactionHash: "tx", RPCEventID: fmt.Sprintf("event-%010d", index), Index: index, Type: kind, Data: json.RawMessage(data)}
}

// eventWithTx is used by tests that write to fleet_upgrades, whose
// (transaction_hash, event_index) uniqueness is global rather than scoped to
// one controller; it derives a transaction hash from the caller-supplied
// prefix so repeated test runs against a persistent database never collide
// with a previous run's rows.
func eventWithTx(network, txHash, kind, data string, index uint32) projection.Event {
	return projection.Event{Network: network, Controller: "controller", TransactionHash: txHash, RPCEventID: fmt.Sprintf("%s-%010d", txHash, index), Index: index, Type: kind, Data: json.RawMessage(data)}
}
