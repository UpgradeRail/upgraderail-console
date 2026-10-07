package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
)

// RunOne claims at most one queued analysis job and runs it to completion
// (ready, blocked, or failed). It returns true if a job was claimed, so the
// caller can poll without sleeping between busy periods.
func RunOne(ctx context.Context, store *Store, runner engine.Runner, artifacts artifactstore.Store, workspaceRoot string) (bool, error) {
	job, err := store.Claim(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	fail := func(err error) (bool, error) { _ = store.Fail(ctx, job.ID, err); return true, err }

	workspace, err := os.MkdirTemp(workspaceRoot, "upgraderail-job-*")
	if err != nil {
		return fail(fmt.Errorf("create job workspace: %w", err))
	}
	defer os.RemoveAll(workspace)

	current, err := materialize(ctx, artifacts, job.CurrentKey, filepath.Join(workspace, "current.wasm"))
	if err != nil {
		return fail(fmt.Errorf("read current artifact: %w", err))
	}
	candidate, err := materialize(ctx, artifacts, job.CandidateKey, filepath.Join(workspace, "candidate.wasm"))
	if err != nil {
		return fail(fmt.Errorf("read candidate artifact: %w", err))
	}

	version, err := runner.Version(ctx, workspace)
	if err != nil {
		return fail(err)
	}
	report, err := runner.Compare(ctx, workspace, current, candidate)
	if err != nil {
		return fail(err)
	}
	manifest, err := runner.BuildManifest(ctx, workspace, current, candidate, "analysis-"+job.ID)
	if err != nil {
		return fail(err)
	}
	if err := store.Complete(ctx, *job, version, report, manifest); err != nil {
		return true, err
	}
	return true, nil
}

// materialize copies an artifact's stored bytes to an absolute path inside
// the job workspace so the Engine binary (which takes file paths, not
// readers) can read it. A missing or unreadable artifact becomes a clear
// failure, not a crash.
func materialize(ctx context.Context, artifacts artifactstore.Store, key, destination string) (string, error) {
	source, err := artifacts.Open(ctx, key)
	if err != nil {
		return "", err
	}
	defer source.Close()
	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create workspace artifact copy: %w", err)
	}
	defer destinationFile.Close()
	if _, err := io.Copy(destinationFile, source); err != nil {
		return "", fmt.Errorf("copy artifact into workspace: %w", err)
	}
	return destination, nil
}
