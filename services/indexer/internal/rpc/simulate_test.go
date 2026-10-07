package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Realistic get_proposal return-value fixtures, shaped the way stellar-rpc's
// xdrFormat=json encodes the real UpgradeController's Proposal/ProposalKind
// types (packages/contracts/generated/src/index.ts): a Proposal struct map
// whose "kind" field is a ProposalKind enum, encoded as a Vec of
// [variant_name_symbol, variant_payload_struct].
const createFleetProposalJSON = `{"map":[
	{"key":{"symbol":"approval_count"},"val":{"u32":2}},
	{"key":{"symbol":"created_ledger"},"val":{"u32":100}},
	{"key":{"symbol":"expires_ledger"},"val":{"u32":5000}},
	{"key":{"symbol":"governance_epoch"},"val":{"u64":"1"}},
	{"key":{"symbol":"id"},"val":{"u64":"42"}},
	{"key":{"symbol":"kind"},"val":{"vec":[
		{"symbol":"CreateFleet"},
		{"map":[
			{"key":{"symbol":"fleet_id"},"val":{"bytes":"66c6565745f6964"}},
			{"key":{"symbol":"initial_wasm_hash"},"val":{"bytes":"696e697469616c5f68617368"}},
			{"key":{"symbol":"manifest_hash"},"val":{"bytes":"6d616e69666573745f686173685f3432"}},
			{"key":{"symbol":"tag"},"val":{"string":"shared"}}
		]}
	]}},
	{"key":{"symbol":"proposer"},"val":{"address":"GPROPOSER"}},
	{"key":{"symbol":"status"},"val":{"vec":[{"symbol":"Active"}]}}
]}`

const upgradeControllerProposalJSON = `{"map":[
	{"key":{"symbol":"id"},"val":{"u64":"7"}},
	{"key":{"symbol":"kind"},"val":{"vec":[
		{"symbol":"UpgradeController"},
		{"map":[
			{"key":{"symbol":"expected_controller_version"},"val":{"u32":3}},
			{"key":{"symbol":"manifest_hash"},"val":{"bytes":"6d616e69666573745f686173685f37"}},
			{"key":{"symbol":"new_controller_version"},"val":{"u32":4}},
			{"key":{"symbol":"new_wasm_hash"},"val":{"bytes":"6e65775f68617368"}}
		]}
	]}}
]}`

const updatePolicyProposalJSON = `{"map":[
	{"key":{"symbol":"id"},"val":{"u64":"9"}},
	{"key":{"symbol":"kind"},"val":{"vec":[
		{"symbol":"UpdatePolicy"},
		{"map":[
			{"key":{"symbol":"policy"},"val":{"map":[
				{"key":{"symbol":"threshold"},"val":{"u32":2}}
			]}}
		]}
	]}}
]}`

func TestDecodeGetProposalResultCreateFleet(t *testing.T) {
	result, err := decodeGetProposalResult(json.RawMessage(createFleetProposalJSON))
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "CreateFleet" {
		t.Fatalf("expected kind CreateFleet, got %q", result.Kind)
	}
	if result.ManifestHash == nil || *result.ManifestHash != "6d616e69666573745f686173685f3432" {
		t.Fatalf("unexpected manifest hash %v", result.ManifestHash)
	}
}

func TestDecodeGetProposalResultUpgradeController(t *testing.T) {
	result, err := decodeGetProposalResult(json.RawMessage(upgradeControllerProposalJSON))
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "UpgradeController" {
		t.Fatalf("expected kind UpgradeController, got %q", result.Kind)
	}
	if result.ManifestHash == nil || *result.ManifestHash != "6d616e69666573745f686173685f37" {
		t.Fatalf("unexpected manifest hash %v", result.ManifestHash)
	}
}

// UpdatePolicyProposal has no manifest_hash field at all: this is "unknown",
// not a decode failure, and must not fabricate a value.
func TestDecodeGetProposalResultUpdatePolicyHasNoManifestHash(t *testing.T) {
	result, err := decodeGetProposalResult(json.RawMessage(updatePolicyProposalJSON))
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "UpdatePolicy" {
		t.Fatalf("expected kind UpdatePolicy, got %q", result.Kind)
	}
	if result.ManifestHash != nil {
		t.Fatalf("expected nil manifest hash for UpdatePolicy, got %v", *result.ManifestHash)
	}
}

func TestDecodeGetProposalResultRejectsUnrecognizedShape(t *testing.T) {
	cases := map[string]string{
		"not a struct":         `{"u32":1}`,
		"missing kind field":   `{"map":[{"key":{"symbol":"id"},"val":{"u64":"1"}}]}`,
		"kind not a vec":       `{"map":[{"key":{"symbol":"kind"},"val":{"symbol":"CreateFleet"}}]}`,
		"empty kind vec":       `{"map":[{"key":{"symbol":"kind"},"val":{"vec":[]}}]}`,
		"kind with no payload": `{"map":[{"key":{"symbol":"kind"},"val":{"vec":[{"symbol":"CreateFleet"}]}}]}`,
	}
	for name, fixture := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeGetProposalResult(json.RawMessage(fixture)); err == nil {
				t.Fatalf("expected a decode error for %s, got nil", name)
			}
		})
	}
}

func TestGetProposalKindRequestsSimulateTransactionWithJSONFormat(t *testing.T) {
	var request struct {
		Method string `json:"method"`
		Params struct {
			Transaction string `json:"transaction"`
			Format      string `json:"xdrFormat"`
		} `json:"params"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"latestLedger":100,"minResourceFee":"100","results":[{"returnValueJson":` + createFleetProposalJSON + `}]}}`))
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, Network: "testnet"}
	result, err := client.GetProposalKind(context.Background(), "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3", 42, "Test SDF Network ; September 2015")
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != "simulateTransaction" {
		t.Fatalf("expected simulateTransaction request, got %q", request.Method)
	}
	if request.Params.Format != "json" {
		t.Fatalf("expected xdrFormat json, got %q", request.Params.Format)
	}
	if request.Params.Transaction == "" {
		t.Fatal("expected a non-empty transaction envelope")
	}
	if result.Kind != "CreateFleet" || result.ManifestHash == nil {
		t.Fatalf("unexpected result %#v", result)
	}
}

func TestGetProposalKindReturnsContractReadErrorForContractFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"latestLedger":100,"error":"HostError: Error(Contract, #50)"}}`))
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, Network: "testnet"}
	_, err := client.GetProposalKind(context.Background(), "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3", 999, "Test SDF Network ; September 2015")
	var contractErr *ContractReadError
	if !errors.As(err, &contractErr) {
		t.Fatalf("expected a ContractReadError, got %v (%T)", err, err)
	}
}

func TestGetProposalKindClassifiesTransportFailureAsTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := Client{Endpoint: server.URL, Network: "testnet"}
	_, err := client.GetProposalKind(context.Background(), "CCMC4WGOCRU34RYO4YK64QVDNOMJBS27SBHH42ARQ7NOCQPZMRZMSLR3", 42, "Test SDF Network ; September 2015")
	var temporary *TemporaryError
	if !errors.As(err, &temporary) {
		t.Fatalf("expected a TemporaryError, got %v (%T)", err, err)
	}
}
