package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/auth"
	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
	"github.com/jackc/pgx/v5/pgxpool"
)

// realWASM is a minimal but genuinely valid WASM module: the magic header
// plus the version field, and nothing else. It is enough to pass the
// magic-byte check the upload endpoint performs; it is not a real Soroban
// contract, which is why TestUploadArtifactStoresRealHashAndMetadata uses an
// actual upgraderail-contracts fixture instead.
var realWASM = []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00}

type artifactTestEnv struct {
	url, token, networkID string
	client                *http.Client
	artifactRoot          string
}

// newArtifactTestEnv seeds a network and an active session against a real
// database and starts a real httptest server with a real filesystem
// artifact store, so the tests below exercise the actual upload and
// analysis-creation code paths end to end rather than mocks.
func newArtifactTestEnv(t *testing.T) artifactTestEnv {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seed, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(seed.Close)

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "artifact-net-" + suffix
	mustExec(t, ctx, seed, `INSERT INTO networks (id, passphrase, rpc_url, protocol_target) VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 29)`, networkID, "passphrase-art-"+suffix)

	token := "session-token-" + suffix
	mustExec(t, ctx, seed, `INSERT INTO sessions (id, public_address, network_id, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		"session-"+suffix, "GUPLOADER", networkID, auth.Hash(token), time.Now().Add(time.Hour))

	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(repository.Close)

	artifactRoot := t.TempDir()
	artifacts := artifactstore.Filesystem{Directory: artifactRoot, MaxBytes: MaxArtifactUploadBytes}
	server := httptest.NewServer(New(repository, "example.com", "https://example.com", artifacts))
	t.Cleanup(server.Close)

	return artifactTestEnv{url: server.URL, token: token, networkID: networkID, client: server.Client(), artifactRoot: artifactRoot}
}

func (env artifactTestEnv) upload(t *testing.T, content []byte, withSession bool) *http.Response {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "candidate.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, env.url+"/api/v1/artifacts", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if withSession {
		request.AddCookie(&http.Cookie{Name: "upgraderail_session", Value: env.token})
	}
	response, err := env.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func (env artifactTestEnv) createAnalysis(t *testing.T, currentID, candidateID string) *http.Response {
	t.Helper()
	payload := map[string]string{"network": env.networkID, "current_artifact_id": currentID, "candidate_artifact_id": candidateID}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, env.url+"/api/v1/analyses", bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "upgraderail_session", Value: env.token})
	response, err := env.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decodeJSONObject(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	defer response.Body.Close()
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

// uniqueFixture reads a real upgraderail-contracts WASM fixture and
// appends a tag unique to this test run. Artifacts are deduplicated by
// content hash, and the test database is shared across test runs and
// packages (including the worker package's own artifact fixtures, which
// use a different storage-key convention), so re-using an unmodified
// fixture's exact bytes risks silently reading back a stale row left by
// an unrelated test. The upload endpoint only checks the leading magic
// bytes, so an appended tag is still exercised as a realistic upload.
func uniqueFixture(t *testing.T, path, tag string) ([]byte, error) {
	t.Helper()
	base, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	suffix := "tag:" + tag + ":" + t.Name() + ":" + time.Now().UTC().Format(time.RFC3339Nano)
	return append(append([]byte{}, base...), []byte(suffix)...), nil
}

func TestUploadArtifactStoresRealHashAndMetadata(t *testing.T) {
	env := newArtifactTestEnv(t)
	fixture, err := uniqueFixture(t, "/home/gamp/upgraderail-contracts/fixtures/wasm/fleet_v1.wasm", "metadata")
	if err != nil {
		t.Skipf("upgraderail-contracts WASM fixture not available: %v", err)
	}
	expectedHash := sha256.Sum256(fixture)
	expectedHex := hex.EncodeToString(expectedHash[:])

	response := env.upload(t, fixture, true)
	if response.StatusCode != http.StatusCreated {
		body := decodeJSONObject(t, response)
		t.Fatalf("expected 201, got %d: %v", response.StatusCode, body)
	}
	body := decodeJSONObject(t, response)
	if body["sha256"] != expectedHex {
		t.Fatalf("expected server-computed hash %s, got %v", expectedHex, body["sha256"])
	}
	if body["content_type"] != "application/wasm" {
		t.Fatalf("unexpected content_type: %v", body["content_type"])
	}
	if size, ok := body["size_bytes"].(float64); !ok || int64(size) != int64(len(fixture)) {
		t.Fatalf("unexpected size_bytes: %v", body["size_bytes"])
	}
	if body["uploader"] != "GUPLOADER" {
		t.Fatalf("expected uploader to be the session address, got %v", body["uploader"])
	}
	id, ok := body["id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected a stable artifact id, got %v", body["id"])
	}

	// The bytes must be retrievable from the same storage abstraction the
	// worker reads from, addressed by the server-computed hash.
	stored := artifactstore.Filesystem{Directory: env.artifactRoot, MaxBytes: MaxArtifactUploadBytes}
	key := filepath.Join(expectedHex[:2], expectedHex)
	file, err := stored.Open(key)
	if err != nil {
		t.Fatalf("expected the artifact to be readable from storage: %v", err)
	}
	onDisk, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(onDisk, fixture) {
		t.Fatal("bytes stored on disk do not match the uploaded bytes")
	}

	// Re-uploading identical bytes is idempotent: same content hashes to
	// the same artifact id rather than creating a duplicate row.
	again := env.upload(t, fixture, true)
	secondBody := decodeJSONObject(t, again)
	if secondBody["id"] != id {
		t.Fatalf("expected re-uploading identical bytes to return the same artifact id, got %v vs %v", secondBody["id"], id)
	}
}

func TestUploadArtifactRejectsNonWASM(t *testing.T) {
	env := newArtifactTestEnv(t)
	response := env.upload(t, []byte("this is not a wasm module, just plain text"), true)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-WASM upload, got %d", response.StatusCode)
	}
}

func TestUploadArtifactRejectsEmptyFile(t *testing.T) {
	env := newArtifactTestEnv(t)
	response := env.upload(t, []byte{}, true)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an empty upload, got %d", response.StatusCode)
	}
}

func TestUploadArtifactRejectsOversizedUpload(t *testing.T) {
	env := newArtifactTestEnv(t)
	oversized := make([]byte, MaxArtifactUploadBytes+1024)
	copy(oversized, realWASM)
	response := env.upload(t, oversized, true)
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for an oversized upload, got %d", response.StatusCode)
	}
}

func TestUploadArtifactRequiresSession(t *testing.T) {
	env := newArtifactTestEnv(t)
	response := env.upload(t, realWASM, false)
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", response.StatusCode)
	}
}

func TestCreateAnalysisRejectsUnknownArtifacts(t *testing.T) {
	env := newArtifactTestEnv(t)
	response := env.createAnalysis(t, "does-not-exist", "also-missing")
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown artifacts, got %d", response.StatusCode)
	}
}

func TestCreateAnalysisAcceptsUploadedArtifacts(t *testing.T) {
	env := newArtifactTestEnv(t)
	current, err := uniqueFixture(t, "/home/gamp/upgraderail-contracts/fixtures/wasm/fleet_v1.wasm", "current")
	if err != nil {
		t.Skipf("upgraderail-contracts WASM fixture not available: %v", err)
	}
	candidate, err := uniqueFixture(t, "/home/gamp/upgraderail-contracts/fixtures/wasm/fleet_v2_compatible.wasm", "candidate")
	if err != nil {
		t.Skipf("upgraderail-contracts WASM fixture not available: %v", err)
	}

	currentBody := decodeJSONObject(t, env.upload(t, current, true))
	candidateBody := decodeJSONObject(t, env.upload(t, candidate, true))
	currentID, _ := currentBody["id"].(string)
	candidateID, _ := candidateBody["id"].(string)
	if currentID == "" || candidateID == "" {
		t.Fatalf("expected both uploads to succeed: %v / %v", currentBody, candidateBody)
	}

	response := env.createAnalysis(t, currentID, candidateID)
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		body := decodeJSONObject(t, response)
		t.Fatalf("expected 202, got %d: %v", response.StatusCode, body)
	}
}
