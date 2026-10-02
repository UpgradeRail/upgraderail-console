package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
)

func RunOne(ctx context.Context, store *Store, runner engine.Runner, artifactRoot, workspaceRoot string) (bool, error) {
	job, err := store.Claim(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	fail := func(err error) (bool, error) { _ = store.Fail(ctx, job.ID, err); return true, err }
	current, err := artifactPath(artifactRoot, job.CurrentKey)
	if err != nil {
		return fail(err)
	}
	candidate, err := artifactPath(artifactRoot, job.CandidateKey)
	if err != nil {
		return fail(err)
	}
	workspace, err := os.MkdirTemp(workspaceRoot, "upgraderail-job-*")
	if err != nil {
		return fail(fmt.Errorf("create job workspace: %w", err))
	}
	defer os.RemoveAll(workspace)
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

func artifactPath(root, key string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", errors.New("artifact root must be absolute")
	}
	cleaned := filepath.Clean(key)
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact storage key is invalid")
	}
	return filepath.Join(root, cleaned), nil
}
