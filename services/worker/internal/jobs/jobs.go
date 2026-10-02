// Package jobs persists analysis queue transitions atomically.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }
type Job struct{ ID, CurrentKey, CandidateKey, CurrentHash, CandidateHash string }

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}
func (s *Store) Close()                         { s.pool.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Claim(ctx context.Context) (*Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var job Job
	err = tx.QueryRow(ctx, `SELECT j.id, current_artifact.storage_key, candidate_artifact.storage_key, current_artifact.sha256, candidate_artifact.sha256 FROM analysis_jobs j JOIN artifacts current_artifact ON current_artifact.id = j.current_artifact_id JOIN artifacts candidate_artifact ON candidate_artifact.id = j.candidate_artifact_id WHERE j.status = 'queued' ORDER BY j.created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&job.ID, &job.CurrentKey, &job.CandidateKey, &job.CurrentHash, &job.CandidateHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE analysis_jobs SET status = 'running', started_at = now(), updated_at = now() WHERE id = $1`, job.ID); err != nil {
		return nil, fmt.Errorf("mark job running: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit job claim: %w", err)
	}
	return &job, nil
}

func (s *Store) Complete(ctx context.Context, job Job, version string, report engine.Report, manifest engine.Manifest) error {
	encodedReport, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	runtime, err := json.Marshal(map[string]string{"storage_compatibility": report.StorageCompatibility, "authorization_behavior": report.AuthorizationBehavior})
	if err != nil {
		return err
	}
	status := "ready"
	if report.Status == "BLOCKED" {
		status = "blocked"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO analysis_reports (id, analysis_job_id, current_wasm_hash, candidate_wasm_hash, status, findings, runtime_evidence, report, engine_version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, job.ID+":report", job.ID, job.CurrentHash, job.CandidateHash, report.Status, report.Findings, runtime, encodedReport, version)
	if err != nil {
		return fmt.Errorf("store report: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO release_manifests (id, analysis_job_id, sha256, bytes) VALUES ($1,$2,$3,$4)`, job.ID+":manifest", job.ID, manifest.SHA256, manifest.Bytes)
	if err != nil {
		return fmt.Errorf("store manifest: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE analysis_jobs SET status = $1, engine_version = $2, finished_at = now(), updated_at = now() WHERE id = $3 AND status = 'running'`, status, version, job.ID)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) Fail(ctx context.Context, id string, failure error) error {
	_, err := s.pool.Exec(ctx, `UPDATE analysis_jobs SET status = 'failed', error_message = $1, finished_at = now(), updated_at = now() WHERE id = $2 AND status = 'running'`, failure.Error(), id)
	return err
}
func (s *Store) Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
