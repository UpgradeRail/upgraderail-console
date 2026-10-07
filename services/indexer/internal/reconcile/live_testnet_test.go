package reconcile

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	indexerrpc "github.com/UpgradeRail/upgraderail-console/services/indexer/internal/rpc"
	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRunReconcilesAgainstLiveTestnetController indexes the real, deployed
// UpgradeController's real events into a fresh database (exactly as the
// indexer would), then — to reproduce the exact bug this feature fixes —
// clears proposal 2's kind/manifest_hash/reconciled_at back to the state a
// pre-reconciliation indexer would have left it in, and runs the real
// reconciliation pass against the live contract over Soroban RPC
// simulateTransaction (read-only; nothing is signed or submitted) to confirm
// it fills the fields back in from the chain rather than from the event data
// that was just cleared.
func TestRunReconcilesAgainstLiveTestnetController(t *testing.T) {
	if os.Getenv("STELLAR_RPC_LIVE") != "1" {
		t.Skip("STELLAR_RPC_LIVE=1 is required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	endpoint := os.Getenv("STELLAR_RPC_URL")
	controller := os.Getenv("UPGRADERAIL_CONTROLLER_ID")
	passphrase := os.Getenv("STELLAR_NETWORK_PASSPHRASE")
	if databaseURL == "" || endpoint == "" || controller == "" || passphrase == "" {
		t.Fatal("DATABASE_URL, STELLAR_RPC_URL, UPGRADERAIL_CONTROLLER_ID, and STELLAR_NETWORK_PASSPHRASE are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "testnet-reconcile-live-" + suffix
	controllerID := "controller-reconcile-live-" + suffix
	if _, err := pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, $2, $3, 28)
	`, networkID, "Test SDF Network ; September 2015 / "+suffix, endpoint); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO controllers (id, network_id, contract_id, governance_epoch, controller_version)
		VALUES ($1, $2, $3, 1, 1)
	`, controllerID, networkID, controller); err != nil {
		t.Fatal(err)
	}

	client := indexerrpc.Client{Endpoint: endpoint, Network: networkID}
	batch, err := client.FetchControllerEvents(ctx, controller, 4969430, 20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ApplyControllerBatch(ctx, projection.NewState(), controllerID, batch.Events, batch.Cursor); err != nil {
		t.Fatal(err)
	}

	var kindBefore, manifestBefore *string
	if err := pool.QueryRow(ctx, `SELECT kind, manifest_hash FROM proposals WHERE controller_id = $1 AND proposal_id = 2`, controllerID).Scan(&kindBefore, &manifestBefore); err != nil {
		t.Fatal(err)
	}
	if kindBefore == nil || *kindBefore != "UpgradeFleet" || manifestBefore == nil {
		t.Fatalf("expected the event-derived projection to already have kind/manifest_hash, got kind=%v manifest=%v", kindBefore, manifestBefore)
	}

	// Reproduce the pre-reconciliation bug state: proposal_created alone
	// (without the later fleet event's derived fields) leaves these NULL.
	if _, err := pool.Exec(ctx, `
		UPDATE proposals SET kind = NULL, manifest_hash = NULL, reconciled_at = NULL, reconciliation_error = NULL
		WHERE controller_id = $1 AND proposal_id = 2
	`, controllerID); err != nil {
		t.Fatal(err)
	}

	pendingBefore, err := db.ProposalsNeedingReconciliation(ctx, controllerID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingBefore) != 1 || pendingBefore[0].ProposalID != 2 {
		t.Fatalf("expected proposal 2 alone to be pending reconciliation, got %#v", pendingBefore)
	}

	count, err := Run(ctx, db, client, controllerID, controller, passphrase, DefaultBatchSize)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 proposal reconciled against the live controller, got %d", count)
	}

	var kindAfter, manifestAfter *string
	if err := pool.QueryRow(ctx, `SELECT kind, manifest_hash FROM proposals WHERE controller_id = $1 AND proposal_id = 2`, controllerID).Scan(&kindAfter, &manifestAfter); err != nil {
		t.Fatal(err)
	}
	if kindAfter == nil || *kindAfter != "UpgradeFleet" {
		t.Fatalf("expected kind to be reconciled back to UpgradeFleet from the live contract, got %v", kindAfter)
	}
	if manifestAfter == nil || *manifestAfter != *manifestBefore {
		t.Fatalf("expected manifest_hash to be reconciled back to the original event-derived value %v, got %v", manifestBefore, manifestAfter)
	}

	pendingAfter, err := db.ProposalsNeedingReconciliation(ctx, controllerID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingAfter) != 0 {
		t.Fatalf("expected no proposals pending after reconciliation, got %#v", pendingAfter)
	}
}
