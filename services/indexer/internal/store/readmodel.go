package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	"github.com/jackc/pgx/v5"
)

// applyReadModel projects one decoded UpgradeController event into the API's
// fleet/proposal/approval/upgrade read-model tables. It runs inside the same
// database transaction as the journal insert and the checkpoint update, so a
// failure here rolls back the whole batch and the checkpoint never advances.
//
// Every statement here is written so that applying the same event twice (a
// duplicate delivery from Stellar RPC, or a replayed batch) leaves the tables
// in the same state as applying it once: inserts use ON CONFLICT DO NOTHING
// on the row's natural unique key, and updates set columns to the value the
// event carries rather than incrementing/accumulating.
func applyReadModel(ctx context.Context, tx pgx.Tx, controllerID string, event projection.Event) error {
	switch event.Type {
	case projection.ProposalCreated:
		var value struct {
			ProposalID      uint64 `json:"proposal_id"`
			Proposer        string `json:"proposer"`
			GovernanceEpoch uint64 `json:"governance_epoch"`
			ExpiresLedger   uint32 `json:"expires_ledger"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		// kind and manifest_hash are intentionally NULL: the proposal_created
		// event does not carry the ProposalKind variant or its manifest hash.
		// The contract only returns that via the get_proposal read method, and
		// the indexer does not yet reconcile against live contract state (see
		// docs/limitations.md). Leaving these NULL avoids fabricating values.
		_, err := tx.Exec(ctx, `
			INSERT INTO proposals
				(id, controller_id, proposal_id, proposer, kind, governance_epoch, created_ledger, expires_ledger,
				 approval_count, status, manifest_hash, created_transaction_hash, created_event_index)
			VALUES ($1, $2, $3, $4, NULL, $5, $6, $7, 0, 'active', NULL, $8, $9)
			ON CONFLICT (controller_id, proposal_id) DO NOTHING
		`, proposalRowID(controllerID, value.ProposalID), controllerID, int64(value.ProposalID), value.Proposer,
			int64(value.GovernanceEpoch), int64(event.Ledger), int64(value.ExpiresLedger), event.TransactionHash, int32(event.Index))
		if err != nil {
			return fmt.Errorf("insert proposal: %w", err)
		}
		return nil

	case projection.ProposalApproved:
		var value struct {
			ProposalID    uint64 `json:"proposal_id"`
			Approver      string `json:"approver"`
			ApprovalCount uint32 `json:"approval_count"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		proposalID := proposalRowID(controllerID, value.ProposalID)
		if err := setApprovalCount(ctx, tx, controllerID, value.ProposalID, value.ApprovalCount); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO approvals (id, proposal_id, approver, approved_ledger, transaction_hash, event_index)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (proposal_id, approver) DO UPDATE
			SET approved_ledger = EXCLUDED.approved_ledger,
			    transaction_hash = EXCLUDED.transaction_hash,
			    event_index = EXCLUDED.event_index,
			    revoked_ledger = NULL,
			    revoked_transaction_hash = NULL
		`, approvalRowID(proposalID, value.Approver), proposalID, value.Approver, int64(event.Ledger), event.TransactionHash, int32(event.Index))
		if err != nil {
			return fmt.Errorf("insert approval: %w", err)
		}
		return nil

	case projection.ApprovalRevoked:
		var value struct {
			ProposalID    uint64 `json:"proposal_id"`
			Approver      string `json:"approver"`
			ApprovalCount uint32 `json:"approval_count"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		if err := setApprovalCount(ctx, tx, controllerID, value.ProposalID, value.ApprovalCount); err != nil {
			return err
		}
		proposalID := proposalRowID(controllerID, value.ProposalID)
		tag, err := tx.Exec(ctx, `
			UPDATE approvals
			SET revoked_ledger = $1, revoked_transaction_hash = $2
			WHERE proposal_id = $3 AND approver = $4
		`, int64(event.Ledger), event.TransactionHash, proposalID, value.Approver)
		if err != nil {
			return fmt.Errorf("revoke approval: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("approval_revoked for proposal %d approver %s has no matching approval row", value.ProposalID, value.Approver)
		}
		return nil

	case projection.ThresholdReached:
		var value struct {
			ProposalID         uint64 `json:"proposal_id"`
			ApprovedLedger     uint32 `json:"approved_ledger"`
			ExecuteAfterLedger uint32 `json:"execute_after_ledger"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		return updateProposalOrFail(ctx, tx, controllerID, value.ProposalID, `
			UPDATE proposals SET approved_ledger = $3, execute_after_ledger = $4, updated_at = now()
			WHERE controller_id = $1 AND proposal_id = $2
		`, int64(value.ApprovedLedger), int64(value.ExecuteAfterLedger))

	case projection.ThresholdReset:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		return updateProposalOrFail(ctx, tx, controllerID, value.ProposalID, `
			UPDATE proposals SET approved_ledger = NULL, execute_after_ledger = NULL, updated_at = now()
			WHERE controller_id = $1 AND proposal_id = $2
		`)

	case projection.ProposalCancelled:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		return updateProposalOrFail(ctx, tx, controllerID, value.ProposalID, `
			UPDATE proposals SET status = 'cancelled', updated_at = now()
			WHERE controller_id = $1 AND proposal_id = $2
		`)

	case projection.ProposalExecuted:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		return updateProposalOrFail(ctx, tx, controllerID, value.ProposalID, `
			UPDATE proposals SET status = 'executed', updated_at = now()
			WHERE controller_id = $1 AND proposal_id = $2
		`)

	case projection.FleetCreated:
		var value struct {
			FleetID  string `json:"fleet_id"`
			Tag      string `json:"tag"`
			WASMHash string `json:"wasm_hash"`
			Ledger   uint32 `json:"ledger"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO fleets (id, controller_id, fleet_hash, tag, current_wasm_hash, created_ledger, created_transaction_hash, created_event_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (controller_id, fleet_hash) DO NOTHING
		`, fleetRowID(controllerID, value.FleetID), controllerID, value.FleetID, value.Tag, value.WASMHash, int64(value.Ledger), event.TransactionHash, int32(event.Index))
		if err != nil {
			return fmt.Errorf("insert fleet: %w", err)
		}
		return nil

	case projection.FleetUpgraded:
		var value struct {
			FleetID      string `json:"fleet_id"`
			ProposalID   uint64 `json:"proposal_id"`
			OldWASMHash  string `json:"old_wasm_hash"`
			NewWASMHash  string `json:"new_wasm_hash"`
			ManifestHash string `json:"manifest_hash"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE fleets SET current_wasm_hash = $1 WHERE controller_id = $2 AND fleet_hash = $3
		`, value.NewWASMHash, controllerID, value.FleetID)
		if err != nil {
			return fmt.Errorf("update fleet: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("fleet_upgraded for fleet %s has no matching fleet row", value.FleetID)
		}
		proposalID := proposalRowID(controllerID, value.ProposalID)
		_, err = tx.Exec(ctx, `
			INSERT INTO fleet_upgrades (id, fleet_id, proposal_id, old_wasm_hash, new_wasm_hash, manifest_hash, ledger_sequence, transaction_hash, event_index)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (transaction_hash, event_index) DO NOTHING
		`, event.Key(), fleetRowID(controllerID, value.FleetID), proposalID, value.OldWASMHash, value.NewWASMHash, value.ManifestHash,
			int64(event.Ledger), event.TransactionHash, int32(event.Index))
		if err != nil {
			return fmt.Errorf("insert fleet upgrade: %w", err)
		}
		return nil

	case projection.PolicyUpdated:
		var value struct {
			GovernanceEpoch uint64 `json:"governance_epoch"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE controllers SET governance_epoch = $1, observed_ledger = $2, updated_at = now() WHERE id = $3
		`, int64(value.GovernanceEpoch), int64(event.Ledger), controllerID)
		if err != nil {
			return fmt.Errorf("update controller policy: %w", err)
		}
		return nil

	case projection.ControllerUpgraded:
		var value struct {
			NewControllerVersion uint32 `json:"new_controller_version"`
			NewWASMHash          string `json:"new_wasm_hash"`
		}
		if err := json.Unmarshal(event.Data, &value); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE controllers SET wasm_hash = $1, controller_version = $2, observed_ledger = $3, updated_at = now() WHERE id = $4
		`, value.NewWASMHash, int32(value.NewControllerVersion), int64(event.Ledger), controllerID)
		if err != nil {
			return fmt.Errorf("update controller upgrade: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported UpgradeController event %q", event.Type)
	}
}

func setApprovalCount(ctx context.Context, tx pgx.Tx, controllerID string, proposalID uint64, count uint32) error {
	return updateProposalOrFail(ctx, tx, controllerID, proposalID, `
		UPDATE proposals SET approval_count = $3, updated_at = now() WHERE controller_id = $1 AND proposal_id = $2
	`, int32(count))
}

func updateProposalOrFail(ctx context.Context, tx pgx.Tx, controllerID string, proposalID uint64, sql string, args ...any) error {
	fullArgs := append([]any{controllerID, int64(proposalID)}, args...)
	tag, err := tx.Exec(ctx, sql, fullArgs...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("proposal %d has no matching proposal row", proposalID)
	}
	return nil
}

func proposalRowID(controllerID string, proposalID uint64) string {
	return controllerID + ":" + strconv.FormatUint(proposalID, 10)
}

func fleetRowID(controllerID, fleetHash string) string {
	return controllerID + ":" + fleetHash
}

func approvalRowID(proposalRowID, approver string) string {
	return proposalRowID + ":" + approver
}
