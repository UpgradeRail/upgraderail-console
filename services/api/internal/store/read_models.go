package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Controller struct {
	ID, NetworkID, ContractID string
	WASMHash                  *string
	GovernanceEpoch           *int64
	ControllerVersion         *int32
}

type Fleet struct {
	ID, ControllerID, FleetHash, Tag, CreatedTransactionHash string
	CurrentWASMHash                                          *string
	CreatedLedger                                            int64
}

type Proposal struct {
	ID, ControllerID, Proposer, Kind, Status, CreatedTransactionHash string
	ProposalID, GovernanceEpoch, CreatedLedger, ExpiresLedger        int64
	ApprovalCount                                                    int32
	ApprovedLedger, ExecuteAfterLedger                               *int64
	ManifestHash                                                     *string
}

type Approval struct {
	ID, ProposalID, Approver, TransactionHash string
	ApprovedLedger                            int64
	RevokedLedger                             *int64
}

type FleetUpgrade struct {
	ID, FleetID, ProposalID, OldWASMHash, NewWASMHash, ManifestHash, TransactionHash string
	LedgerSequence                                                                   int64
}

func (s *Store) ListControllers(ctx context.Context, page Page) ([]Controller, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, network_id, contract_id, wasm_hash, governance_epoch, controller_version FROM controllers ORDER BY updated_at DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list controllers: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Controller])
}

func (s *Store) GetController(ctx context.Context, id string) (Controller, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, network_id, contract_id, wasm_hash, governance_epoch, controller_version FROM controllers WHERE id = $1`, id)
	var value Controller
	err := row.Scan(&value.ID, &value.NetworkID, &value.ContractID, &value.WASMHash, &value.GovernanceEpoch, &value.ControllerVersion)
	if err != nil {
		return Controller{}, fmt.Errorf("get controller: %w", err)
	}
	return value, nil
}

func (s *Store) ListFleets(ctx context.Context, page Page) ([]Fleet, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, controller_id, fleet_hash, tag, current_wasm_hash, created_ledger, created_transaction_hash FROM fleets ORDER BY created_ledger DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list fleets: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Fleet])
}

func (s *Store) GetFleet(ctx context.Context, id string) (Fleet, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, controller_id, fleet_hash, tag, current_wasm_hash, created_ledger, created_transaction_hash FROM fleets WHERE id = $1`, id)
	var value Fleet
	err := row.Scan(&value.ID, &value.ControllerID, &value.FleetHash, &value.Tag, &value.CurrentWASMHash, &value.CreatedLedger, &value.CreatedTransactionHash)
	if err != nil {
		return Fleet{}, fmt.Errorf("get fleet: %w", err)
	}
	return value, nil
}

func (s *Store) ListFleetUpgrades(ctx context.Context, fleetID string, page Page) ([]FleetUpgrade, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, fleet_id, proposal_id, old_wasm_hash, new_wasm_hash, manifest_hash, ledger_sequence, transaction_hash FROM fleet_upgrades WHERE fleet_id = $1 ORDER BY ledger_sequence DESC LIMIT $2 OFFSET $3`, fleetID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list fleet upgrades: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[FleetUpgrade])
}

func (s *Store) ListProposals(ctx context.Context, page Page) ([]Proposal, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, controller_id, proposal_id, proposer, kind, governance_epoch, created_ledger, expires_ledger, approval_count, approved_ledger, execute_after_ledger, status, manifest_hash, created_transaction_hash FROM proposals ORDER BY created_ledger DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Proposal])
}

func (s *Store) GetProposal(ctx context.Context, id string) (Proposal, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, controller_id, proposal_id, proposer, kind, governance_epoch, created_ledger, expires_ledger, approval_count, approved_ledger, execute_after_ledger, status, manifest_hash, created_transaction_hash FROM proposals WHERE id = $1`, id)
	var value Proposal
	err := row.Scan(&value.ID, &value.ControllerID, &value.ProposalID, &value.Proposer, &value.Kind, &value.GovernanceEpoch, &value.CreatedLedger, &value.ExpiresLedger, &value.ApprovalCount, &value.ApprovedLedger, &value.ExecuteAfterLedger, &value.Status, &value.ManifestHash, &value.CreatedTransactionHash)
	if err != nil {
		return Proposal{}, fmt.Errorf("get proposal: %w", err)
	}
	return value, nil
}

func (s *Store) ListApprovals(ctx context.Context, proposalID string, page Page) ([]Approval, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, proposal_id, approver, approved_ledger, transaction_hash, revoked_ledger FROM approvals WHERE proposal_id = $1 ORDER BY approved_ledger DESC LIMIT $2 OFFSET $3`, proposalID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Approval])
}
