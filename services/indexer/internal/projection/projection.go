// Package projection applies decoded UpgradeController events deterministically.
package projection

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ProposalCreated    = "proposal_created"
	ProposalApproved   = "proposal_approved"
	ApprovalRevoked    = "approval_revoked"
	ThresholdReached   = "threshold_reached"
	ThresholdReset     = "threshold_reset"
	ProposalCancelled  = "proposal_cancelled"
	ProposalExecuted   = "proposal_executed"
	FleetCreated       = "fleet_created"
	FleetUpgraded      = "fleet_upgraded"
	PolicyUpdated      = "policy_updated"
	ControllerUpgraded = "controller_upgraded"
)

type Event struct {
	Network, Controller, TransactionHash string
	RPCEventID                           string
	Ledger                               uint32
	Index                                uint32
	Type                                 string
	Data                                 json.RawMessage
}

func (e Event) Key() string { return fmt.Sprintf("%s:%s:%d", e.Network, e.TransactionHash, e.Index) }

type Fleet struct {
	ID, Tag, WASMHash string
	CreatedLedger     uint32
}
type Proposal struct {
	ID                                 uint64
	Proposer, Status                   string
	ApprovalCount                      uint32
	ApprovedLedger, ExecuteAfterLedger *uint32
}
type Upgrade struct {
	FleetID                                string
	ProposalID                             uint64
	OldWASMHash, NewWASMHash, ManifestHash string
}

type State struct {
	Cursor    string
	Seen      map[string]struct{}
	Fleets    map[string]Fleet
	Proposals map[uint64]Proposal
	Upgrades  []Upgrade
}

func NewState() State {
	return State{Seen: map[string]struct{}{}, Fleets: map[string]Fleet{}, Proposals: map[uint64]Proposal{}}
}

// ApplyBatch returns a new state. A malformed event never partially advances a checkpoint.
func ApplyBatch(previous State, events []Event, nextCursor string) (State, error) {
	next := copyState(previous)
	for _, event := range events {
		if _, exists := next.Seen[event.Key()]; exists {
			continue
		}
		if err := apply(&next, event); err != nil {
			return previous, err
		}
		next.Seen[event.Key()] = struct{}{}
	}
	next.Cursor = nextCursor
	return next, nil
}

func apply(state *State, event Event) error {
	switch event.Type {
	case ProposalCreated:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
			Proposer   string `json:"proposer"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		state.Proposals[value.ProposalID] = Proposal{ID: value.ProposalID, Proposer: value.Proposer, Status: "active"}
	case ProposalApproved, ApprovalRevoked:
		var value struct {
			ProposalID    uint64 `json:"proposal_id"`
			ApprovalCount uint32 `json:"approval_count"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		proposal, ok := state.Proposals[value.ProposalID]
		if !ok {
			return fmt.Errorf("proposal %d is missing", value.ProposalID)
		}
		proposal.ApprovalCount = value.ApprovalCount
		state.Proposals[value.ProposalID] = proposal
	case ThresholdReached:
		var value struct {
			ProposalID         uint64 `json:"proposal_id"`
			ApprovedLedger     uint32 `json:"approved_ledger"`
			ExecuteAfterLedger uint32 `json:"execute_after_ledger"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		proposal, ok := state.Proposals[value.ProposalID]
		if !ok {
			return fmt.Errorf("proposal %d is missing", value.ProposalID)
		}
		proposal.ApprovedLedger = &value.ApprovedLedger
		proposal.ExecuteAfterLedger = &value.ExecuteAfterLedger
		state.Proposals[value.ProposalID] = proposal
	case ThresholdReset:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		proposal, ok := state.Proposals[value.ProposalID]
		if !ok {
			return fmt.Errorf("proposal %d is missing", value.ProposalID)
		}
		proposal.ApprovedLedger = nil
		proposal.ExecuteAfterLedger = nil
		state.Proposals[value.ProposalID] = proposal
	case ProposalCancelled, ProposalExecuted:
		var value struct {
			ProposalID uint64 `json:"proposal_id"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		proposal, ok := state.Proposals[value.ProposalID]
		if !ok {
			return fmt.Errorf("proposal %d is missing", value.ProposalID)
		}
		if event.Type == ProposalCancelled {
			proposal.Status = "cancelled"
		} else {
			proposal.Status = "executed"
		}
		state.Proposals[value.ProposalID] = proposal
	case FleetCreated:
		var value struct {
			FleetID  string `json:"fleet_id"`
			Tag      string `json:"tag"`
			WASMHash string `json:"wasm_hash"`
			Ledger   uint32 `json:"ledger"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		state.Fleets[value.FleetID] = Fleet{ID: value.FleetID, Tag: value.Tag, WASMHash: value.WASMHash, CreatedLedger: value.Ledger}
	case FleetUpgraded:
		var value struct {
			FleetID      string `json:"fleet_id"`
			ProposalID   uint64 `json:"proposal_id"`
			OldWASMHash  string `json:"old_wasm_hash"`
			NewWASMHash  string `json:"new_wasm_hash"`
			ManifestHash string `json:"manifest_hash"`
		}
		if err := decode(event.Data, &value); err != nil {
			return err
		}
		fleet, ok := state.Fleets[value.FleetID]
		if !ok {
			return fmt.Errorf("fleet %s is missing", value.FleetID)
		}
		fleet.WASMHash = value.NewWASMHash
		state.Fleets[value.FleetID] = fleet
		state.Upgrades = append(state.Upgrades, Upgrade{FleetID: value.FleetID, ProposalID: value.ProposalID, OldWASMHash: value.OldWASMHash, NewWASMHash: value.NewWASMHash, ManifestHash: value.ManifestHash})
	case PolicyUpdated, ControllerUpgraded:
		return nil
	default:
		return fmt.Errorf("unsupported UpgradeController event %q", event.Type)
	}
	return nil
}

func decode(data json.RawMessage, output any) error {
	if len(data) == 0 {
		return errors.New("event data is required")
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode event data: %w", err)
	}
	return nil
}
func copyState(value State) State {
	next := NewState()
	next.Cursor = value.Cursor
	for key := range value.Seen {
		next.Seen[key] = struct{}{}
	}
	for key, fleet := range value.Fleets {
		next.Fleets[key] = fleet
	}
	for key, proposal := range value.Proposals {
		next.Proposals[key] = proposal
	}
	next.Upgrades = append(next.Upgrades, value.Upgrades...)
	return next
}
