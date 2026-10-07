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
			FleetID      string `json:"fleet_id"`
			Tag          string `json:"tag"`
			WASMHash     string `json:"wasm_hash"`
			Ledger       uint32 `json:"ledger"`
			ProposalID   uint64 `json:"proposal_id"`
			ManifestHash string `json:"manifest_hash"`
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
		if value.ProposalID != 0 {
			_, err = tx.Exec(ctx, `
				UPDATE proposals
				SET kind = COALESCE(kind, 'CreateFleet'),
				    manifest_hash = COALESCE(manifest_hash, NULLIF($3, '')),
				    reconciled_at = COALESCE(reconciled_at, now()),
				    reconciliation_error = NULL,
				    updated_at = now()
				WHERE controller_id = $1 AND proposal_id = $2
			`, controllerID, int64(value.ProposalID), value.ManifestHash)
			if err != nil {
				return fmt.Errorf("update proposal for fleet creation: %w", err)
			}
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
		if value.ProposalID != 0 {
			_, err = tx.Exec(ctx, `
				UPDATE proposals
				SET kind = COALESCE(kind, 'UpgradeFleet'),
				    manifest_hash = COALESCE(manifest_hash, NULLIF($3, '')),
				    reconciled_at = COALESCE(reconciled_at, now()),
				    reconciliation_error = NULL,
				    updated_at = now()
				WHERE controller_id = $1 AND proposal_id = $2
			`, controllerID, int64(value.ProposalID), value.ManifestHash)
			if err != nil {
				return fmt.Errorf("update proposal for fleet upgrade: %w", err)
			}
		}
		return nil

	case projection.PolicyUpdated:
		var value struct {
			ProposalID      uint64 `json:"proposal_id"`
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
		if value.ProposalID != 0 {
			_, err = tx.Exec(ctx, `
				UPDATE proposals
				SET kind = COALESCE(kind, 'UpdatePolicy'),
				    reconciled_at = COALESCE(reconciled_at, now()),
				    reconciliation_error = NULL,
				    updated_at = now()
				WHERE controller_id = $1 AND proposal_id = $2
			`, controllerID, int64(value.ProposalID))
			if err != nil {
				return fmt.Errorf("update proposal for policy update: %w", err)
			}
		}
		return nil

	case projection.ControllerUpgraded:
		var value struct {
			ProposalID           uint64 `json:"proposal_id"`
			NewControllerVersion uint32 `json:"new_controller_version"`
			NewWASMHash          string `json:"new_wasm_hash"`
			ManifestHash         string `json:"manifest_hash"`
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
		if value.ProposalID != 0 {
			_, err = tx.Exec(ctx, `
				UPDATE proposals
				SET kind = COALESCE(kind, 'UpgradeController'),
				    manifest_hash = COALESCE(manifest_hash, NULLIF($3, '')),
				    reconciled_at = COALESCE(reconciled_at, now()),
				    reconciliation_error = NULL,
				    updated_at = now()
				WHERE controller_id = $1 AND proposal_id = $2
			`, controllerID, int64(value.ProposalID), value.ManifestHash)
			if err != nil {
				return fmt.Errorf("update proposal for controller upgrade: %w", err)
			}
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

// ReconcileProposal updates an indexed proposal's kind and manifest_hash from a
// verified read-only contract call (such as UpgradeController.get_proposal) or
// authenticated manifest storage linkage, and marks it reconciled so it is not
// re-selected by ProposalsNeedingReconciliation. It is idempotent and performs
// no chain mutation.
//
// kind/manifestHash being nil is "the contract read did not provide this
// field" (e.g. an UpdatePolicy proposal has no manifest_hash); it leaves the
// existing column value untouched rather than clobbering it with NULL, and it
// is not an error — call RecordReconciliationFailure instead when the read
// itself failed.
func (s *Store) ReconcileProposal(ctx context.Context, controllerID string, proposalID uint64, kind *string, manifestHash *string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE proposals
		SET kind = COALESCE($3, kind),
		    manifest_hash = COALESCE($4, manifest_hash),
		    reconciled_at = now(),
		    reconciliation_error = NULL,
		    updated_at = now()
		WHERE controller_id = $1 AND proposal_id = $2
	`, controllerID, int64(proposalID), kind, manifestHash)
	if err != nil {
		return fmt.Errorf("reconcile proposal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("proposal %d not found for reconciliation", proposalID)
	}
	return nil
}

// ProposalToReconcile identifies one proposal row that ProposalsNeedingReconciliation
// selected as not yet successfully reconciled.
type ProposalToReconcile struct {
	ProposalID uint64
}

// ProposalsNeedingReconciliation returns up to limit proposals for controllerID
// that have never been successfully reconciled: either never attempted, or
// their last attempt hit a retryable (transient RPC) failure. A proposal whose
// last attempt hit a non-retryable error is marked reconciled_at by
// RecordReconciliationFailure's permanent flag and so will not reappear here;
// this keeps one bad proposal from being retried forever while still letting
// transient failures retry on a later indexer loop iteration. The limit bounds
// the batch so a large backlog cannot be reconciled in a single pass.
func (s *Store) ProposalsNeedingReconciliation(ctx context.Context, controllerID string, limit int) ([]ProposalToReconcile, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT proposal_id FROM proposals
		WHERE controller_id = $1 AND reconciled_at IS NULL
		ORDER BY proposal_id ASC
		LIMIT $2
	`, controllerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list proposals needing reconciliation: %w", err)
	}
	defer rows.Close()
	var result []ProposalToReconcile
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan proposal needing reconciliation: %w", err)
		}
		result = append(result, ProposalToReconcile{ProposalID: uint64(id)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read proposals needing reconciliation: %w", err)
	}
	return result, nil
}

// RecordReconciliationFailure records a failed reconciliation attempt's error
// message. When permanent is true (the failure was not a transient RPC
// problem — e.g. the contract rejected the call, or the response could not be
// decoded), reconciled_at is also set so the proposal is not retried forever;
// a transient failure leaves reconciled_at NULL so the next indexer loop
// iteration retries it.
func (s *Store) RecordReconciliationFailure(ctx context.Context, controllerID string, proposalID uint64, message string, permanent bool) error {
	var err error
	if permanent {
		_, err = s.pool.Exec(ctx, `
			UPDATE proposals SET reconciliation_error = $3, reconciled_at = now()
			WHERE controller_id = $1 AND proposal_id = $2
		`, controllerID, int64(proposalID), message)
	} else {
		_, err = s.pool.Exec(ctx, `
			UPDATE proposals SET reconciliation_error = $3
			WHERE controller_id = $1 AND proposal_id = $2
		`, controllerID, int64(proposalID), message)
	}
	if err != nil {
		return fmt.Errorf("record reconciliation failure: %w", err)
	}
	return nil
}
