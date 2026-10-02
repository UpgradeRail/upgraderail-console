package store

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	indexerrpc "github.com/UpgradeRail/upgraderail-console/services/indexer/internal/rpc"
)

func TestLiveReadOnlyTestnetControllerProjection(t *testing.T) {
	if os.Getenv("STELLAR_RPC_LIVE") != "1" {
		t.Skip("STELLAR_RPC_LIVE=1 is required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	endpoint := os.Getenv("STELLAR_RPC_URL")
	controller := os.Getenv("UPGRADERAIL_CONTROLLER_ID")
	startLedgerText := os.Getenv("UPGRADERAIL_START_LEDGER")
	if databaseURL == "" || endpoint == "" || controller == "" || startLedgerText == "" {
		t.Fatal("DATABASE_URL, STELLAR_RPC_URL, UPGRADERAIL_CONTROLLER_ID, and UPGRADERAIL_START_LEDGER are required")
	}
	startLedger, err := strconv.ParseUint(startLedgerText, 10, 32)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "testnet-live-" + suffix
	controllerID := "controller-live-" + suffix
	_, err = store.pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, 'Test SDF Network ; September 2015', $2, 28)
	`, networkID, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `
		INSERT INTO controllers (id, network_id, contract_id, governance_epoch, controller_version)
		VALUES ($1, $2, $3, 1, 1)
	`, controllerID, networkID, controller)
	if err != nil {
		t.Fatal(err)
	}

	batch, err := indexerrpc.Client{Endpoint: endpoint, Network: networkID}.FetchControllerEvents(ctx, controller, uint32(startLedger), 20)
	if err != nil {
		t.Fatal(err)
	}
	state, err := store.ApplyControllerBatch(ctx, projection.NewState(), controllerID, batch.Events, batch.Cursor)
	if err != nil {
		t.Fatal(err)
	}

	proposal := state.Proposals[2]
	if proposal.Status != "executed" || proposal.ApprovalCount != 2 {
		t.Fatalf("unexpected proposal 2 projection: %#v", proposal)
	}
	fleet := state.Fleets["8863397c695ebe1dd9f1922afd511aee68999753d13cd61345767487564dc820"]
	if fleet.WASMHash != "67a657dd6c64f4255f5e928782a058aaeff30c14c7ca530d2604c554d30d76c7" {
		t.Fatalf("unexpected fleet projection: %#v", fleet)
	}

	var journaled int
	var cursor string
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM controller_events WHERE controller_id = $1`, controllerID).Scan(&journaled); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT cursor FROM indexer_checkpoints WHERE controller_id = $1`, controllerID).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if journaled == 0 || cursor == "" {
		t.Fatalf("expected journal and checkpoint, got journaled=%d cursor=%q", journaled, cursor)
	}
}
