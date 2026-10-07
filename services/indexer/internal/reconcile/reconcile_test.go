package reconcile

import (
	"context"
	"errors"
	"testing"

	indexerrpc "github.com/UpgradeRail/upgraderail-console/services/indexer/internal/rpc"
	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/store"
)

type fakeStore struct {
	pending    []store.ProposalToReconcile
	reconciled map[uint64]indexerrpc.ProposalKindResult
	failures   map[uint64]struct {
		message   string
		permanent bool
	}
	reconcileErr error
	listErr      error
	recordErr    error
}

func newFakeStore(pending ...uint64) *fakeStore {
	f := &fakeStore{reconciled: map[uint64]indexerrpc.ProposalKindResult{}, failures: map[uint64]struct {
		message   string
		permanent bool
	}{}}
	for _, id := range pending {
		f.pending = append(f.pending, store.ProposalToReconcile{ProposalID: id})
	}
	return f
}

func (f *fakeStore) ProposalsNeedingReconciliation(ctx context.Context, controllerID string, limit int) ([]store.ProposalToReconcile, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if limit < len(f.pending) {
		return f.pending[:limit], nil
	}
	return f.pending, nil
}

func (f *fakeStore) ReconcileProposal(ctx context.Context, controllerID string, proposalID uint64, kind *string, manifestHash *string) error {
	if f.reconcileErr != nil {
		return f.reconcileErr
	}
	result := indexerrpc.ProposalKindResult{}
	if kind != nil {
		result.Kind = *kind
	}
	result.ManifestHash = manifestHash
	f.reconciled[proposalID] = result
	return nil
}

func (f *fakeStore) RecordReconciliationFailure(ctx context.Context, controllerID string, proposalID uint64, message string, permanent bool) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	f.failures[proposalID] = struct {
		message   string
		permanent bool
	}{message, permanent}
	return nil
}

type fakeRPC struct {
	results map[uint64]indexerrpc.ProposalKindResult
	errs    map[uint64]error
}

func (f *fakeRPC) GetProposalKind(ctx context.Context, controllerID string, proposalID uint64, passphrase string) (indexerrpc.ProposalKindResult, error) {
	if err, ok := f.errs[proposalID]; ok {
		return indexerrpc.ProposalKindResult{}, err
	}
	return f.results[proposalID], nil
}

func TestRunReconcilesPendingProposals(t *testing.T) {
	manifest := "manifest-hash-1"
	db := newFakeStore(1, 2)
	rpc := &fakeRPC{results: map[uint64]indexerrpc.ProposalKindResult{
		1: {Kind: "CreateFleet", ManifestHash: &manifest},
		2: {Kind: "UpdatePolicy"}, // no manifest_hash: genuinely unknown
	}}

	count, err := Run(context.Background(), db, rpc, "controller", "passphrase", 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 reconciled, got %d", count)
	}
	if db.reconciled[1].Kind != "CreateFleet" || db.reconciled[1].ManifestHash == nil || *db.reconciled[1].ManifestHash != manifest {
		t.Fatalf("unexpected reconciliation for proposal 1: %#v", db.reconciled[1])
	}
	if db.reconciled[2].Kind != "UpdatePolicy" || db.reconciled[2].ManifestHash != nil {
		t.Fatalf("unexpected reconciliation for proposal 2: %#v", db.reconciled[2])
	}
}

func TestRunRetriesTransientFailureButNotPermanentOne(t *testing.T) {
	db := newFakeStore(1, 2)
	rpc := &fakeRPC{errs: map[uint64]error{
		1: &indexerrpc.TemporaryError{Err: errors.New("rpc unavailable")},
		2: &indexerrpc.ContractReadError{Message: "HostError: Error(Contract, #50)"},
	}}

	count, err := Run(context.Background(), db, rpc, "controller", "passphrase", 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reconciled, got %d", count)
	}
	if db.failures[1].permanent {
		t.Fatal("expected proposal 1's transient RPC failure to be marked retryable")
	}
	if !db.failures[2].permanent {
		t.Fatal("expected proposal 2's contract rejection to be marked non-retryable")
	}
}

func TestRunBoundsBatchSize(t *testing.T) {
	db := newFakeStore(1, 2, 3, 4, 5)
	rpc := &fakeRPC{results: map[uint64]indexerrpc.ProposalKindResult{
		1: {Kind: "CreateFleet"}, 2: {Kind: "CreateFleet"},
	}}

	count, err := Run(context.Background(), db, rpc, "controller", "passphrase", 2)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected batch size to bound reconciliation to 2, got %d", count)
	}
}

func TestRunDoesNotFailTheCallerWhenAnIndividualReadFails(t *testing.T) {
	db := newFakeStore(1)
	rpc := &fakeRPC{errs: map[uint64]error{1: errors.New("boom")}}

	count, err := Run(context.Background(), db, rpc, "controller", "passphrase", 10)
	if err != nil {
		t.Fatalf("Run must not surface a single proposal's read failure as its own error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reconciled, got %d", count)
	}
}
