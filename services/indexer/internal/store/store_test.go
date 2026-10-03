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
