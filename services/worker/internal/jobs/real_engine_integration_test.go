package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
)

func TestRunOneWithRealEnginePersistsReportAndManifest(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	engineBinary := os.Getenv("UPGRADERAIL_ENGINE_BIN")
	fixtureDirectory := os.Getenv("UPGRADERAIL_CONTRACT_FIXTURE_DIR")
	if databaseURL == "" || engineBinary == "" || fixtureDirectory == "" {
		t.Skip("DATABASE_URL, UPGRADERAIL_ENGINE_BIN, and UPGRADERAIL_CONTRACT_FIXTURE_DIR are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "testnet-worker-" + suffix
	jobID := "job-real-engine-" + suffix

	artifactRoot := t.TempDir()
	current := copyFixture(t, fixtureDirectory, artifactRoot, "fleet_v1.wasm", "current/fleet_v1.wasm")
	candidate := copyFixture(t, fixtureDirectory, artifactRoot, "fleet_v2_compatible.wasm", "candidate/fleet_v2_compatible.wasm")

	_, err = store.pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 28)
	`, networkID, "test-passphrase-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	currentID := insertArtifact(t, ctx, store, current)
	candidateID := insertArtifact(t, ctx, store, candidate)
	_, err = store.pool.Exec(ctx, `
		INSERT INTO analysis_jobs (id, network_id, current_artifact_id, candidate_artifact_id, status)
		VALUES ($1, $2, $3, $4, 'queued')
	`, jobID, networkID, currentID, candidateID)
	if err != nil {
		t.Fatal(err)
	}

	ran, err := RunOne(ctx, store, engine.Runner{Binary: engineBinary, Timeout: 60 * time.Second}, artifactstore.Filesystem{Directory: artifactRoot, MaxBytes: 1 << 62}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("expected queued job to run")
	}

	var status, engineVersion, reportStatus, storageEvidence, authEvidence, manifestHash string
	var manifestBytes []byte
	err = store.pool.QueryRow(ctx, `
		SELECT j.status, j.engine_version, r.status,
		       r.runtime_evidence->>'storage_compatibility',
		       r.runtime_evidence->>'authorization_behavior',
		       m.sha256, m.bytes
		FROM analysis_jobs j
		JOIN analysis_reports r ON r.analysis_job_id = j.id
		JOIN release_manifests m ON m.analysis_job_id = j.id
		WHERE j.id = $1
	`, jobID).Scan(&status, &engineVersion, &reportStatus, &storageEvidence, &authEvidence, &manifestHash, &manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if status != "ready" || reportStatus != "READY" {
		t.Fatalf("unexpected statuses: job=%q report=%q", status, reportStatus)
	}
	if !strings.Contains(engineVersion, "upgraderail 0.1.0") {
		t.Fatalf("unexpected engine version %q", engineVersion)
	}
	if storageEvidence == "" || authEvidence == "" {
		t.Fatalf("missing runtime evidence: storage=%q auth=%q", storageEvidence, authEvidence)
	}
	if len(manifestBytes) == 0 || len(manifestHash) != 64 {
		t.Fatalf("manifest evidence was not persisted: hash=%q bytes=%d", manifestHash, len(manifestBytes))
	}
}

func TestRunOneWithRealEngineRecordsFailureReason(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	engineBinary := os.Getenv("UPGRADERAIL_ENGINE_BIN")
	fixtureDirectory := os.Getenv("UPGRADERAIL_CONTRACT_FIXTURE_DIR")
	if databaseURL == "" || engineBinary == "" || fixtureDirectory == "" {
		t.Skip("DATABASE_URL, UPGRADERAIL_ENGINE_BIN, and UPGRADERAIL_CONTRACT_FIXTURE_DIR are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "testnet-worker-failure-" + suffix
	jobID := "job-real-engine-failure-" + suffix

	artifactRoot := t.TempDir()
	current := copyFixture(t, fixtureDirectory, artifactRoot, "fleet_v1.wasm", "current/fleet_v1.wasm")
	missing := artifactRecord{Key: "candidate/missing.wasm", SHA256: strings.Repeat("0", 64), Size: 1}

	_, err = store.pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 28)
	`, networkID, "test-passphrase-failure-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	currentID := insertArtifact(t, ctx, store, current)
	candidateID := insertArtifact(t, ctx, store, missing)
	_, err = store.pool.Exec(ctx, `
		INSERT INTO analysis_jobs (id, network_id, current_artifact_id, candidate_artifact_id, status)
		VALUES ($1, $2, $3, $4, 'queued')
	`, jobID, networkID, currentID, candidateID)
	if err != nil {
		t.Fatal(err)
	}

	ran, err := RunOne(ctx, store, engine.Runner{Binary: engineBinary, Timeout: 60 * time.Second}, artifactstore.Filesystem{Directory: artifactRoot, MaxBytes: 1 << 62}, t.TempDir())
	if err == nil {
		t.Fatal("expected engine failure for missing candidate artifact")
	}
	if !ran {
		t.Fatal("expected queued job to be claimed")
	}

	var status, message string
	if err := store.pool.QueryRow(ctx, `SELECT status, error_message FROM analysis_jobs WHERE id = $1`, jobID).Scan(&status, &message); err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("expected failed job, got %q", status)
	}
	// The missing artifact is caught while materializing it into the job
	// workspace, before the Engine binary is ever invoked, so the failure
	// reason names the artifact rather than describing an engine failure.
	if !strings.Contains(message, "artifact") || !strings.Contains(message, "missing.wasm") {
		t.Fatalf("failure reason was not useful: %q", message)
	}
}

type artifactRecord struct {
	Key    string
	SHA256 string
	Size   int64
}

func copyFixture(t *testing.T, fixtureDirectory, artifactRoot, fixtureName, storageKey string) artifactRecord {
	t.Helper()
	source := filepath.Join(fixtureDirectory, fixtureName)
	bytes, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(artifactRoot, storageKey)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	return artifactRecord{Key: storageKey, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(bytes))}
}

func insertArtifact(t *testing.T, ctx context.Context, store *Store, artifact artifactRecord) string {
	t.Helper()
	id := artifactID(artifact)
	err := store.pool.QueryRow(ctx, `
		INSERT INTO artifacts (id, sha256, size_bytes, storage_key, content_type)
		VALUES ($1, $2, $3, $4, 'application/wasm')
		ON CONFLICT (sha256) DO UPDATE
		SET size_bytes = EXCLUDED.size_bytes,
		    storage_key = EXCLUDED.storage_key,
		    content_type = EXCLUDED.content_type
		RETURNING id
	`, id, artifact.SHA256, artifact.Size, artifact.Key).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func artifactID(artifact artifactRecord) string {
	return "artifact-" + artifact.SHA256[:24]
}
