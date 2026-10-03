// Package rpc fetches UpgradeController events from Stellar RPC.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
)

type Client struct {
	Endpoint   string
	Network    string
	HTTPClient *http.Client
}

type Batch struct {
	Events []projection.Event
	Cursor string
}

func (c Client) FetchControllerEvents(ctx context.Context, controller string, startLedger uint32, limit uint32) (Batch, error) {
	if startLedger == 0 {
		return Batch{}, errors.New("start ledger is required")
	}
	return c.fetchControllerEvents(ctx, controller, startLedger, "", limit)
}

func (c Client) FetchControllerEventsAfter(ctx context.Context, controller, cursor string, limit uint32) (Batch, error) {
	if cursor == "" {
		return Batch{}, errors.New("checkpoint cursor is required")
	}
	return c.fetchControllerEvents(ctx, controller, 0, cursor, limit)
}

func (c Client) fetchControllerEvents(ctx context.Context, controller string, startLedger uint32, cursor string, limit uint32) (Batch, error) {
	if c.Endpoint == "" {
		return Batch{}, errors.New("stellar rpc endpoint is required")
	}
	if c.Network == "" {
		return Batch{}, errors.New("network is required")
	}
	if controller == "" {
		return Batch{}, errors.New("controller id is required")
	}
	if limit == 0 {
		limit = 200
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	requestBody := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getEvents",
		Params: getEventsParams{
			StartLedger: startLedger,
			Pagination:  pagination{Limit: limit, Cursor: cursor},
			Filters: []eventFilter{{
				Type:        "contract",
				ContractIDs: []string{controller},
			}},
			XDRFormat: "json",
		},
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return Batch{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return Batch{}, err
	}
	request.Header.Set("content-type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return Batch{}, fmt.Errorf("stellar rpc getEvents: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return Batch{}, fmt.Errorf("stellar rpc getEvents returned HTTP %d", response.StatusCode)
	}

	var decoded rpcResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return Batch{}, fmt.Errorf("decode stellar rpc response: %w", err)
	}
	if decoded.Error != nil {
		return Batch{}, fmt.Errorf("stellar rpc getEvents error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if decoded.Result.Cursor == "" {
		return Batch{}, errors.New("stellar rpc getEvents returned no cursor")
	}
	events := make([]projection.Event, 0, len(decoded.Result.Events))
	for _, event := range decoded.Result.Events {
		normalized, err := normalize(c.Network, event)
		if err != nil {
			return Batch{}, err
		}
		if normalized.Controller != controller {
			return Batch{}, fmt.Errorf("stellar rpc returned event for controller %s", normalized.Controller)
		}
		events = append(events, normalized)
	}
	return Batch{Events: events, Cursor: decoded.Result.Cursor}, nil
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  getEventsParams `json:"params"`
}

type getEventsParams struct {
	StartLedger uint32        `json:"startLedger,omitempty"`
	Pagination  pagination    `json:"pagination"`
	Filters     []eventFilter `json:"filters"`
	XDRFormat   string        `json:"xdrFormat"`
}

type pagination struct {
	Limit  uint32 `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
}

type eventFilter struct {
	Type        string   `json:"type"`
	ContractIDs []string `json:"contractIds"`
}

type rpcResponse struct {
	Result getEventsResult `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type getEventsResult struct {
	Events []rawEvent `json:"events"`
	Cursor string     `json:"cursor"`
}

type rawEvent struct {
	Ledger     uint32  `json:"ledger"`
	ContractID string  `json:"contractId"`
	ID         string  `json:"id"`
	TxHash     string  `json:"txHash"`
	TopicJSON  []scVal `json:"topicJson"`
	ValueJSON  scVal   `json:"valueJson"`
}

type scVal map[string]json.RawMessage

func normalize(network string, event rawEvent) (projection.Event, error) {
	if len(event.TopicJSON) == 0 {
		return projection.Event{}, errors.New("event topic is required")
	}
	eventType, err := symbol(event.TopicJSON[0])
	if err != nil {
		return projection.Event{}, fmt.Errorf("decode event name: %w", err)
	}
	index, err := eventIndex(event.ID)
	if err != nil {
		return projection.Event{}, err
	}
	data, err := eventData(eventType, event)
	if err != nil {
		return projection.Event{}, fmt.Errorf("normalize %s: %w", eventType, err)
	}
	return projection.Event{
		Network:         network,
		Controller:      event.ContractID,
		RPCEventID:      event.ID,
		TransactionHash: event.TxHash,
		Ledger:          event.Ledger,
		Index:           index,
		Type:            eventType,
		Data:            data,
	}, nil
}

func eventData(eventType string, event rawEvent) (json.RawMessage, error) {
	value, err := scMap(event.ValueJSON)
	if err != nil {
		return nil, err
	}
	proposalID := func() (uint64, error) {
		if len(event.TopicJSON) < 2 {
			return 0, errors.New("proposal id topic is required")
		}
		return u64(event.TopicJSON[1])
	}
	switch eventType {
	case projection.ProposalCreated:
		id, err := proposalID()
		if err != nil {
			return nil, err
		}
		proposer, err := address(value["proposer"])
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{"proposal_id": id, "proposer": proposer})
	case projection.ProposalApproved, projection.ApprovalRevoked:
		id, err := proposalID()
		if err != nil {
			return nil, err
		}
		count, err := u32(value["approval_count"])
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{"proposal_id": id, "approval_count": count})
	case projection.ThresholdReached:
		id, err := proposalID()
		if err != nil {
			return nil, err
		}
		approvedLedger, err := u32(value["approved_ledger"])
		if err != nil {
			return nil, err
		}
		executeAfterLedger, err := u32(value["execute_after_ledger"])
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{
			"proposal_id":          id,
			"approved_ledger":      approvedLedger,
			"execute_after_ledger": executeAfterLedger,
		})
	case projection.ThresholdReset, projection.ProposalCancelled, projection.ProposalExecuted:
		id, err := proposalID()
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{"proposal_id": id})
	case projection.FleetCreated:
		fleetID, err := topicBytes(event.TopicJSON)
		if err != nil {
			return nil, err
		}
		tag, err := stringValue(value["tag"])
		if err != nil {
			return nil, err
		}
		wasmHash, err := bytesValue(value["wasm_hash"])
		if err != nil {
			return nil, err
		}
		ledger, err := u32(value["ledger"])
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{"fleet_id": fleetID, "tag": tag, "wasm_hash": wasmHash, "ledger": ledger})
	case projection.FleetUpgraded:
		fleetID, err := topicBytes(event.TopicJSON)
		if err != nil {
			return nil, err
		}
		id, err := u64(value["proposal_id"])
		if err != nil {
			return nil, err
		}
		oldHash, err := bytesValue(value["old_wasm_hash"])
		if err != nil {
			return nil, err
		}
		newHash, err := bytesValue(value["new_wasm_hash"])
		if err != nil {
			return nil, err
		}
		manifestHash, err := bytesValue(value["manifest_hash"])
		if err != nil {
			return nil, err
		}
		return marshalData(map[string]any{
			"fleet_id":      fleetID,
			"proposal_id":   id,
			"old_wasm_hash": oldHash,
			"new_wasm_hash": newHash,
			"manifest_hash": manifestHash,
		})
	case projection.PolicyUpdated, projection.ControllerUpgraded:
		return marshalData(map[string]any{})
	default:
		return nil, fmt.Errorf("unsupported UpgradeController event %q", eventType)
	}
}

func eventIndex(id string) (uint32, error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid event id %q", id)
	}
	value, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid event index %q: %w", parts[1], err)
	}
	return uint32(value), nil
}

func topicBytes(topics []scVal) (string, error) {
	if len(topics) < 2 {
		return "", errors.New("bytes topic is required")
	}
	return bytesValue(topics[1])
}

func scMap(value scVal) (map[string]scVal, error) {
	var entries []struct {
		Key scVal `json:"key"`
		Val scVal `json:"val"`
	}
	if err := json.Unmarshal(value["map"], &entries); err != nil {
		return nil, fmt.Errorf("decode event value map: %w", err)
	}
	result := map[string]scVal{}
	for _, entry := range entries {
		key, err := symbol(entry.Key)
		if err != nil {
			return nil, fmt.Errorf("decode map key: %w", err)
		}
		result[key] = entry.Val
	}
	return result, nil
}

func symbol(value scVal) (string, error) {
	return stringField(value, "symbol")
}

func address(value scVal) (string, error) {
	return stringField(value, "address")
}

func stringValue(value scVal) (string, error) {
	return stringField(value, "string")
}

func bytesValue(value scVal) (string, error) {
	return stringField(value, "bytes")
}

func stringField(value scVal, field string) (string, error) {
	raw, ok := value[field]
	if !ok {
		return "", fmt.Errorf("%s is required", field)
	}
	var decoded string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("decode %s: %w", field, err)
	}
	return decoded, nil
}

func u32(value scVal) (uint32, error) {
	raw, ok := value["u32"]
	if !ok {
		return 0, errors.New("u32 is required")
	}
	var decoded uint32
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("decode u32: %w", err)
	}
	return decoded, nil
}

func u64(value scVal) (uint64, error) {
	raw, ok := value["u64"]
	if !ok {
		return 0, errors.New("u64 is required")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strconv.ParseUint(text, 10, 64)
	}
	var decoded uint64
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return 0, fmt.Errorf("decode u64: %w", err)
	}
	return decoded, nil
}

func marshalData(value map[string]any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	return json.RawMessage(encoded), err
}
