package jobs

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
)

// A missing artifact must fail the job with a clear, honest error before
// the Engine binary is ever invoked, so this does not require the real
// Engine binary to be installed; an empty Binary would itself error if
// RunOne ever reached runner.Compare for this case, which it must not.
func TestRunOneFailsJobWhenArtifactBytesAreMissing(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	networkID := "testnet-worker-missing-" + suffix
	jobID := "job-missing-artifact-" + suffix

	artifactRoot := t.TempDir()
	// artifactID truncates SHA256 to its first 24 characters to build the
	// artifacts.id primary key, so the differentiator and the uniqueness
	// both have to live in that prefix, not just anywhere in the string.
	missingCurrent := artifactRecord{Key: "current/missing-" + suffix + ".wasm", SHA256: "c-" + suffix, Size: 1}
	missingCandidate := artifactRecord{Key: "candidate/missing-" + suffix + ".wasm", SHA256: "d-" + suffix, Size: 1}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO networks (id, passphrase, rpc_url, protocol_target)
		VALUES ($1, $2, 'https://soroban-testnet.stellar.org', 28)
	`, networkID, "test-passphrase-missing-"+suffix); err != nil {
		t.Fatal(err)
	}
	currentID := insertArtifact(t, ctx, store, missingCurrent)
	candidateID := insertArtifact(t, ctx, store, missingCandidate)
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO analysis_jobs (id, network_id, current_artifact_id, candidate_artifact_id, status)
		VALUES ($1, $2, $3, $4, 'queued')
	`, jobID, networkID, currentID, candidateID); err != nil {
		t.Fatal(err)
	}

	// Binary is intentionally left empty: if RunOne ever tried to invoke the
	// Engine for this job it would fail with an engine-binary error instead
	// of the expected artifact-read error, and this assertion would catch it.
	ran, err := RunOne(ctx, store, engine.Runner{Timeout: 10 * time.Second}, artifactstore.Filesystem{Directory: artifactRoot, MaxBytes: 1 << 62}, t.TempDir())
	if err == nil {
		t.Fatal("expected a missing current artifact to fail the job")
	}
	if !ran {
		t.Fatal("expected the queued job to be claimed")
	}
	if !strings.Contains(err.Error(), "current artifact") || !strings.Contains(err.Error(), "missing-") {
		t.Fatalf("expected a clear current-artifact error, got %v", err)
	}

	var status, message string
	if err := store.pool.QueryRow(ctx, `SELECT status, error_message FROM analysis_jobs WHERE id = $1`, jobID).Scan(&status, &message); err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("expected failed job, got %q", status)
	}
	if !strings.Contains(message, "missing-") {
		t.Fatalf("failure reason was not useful: %q", message)
	}
}
