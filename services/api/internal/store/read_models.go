package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Controller struct {
	ID                string  `json:"id"`
	NetworkID         string  `json:"network_id"`
	ContractID        string  `json:"contract_id"`
	WASMHash          *string `json:"wasm_hash"`
	GovernanceEpoch   *int64  `json:"governance_epoch"`
	ControllerVersion *int32  `json:"controller_version"`
}

type Fleet struct {
	ID                     string  `json:"id"`
	ControllerID           string  `json:"controller_id"`
	FleetHash              string  `json:"fleet_hash"`
	Tag                    string  `json:"tag"`
	CurrentWASMHash        *string `json:"current_wasm_hash"`
	CreatedLedger          int64   `json:"created_ledger"`
	CreatedTransactionHash string  `json:"created_transaction_hash"`
}

// Kind and ManifestHash are nullable: the proposal_created event does not
// carry the ProposalKind variant or its manifest hash, so until the indexer
// adds read-only contract reconciliation these stay null rather than being
// fabricated (see docs/limitations.md).
type Proposal struct {
	ID                     string  `json:"id"`
	ControllerID           string  `json:"controller_id"`
	ProposalID             int64   `json:"proposal_id"`
	Proposer               string  `json:"proposer"`
	Kind                   *string `json:"kind"`
	GovernanceEpoch        int64   `json:"governance_epoch"`
	CreatedLedger          int64   `json:"created_ledger"`
	ExpiresLedger          int64   `json:"expires_ledger"`
	ApprovalCount          int32   `json:"approval_count"`
	ApprovedLedger         *int64  `json:"approved_ledger"`
	ExecuteAfterLedger     *int64  `json:"execute_after_ledger"`
	Status                 string  `json:"status"`
	ManifestHash           *string `json:"manifest_hash"`
	CreatedTransactionHash string  `json:"created_transaction_hash"`
}

type Approval struct {
	ID              string `json:"id"`
	ProposalID      string `json:"proposal_id"`
	Approver        string `json:"approver"`
	ApprovedLedger  int64  `json:"approved_ledger"`
	TransactionHash string `json:"transaction_hash"`
	RevokedLedger   *int64 `json:"revoked_ledger"`
}

type FleetUpgrade struct {
	ID              string `json:"id"`
	FleetID         string `json:"fleet_id"`
	ProposalID      string `json:"proposal_id"`
	OldWASMHash     string `json:"old_wasm_hash"`
	NewWASMHash     string `json:"new_wasm_hash"`
	ManifestHash    string `json:"manifest_hash"`
	LedgerSequence  int64  `json:"ledger_sequence"`
	TransactionHash string `json:"transaction_hash"`
}

type ControllerEvent struct {
	ID              string          `json:"id"`
	ControllerID    string          `json:"controller_id"`
	NetworkID       string          `json:"network_id"`
	LedgerSequence  int64           `json:"ledger_sequence"`
	TransactionHash string          `json:"transaction_hash"`
	EventIndex      int32           `json:"event_index"`
	EventType       string          `json:"event_type"`
	Topics          json.RawMessage `json:"topics"`
	Data            json.RawMessage `json:"data"`
	ObservedAt      time.Time       `json:"observed_at"`
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

func (s *Store) ListAllFleetUpgrades(ctx context.Context, page Page) ([]FleetUpgrade, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, fleet_id, proposal_id, old_wasm_hash, new_wasm_hash, manifest_hash, ledger_sequence, transaction_hash FROM fleet_upgrades ORDER BY ledger_sequence DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list all fleet upgrades: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[FleetUpgrade])
}

func (s *Store) ListEvents(ctx context.Context, page Page) ([]ControllerEvent, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, controller_id, network_id, ledger_sequence, transaction_hash, event_index, event_type, topics, data, observed_at FROM controller_events ORDER BY ledger_sequence DESC, event_index DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list controller events: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[ControllerEvent])
}
