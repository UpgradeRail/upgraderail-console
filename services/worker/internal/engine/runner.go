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

type Manifest struct {
	Bytes  []byte
	SHA256 string
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

func (r Runner) BuildManifest(ctx context.Context, workingDirectory, current, candidate, releaseName string) (Manifest, error) {
	if r.Binary == "" {
		return Manifest{}, errors.New("engine binary is required")
	}
	if !filepath.IsAbs(workingDirectory) || !filepath.IsAbs(current) || !filepath.IsAbs(candidate) {
		return Manifest{}, errors.New("engine paths must be absolute")
	}
	configPath := filepath.Join(workingDirectory, "upgraderail.toml")
	manifestPath := filepath.Join(workingDirectory, "release-manifest.json")
	config := fmt.Sprintf("schema_version = 1\n\n[project]\nname = %q\n\n[analysis]\nprotocol_profile = 28\ncurrent_wasm = %q\ncandidate_wasm = %q\n\n[policy]\nrequire_simulation = false\n", releaseName, current, candidate)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		return Manifest{}, fmt.Errorf("write engine config: %w", err)
	}
	if _, err := r.run(ctx, workingDirectory, "manifest", "build", "--config", configPath, "--out", manifestPath); err != nil {
		return Manifest{}, err
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("read engine manifest: %w", err)
	}
	output, err := r.run(ctx, workingDirectory, "manifest", "hash", manifestPath)
	if err != nil {
		return Manifest{}, err
	}
	hash := string(bytes.TrimSpace(output))
	if len(hash) != 64 {
		return Manifest{}, fmt.Errorf("invalid engine manifest hash %q", hash)
	}
	return Manifest{Bytes: manifestBytes, SHA256: hash}, nil
}

func (r Runner) Version(ctx context.Context, workingDirectory string) (string, error) {
	output, err := r.run(ctx, workingDirectory, "version")
	return string(bytes.TrimSpace(output)), err
}

func (r Runner) run(ctx context.Context, workingDirectory string, arguments ...string) ([]byte, error) {
	if r.Timeout <= 0 {
		r.Timeout = 2 * time.Minute
	}
	path, err := exec.LookPath(r.Binary)
	if err != nil {
		return nil, fmt.Errorf("find engine binary: %w", err)
	}
	jobContext, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	command := exec.CommandContext(jobContext, path, arguments...)
	command.Dir = workingDirectory
	command.Env = safeEnvironment()
	var stdout, stderr limitedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if jobContext.Err() != nil {
		return nil, fmt.Errorf("engine timed out: %w", jobContext.Err())
	}
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
			return nil, fmt.Errorf("engine exited: %w; stderr: %s", err, stderr.String())
		}
	}
	return stdout.Bytes(), nil
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
