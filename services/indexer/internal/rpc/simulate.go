// Proposal detail reads. These use Soroban's simulateTransaction (read-only
// simulation) to call the UpgradeController's get_proposal method; nothing
// here signs, submits, or otherwise mutates chain state. The transaction
// built below is never sent to the network — it exists only so the RPC
// server has something to simulate.
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stellar/go-stellar-sdk/keypair"
	protocol "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
)

// ContractReadError marks a successfully-delivered RPC response in which the
// contract itself reported an error (e.g. an unknown proposal id). It is not
// retryable: a different simulation of the same call will not change it.
type ContractReadError struct{ Message string }

func (e *ContractReadError) Error() string { return "contract returned an error: " + e.Message }

// ProposalKindResult is what reconciliation needs out of UpgradeController's
// get_proposal: the ProposalKind variant tag, and the manifest_hash the
// variant's payload carries, if any.
//
// ManifestHash is nil in exactly two cases, which callers must not conflate:
//   - the variant is UpdatePolicy, whose payload has no manifest_hash field
//     at all (genuinely unknown, not a read failure); or
//   - this result was never produced because the read itself failed (callers
//     get a non-nil error instead, and must not treat a zero-value
//     ProposalKindResult as "reconciled").
type ProposalKindResult struct {
	Kind         string
	ManifestHash *string
}

// GetProposalKind reads one proposal's kind/manifest_hash from the live
// UpgradeController contract via simulateTransaction. It performs no chain
// mutation: the built transaction is simulated only, using a throwaway,
// never-signed source keypair, and is never submitted.
func (c Client) GetProposalKind(ctx context.Context, controllerID string, proposalID uint64, passphrase string) (ProposalKindResult, error) {
	if c.Endpoint == "" {
		return ProposalKindResult{}, errors.New("stellar rpc endpoint is required")
	}
	if passphrase == "" {
		return ProposalKindResult{}, errors.New("network passphrase is required")
	}
	if controllerID == "" {
		return ProposalKindResult{}, errors.New("controller id is required")
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	txXDR, err := buildGetProposalTransaction(controllerID, proposalID, passphrase)
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("build get_proposal simulation: %w", err)
	}

	client := rpcclient.NewClient(c.Endpoint, httpClient)
	defer client.Close()

	resp, err := client.SimulateTransaction(rpcCtx, protocol.SimulateTransactionRequest{
		Transaction: txXDR,
		Format:      "json",
	})
	if err != nil {
		return ProposalKindResult{}, &TemporaryError{Err: fmt.Errorf("stellar rpc simulateTransaction: %w", err)}
	}
	if resp.Error != "" {
		// The contract itself rejected the call (e.g. ProposalNotFound). This is
		// a definitive answer, not a transport failure, so it is not retryable.
		return ProposalKindResult{}, &ContractReadError{Message: resp.Error}
	}
	if len(resp.Results) == 0 || resp.Results[0].ReturnValueJSON == nil {
		return ProposalKindResult{}, errors.New("stellar rpc simulateTransaction returned no result for get_proposal")
	}

	return decodeGetProposalResult(resp.Results[0].ReturnValueJSON)
}

// buildGetProposalTransaction builds an unsigned, never-submitted transaction
// invoking UpgradeController.get_proposal(proposal_id). The source account is
// a freshly generated keypair used only for its public address; its private
// key is discarded immediately and never used to sign anything.
func buildGetProposalTransaction(controllerID string, proposalID uint64, passphrase string) (string, error) {
	source := keypair.MustRandom()

	decoded, err := strkey.Decode(strkey.VersionByteContract, controllerID)
	if err != nil {
		return "", fmt.Errorf("decode controller contract id: %w", err)
	}
	var contractID xdr.ContractId
	copy(contractID[:], decoded)

	idValue := xdr.Uint64(proposalID)
	op := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: xdr.ScAddress{
					Type:       xdr.ScAddressTypeScAddressTypeContract,
					ContractId: &contractID,
				},
				FunctionName: "get_proposal",
				Args: xdr.ScVec{
					xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: &idValue},
				},
			},
		},
		SourceAccount: source.Address(),
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{AccountID: source.Address(), Sequence: 0},
		Operations:    []txnbuild.Operation{op},
		BaseFee:       txnbuild.MinBaseFee,
		Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(60)},
	})
	if err != nil {
		return "", fmt.Errorf("build transaction: %w", err)
	}
	_ = passphrase // reserved for signing; this transaction is never signed or sent.
	return tx.Base64()
}

// decodeGetProposalResult decodes the JSON-XDR scVal for get_proposal's
// return value. The contract's get_proposal returns the Proposal struct
// directly on success (a Soroban Result<T, E> return fails the simulation
// rather than appearing as an on-wire wrapper), so this decodes the Proposal
// struct's "kind" field: a ProposalKind enum, which Soroban encodes as a Vec
// of [variant_name_symbol, payload_struct]. Only the fields reconciliation
// needs (kind, manifest_hash) are decoded; an unrecognized shape is a hard
// error rather than a silently-wrong value.
func decodeGetProposalResult(raw json.RawMessage) (ProposalKindResult, error) {
	var top scVal
	if err := json.Unmarshal(raw, &top); err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode get_proposal return value: %w", err)
	}
	fields, err := scMap(top)
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode proposal struct: %w", err)
	}
	kindValue, ok := fields["kind"]
	if !ok {
		return ProposalKindResult{}, errors.New("proposal struct has no kind field")
	}
	return decodeProposalKind(kindValue)
}

func decodeProposalKind(value scVal) (ProposalKindResult, error) {
	items, err := vecOf(value)
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode proposal kind: %w", err)
	}
	if len(items) == 0 {
		return ProposalKindResult{}, errors.New("proposal kind vec is empty")
	}
	tag, err := symbol(items[0])
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode proposal kind tag: %w", err)
	}
	if len(items) < 2 {
		// All four real ProposalKind variants carry a payload; a tag with no
		// payload is an unrecognized shape, not a value to guess at.
		return ProposalKindResult{}, fmt.Errorf("proposal kind %q has no payload", tag)
	}
	payload, err := scMap(items[1])
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode proposal kind %q payload: %w", tag, err)
	}
	manifestRaw, ok := payload["manifest_hash"]
	if !ok {
		// UpdatePolicy's payload (GovernancePolicy) genuinely has no
		// manifest_hash field. This is "unknown", not "read failed".
		return ProposalKindResult{Kind: tag}, nil
	}
	hash, err := bytesValue(manifestRaw)
	if err != nil {
		return ProposalKindResult{}, fmt.Errorf("decode proposal kind %q manifest_hash: %w", tag, err)
	}
	return ProposalKindResult{Kind: tag, ManifestHash: &hash}, nil
}

func vecOf(value scVal) ([]scVal, error) {
	raw, ok := value["vec"]
	if !ok {
		return nil, errors.New("vec is required")
	}
	var items []scVal
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode vec: %w", err)
	}
	return items, nil
}
