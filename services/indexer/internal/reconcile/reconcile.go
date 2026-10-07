// Package reconcile fills in a proposal's kind and manifest_hash from the
// live UpgradeController contract when the indexed event stream alone
// couldn't provide them (proposal_created does not carry either field).
// Reconciliation is a read-only side process: it never signs or submits a
// transaction, it never advances the event-journal checkpoint, and a
// reconciliation failure never blocks the main indexing loop — it is retried,
// bounded, on a later call.
package reconcile

import (
	"context"
	"errors"
	"log/slog"

	indexerrpc "github.com/UpgradeRail/upgraderail-console/services/indexer/internal/rpc"
	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/store"
)

// ProposalReader is the subset of indexerrpc.Client that reconciliation
// needs; it is an interface so tests can substitute a fake without a
// network-backed Soroban RPC client.
type ProposalReader interface {
	GetProposalKind(ctx context.Context, controllerID string, proposalID uint64, passphrase string) (indexerrpc.ProposalKindResult, error)
}

// ProposalStore is the subset of *store.Store that reconciliation needs.
type ProposalStore interface {
	ProposalsNeedingReconciliation(ctx context.Context, controllerID string, limit int) ([]store.ProposalToReconcile, error)
	ReconcileProposal(ctx context.Context, controllerID string, proposalID uint64, kind *string, manifestHash *string) error
	RecordReconciliationFailure(ctx context.Context, controllerID string, proposalID uint64, message string, permanent bool) error
}

// DefaultBatchSize bounds how many proposals a single reconciliation pass
// will attempt, so a large backlog is worked down over several indexer loop
// iterations rather than in one unbounded pass.
const DefaultBatchSize = 25

// Run reconciles up to batchSize proposals that are missing kind/manifest_hash
// (or whose last attempt hit a retryable failure) for the controller row
// identified by controllerID (the proposals table's internal foreign key, as
// used elsewhere in this package's DB calls). contractID is the actual
// Stellar contract address (a "C..." strkey) that the live RPC call targets
// — these two identifiers are not interchangeable, and conflating them fails
// every RPC read without touching the database at all. Run never returns an
// error for an individual proposal's read failing — that proposal is
// recorded and skipped — so a temporarily unreachable RPC endpoint cannot
// stall the caller's main loop. It returns the count of proposals it
// successfully reconciled.
func Run(ctx context.Context, db ProposalStore, rpc ProposalReader, controllerID, contractID, passphrase string, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	pending, err := db.ProposalsNeedingReconciliation(ctx, controllerID, batchSize)
	if err != nil {
		return 0, err
	}
	reconciled := 0
	for _, proposal := range pending {
		if err := ctx.Err(); err != nil {
			return reconciled, nil
		}
		result, err := rpc.GetProposalKind(ctx, contractID, proposal.ProposalID, passphrase)
		if err != nil {
			var temporary *indexerrpc.TemporaryError
			permanent := !errors.As(err, &temporary)
			if recordErr := db.RecordReconciliationFailure(ctx, controllerID, proposal.ProposalID, err.Error(), permanent); recordErr != nil {
				slog.Warn("reconciliation failed and recording the failure also failed", "service", "indexer", "proposal_id", proposal.ProposalID, "error", recordErr.Error())
				continue
			}
			if permanent {
				slog.Warn("proposal reconciliation failed permanently; leaving fields unreconciled", "service", "indexer", "proposal_id", proposal.ProposalID, "error", err.Error())
			} else {
				slog.Warn("proposal reconciliation failed transiently; will retry", "service", "indexer", "proposal_id", proposal.ProposalID, "error", err.Error())
			}
			continue
		}
		if err := db.ReconcileProposal(ctx, controllerID, proposal.ProposalID, &result.Kind, result.ManifestHash); err != nil {
			slog.Warn("reconciliation read succeeded but writing it failed", "service", "indexer", "proposal_id", proposal.ProposalID, "error", err.Error())
			continue
		}
		reconciled++
	}
	return reconciled, nil
}
