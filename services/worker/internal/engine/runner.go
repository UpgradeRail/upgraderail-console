// Package engine invokes the configured UpgradeRail Engine binary.
package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const maxOutputBytes = 4 << 20

type Report struct {
	Status                string            `json:"status"`
	Findings              []json.RawMessage `json:"findings"`
	StorageCompatibility  string            `json:"storage_compatibility"`
	AuthorizationBehavior string            `json:"authorization_behavior"`
}

type Runner struct {
	Binary  string
	Timeout time.Duration
}

func (r Runner) Compare(ctx context.Context, workingDirectory, current, candidate string) (Report, error) {
	if r.Binary == "" {
		return Report{}, errors.New("engine binary is required")
	}
	if r.Timeout <= 0 {
		r.Timeout = 2 * time.Minute
	}
	if !filepath.IsAbs(workingDirectory) || !filepath.IsAbs(current) || !filepath.IsAbs(candidate) {
		return Report{}, errors.New("engine paths must be absolute")
	}

	path, err := exec.LookPath(r.Binary)
	if err != nil {
		return Report{}, fmt.Errorf("find engine binary: %w", err)
	}
	jobContext, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	command := exec.CommandContext(jobContext, path, "compare", "--current", current, "--candidate", candidate, "--format", "json")
	command.Dir = workingDirectory
	command.Env = safeEnvironment()
	var stdout, stderr limitedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if jobContext.Err() != nil {
		return Report{}, fmt.Errorf("engine timed out: %w", jobContext.Err())
	}

	report, decodeErr := decodeReport(stdout.Bytes())
	if decodeErr != nil {
		return Report{}, fmt.Errorf("decode engine JSON: %w; stderr: %s", decodeErr, stderr.String())
	}
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
			return Report{}, fmt.Errorf("engine exited: %w; stderr: %s", err, stderr.String())
		}
	}
	return report, nil
}

func decodeReport(data []byte) (Report, error) {
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return Report{}, err
	}
	if report.Status != "BLOCKED" && report.Status != "READY_WITH_WARNINGS" && report.Status != "READY" {
		return Report{}, fmt.Errorf("unsupported engine status %q", report.Status)
	}
	if report.StorageCompatibility == "" || report.AuthorizationBehavior == "" {
		return Report{}, errors.New("engine report is missing explicit evidence")
	}
	return report, nil
}

func safeEnvironment() []string {
	if path := os.Getenv("PATH"); path != "" {
		return []string{"PATH=" + path}
	}
	return nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(value []byte) (int, error) {
	if b.Len()+len(value) > maxOutputBytes {
		return 0, io.ErrShortBuffer
	}
	return b.Buffer.Write(value)
}
