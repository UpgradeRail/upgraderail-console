package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
)

func TestFetchControllerEventsRequestsJSONXDRAndNormalizesEvents(t *testing.T) {
	var request rpcRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"cursor":"cursor-1","events":[{"type":"contract","ledger":4969465,"contractId":"controller","id":"0021343689653669888-0000000000","txHash":"tx","topicJson":[{"symbol":"fleet_created"},{"bytes":"fleet"}],"valueJson":{"map":[{"key":{"symbol":"ledger"},"val":{"u32":4969465}},{"key":{"symbol":"tag"},"val":{"string":"shared"}},{"key":{"symbol":"wasm_hash"},"val":{"bytes":"wasm"}}]}}]}}`))
	}))
	defer server.Close()

	batch, err := Client{Endpoint: server.URL, Network: "testnet"}.FetchControllerEvents(context.Background(), "controller", 10, 5)
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != "getEvents" || request.Params.XDRFormat != "json" || request.Params.StartLedger != 10 || request.Params.Pagination.Limit != 5 {
		t.Fatalf("unexpected request %#v", request)
	}
	if len(request.Params.Filters) != 1 || request.Params.Filters[0].ContractIDs[0] != "controller" {
		t.Fatalf("unexpected filters %#v", request.Params.Filters)
	}
	if batch.Cursor != "cursor-1" || len(batch.Events) != 1 {
		t.Fatalf("unexpected batch %#v", batch)
	}
	event := batch.Events[0]
	if event.Type != projection.FleetCreated || event.Network != "testnet" || event.Controller != "controller" || event.Index != 0 || event.RPCEventID == "" {
		t.Fatalf("unexpected event %#v", event)
	}
	if string(event.Data) != `{"fleet_id":"fleet","ledger":4969465,"tag":"shared","wasm_hash":"wasm"}` {
		t.Fatalf("unexpected data %s", event.Data)
	}
}

func TestFetchControllerEventsAfterUsesCheckpointCursor(t *testing.T) {
	var request map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"cursor":"next-cursor","events":[]}}`))
	}))
	defer server.Close()
	batch, err := Client{Endpoint: server.URL, Network: "testnet"}.FetchControllerEventsAfter(context.Background(), "controller", "saved-cursor", 20)
	if err != nil {
		t.Fatal(err)
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(request["params"], &params); err != nil {
		t.Fatal(err)
	}
	if _, ok := params["startLedger"]; ok {
		t.Fatal("startLedger must be omitted when resuming from a cursor")
	}
	var page pagination
	if err := json.Unmarshal(params["pagination"], &page); err != nil {
		t.Fatal(err)
	}
	if page.Cursor != "saved-cursor" || batch.Cursor != "next-cursor" {
		t.Fatalf("unexpected cursors: %#v %#v", page, batch)
	}
}

func TestFetchControllerEventsClassifiesTemporaryFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	client := Client{Endpoint: server.URL, Network: "testnet"}
	_, err := client.FetchControllerEvents(context.Background(), "controller", 1, 10)
	var temporary *TemporaryError
	if !errors.As(err, &temporary) {
		t.Fatalf("HTTP 503 should be retryable: %v", err)
	}
	server.Close()
	_, err = client.FetchControllerEvents(context.Background(), "controller", 1, 10)
	if !errors.As(err, &temporary) {
		t.Fatalf("transport failure should be retryable: %v", err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer bad.Close()
	_, err = (Client{Endpoint: bad.URL, Network: "testnet"}).FetchControllerEvents(context.Background(), "controller", 1, 10)
	if err == nil || errors.As(err, &temporary) {
		t.Fatalf("HTTP 400 must remain fatal: %v", err)
	}
}

func TestCheckNetworkRejectsWrongPassphrase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"passphrase":"Test SDF Network ; September 2015"}}`))
	}))
	defer server.Close()
	client := Client{Endpoint: server.URL}
	if err := client.CheckNetwork(context.Background(), "Public Global Stellar Network ; September 2015"); err == nil {
		t.Fatal("wrong network was accepted")
	}
	if err := client.CheckNetwork(context.Background(), "Test SDF Network ; September 2015"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeRejectsUnknownControllerEvent(t *testing.T) {
	_, err := normalize("testnet", rawEvent{
		ContractID: "controller",
		ID:         "1-0000000000",
		TxHash:     "tx",
		TopicJSON:  []scVal{{"symbol": json.RawMessage(`"invented"`)}},
		ValueJSON:  scVal{"map": json.RawMessage(`[]`)},
	})
	if err == nil {
		t.Fatal("expected unknown event to fail")
	}
}

func TestFetchControllerEventsLive(t *testing.T) {
	if os.Getenv("STELLAR_RPC_LIVE") != "1" {
		t.Skip("STELLAR_RPC_LIVE=1 is required")
	}
	endpoint := os.Getenv("STELLAR_RPC_URL")
	controller := os.Getenv("UPGRADERAIL_CONTROLLER_ID")
	startLedgerText := os.Getenv("UPGRADERAIL_START_LEDGER")
	if endpoint == "" || controller == "" || startLedgerText == "" {
		t.Fatal("STELLAR_RPC_URL, UPGRADERAIL_CONTROLLER_ID, and UPGRADERAIL_START_LEDGER are required")
	}
	startLedger, err := strconv.ParseUint(startLedgerText, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := Client{Endpoint: endpoint, Network: "testnet"}.FetchControllerEvents(context.Background(), controller, uint32(startLedger), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Events) == 0 || batch.Cursor == "" {
		t.Fatalf("expected live events and cursor, got %#v", batch)
	}
	seen := map[string]bool{}
	for _, event := range batch.Events {
		seen[event.Type] = true
	}
	if !seen[projection.ProposalCreated] || !seen[projection.FleetCreated] {
		t.Fatalf("expected controller proposal and fleet events, saw %#v", seen)
	}
}
