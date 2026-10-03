package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/auth"
)

type Session = auth.Session
type AnalysisInput struct{ ID, Network, CurrentArtifactID, CandidateArtifactID, CreatedBy string }
type AnalysisJob struct {
	ID, Network, Status, EngineVersion, ErrorMessage string
	CurrentArtifactID, CreatedBy                     *string
	CandidateArtifactID                              string
	CreatedAt                                        time.Time
	StartedAt, FinishedAt                            *time.Time
}
type AnalysisReport struct {
	Status, CurrentWASMHash, CandidateWASMHash, EngineVersion string
	Findings, RuntimeEvidence, Report                         json.RawMessage
	CreatedAt                                                 time.Time
}
type Manifest struct {
	SHA256    string
	Bytes     []byte
	CreatedAt time.Time
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (auth.Session, error) {
	var value Session
	err := s.pool.QueryRow(ctx, `SELECT public_address, network_id FROM sessions WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`, tokenHash).Scan(&value.Address, &value.Network)
	if err != nil {
		return auth.Session{}, fmt.Errorf("get session: %w", err)
	}
	return value, nil
}
func (s *Store) CreateAnalysis(ctx context.Context, input AnalysisInput) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO analysis_jobs (id, network_id, current_artifact_id, candidate_artifact_id, status, created_by) VALUES ($1,$2,NULLIF($3,''),$4,'queued',$5)`, input.ID, input.Network, input.CurrentArtifactID, input.CandidateArtifactID, input.CreatedBy)
	if err != nil {
		return fmt.Errorf("create analysis: %w", err)
	}
	return nil
}
func (s *Store) GetAnalysis(ctx context.Context, id string) (AnalysisJob, error) {
	var value AnalysisJob
	err := s.pool.QueryRow(ctx, `SELECT id, network_id, current_artifact_id, candidate_artifact_id, status, COALESCE(engine_version,''), COALESCE(error_message,''), created_by, created_at, started_at, finished_at FROM analysis_jobs WHERE id = $1`, id).Scan(&value.ID, &value.Network, &value.CurrentArtifactID, &value.CandidateArtifactID, &value.Status, &value.EngineVersion, &value.ErrorMessage, &value.CreatedBy, &value.CreatedAt, &value.StartedAt, &value.FinishedAt)
	if err != nil {
		return AnalysisJob{}, fmt.Errorf("get analysis: %w", err)
	}
	return value, nil
}
func (s *Store) GetAnalysisReport(ctx context.Context, analysisID string) (AnalysisReport, error) {
	var value AnalysisReport
	err := s.pool.QueryRow(ctx, `SELECT status, current_wasm_hash, candidate_wasm_hash, findings, runtime_evidence, report, engine_version, created_at FROM analysis_reports WHERE analysis_job_id = $1`, analysisID).Scan(&value.Status, &value.CurrentWASMHash, &value.CandidateWASMHash, &value.Findings, &value.RuntimeEvidence, &value.Report, &value.EngineVersion, &value.CreatedAt)
	if err != nil {
		return AnalysisReport{}, fmt.Errorf("get analysis report: %w", err)
	}
	return value, nil
}
func (s *Store) GetManifest(ctx context.Context, analysisID string) (Manifest, error) {
	var value Manifest
	err := s.pool.QueryRow(ctx, `SELECT sha256, bytes, created_at FROM release_manifests WHERE analysis_job_id = $1`, analysisID).Scan(&value.SHA256, &value.Bytes, &value.CreatedAt)
	if err != nil {
		return Manifest{}, fmt.Errorf("get manifest: %w", err)
	}
	return value, nil
}
