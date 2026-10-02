package projection

import (
	"encoding/json"
	"testing"
)

func event(kind, data string, index uint32) Event {
	return Event{Network: "testnet", Controller: "controller", TransactionHash: "tx", Index: index, Type: kind, Data: json.RawMessage(data)}
}

func TestApplyBatchIsIdempotentAndAdvancesCheckpointAfterSuccess(t *testing.T) {
	state := NewState()
	events := []Event{event(ProposalCreated, `{"proposal_id":2,"proposer":"GABC"}`, 0), event(FleetCreated, `{"fleet_id":"fleet","tag":"shared","wasm_hash":"old","ledger":10}`, 1), event(FleetUpgraded, `{"fleet_id":"fleet","proposal_id":2,"old_wasm_hash":"old","new_wasm_hash":"new","manifest_hash":"manifest"}`, 2)}
	first, err := ApplyBatch(state, events, "cursor-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ApplyBatch(first, events, "cursor-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Upgrades) != 1 || second.Fleets["fleet"].WASMHash != "new" || second.Cursor != "cursor-2" {
		t.Fatalf("unexpected state %#v", second)
	}
}

func TestApplyBatchDoesNotAdvanceOnMalformedEvent(t *testing.T) {
	state := NewState()
	state.Cursor = "old"
	_, err := ApplyBatch(state, []Event{event("invented", `{}`, 0)}, "new")
	if err == nil || state.Cursor != "old" {
		t.Fatal("expected the original checkpoint to remain unchanged")
	}
}
