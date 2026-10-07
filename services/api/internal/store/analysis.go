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
	ID                  string     `json:"id"`
	Network             string     `json:"network"`
	Status              string     `json:"status"`
	EngineVersion       string     `json:"engine_version"`
	ErrorMessage        string     `json:"error_message"`
	CurrentArtifactID   *string    `json:"current_artifact_id"`
	CandidateArtifactID string     `json:"candidate_artifact_id"`
	CreatedBy           *string    `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	StartedAt           *time.Time `json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at"`
}
type AnalysisReport struct {
	Status            string          `json:"status"`
	CurrentWASMHash   string          `json:"current_wasm_hash"`
	CandidateWASMHash string          `json:"candidate_wasm_hash"`
	Findings          json.RawMessage `json:"findings"`
	RuntimeEvidence   json.RawMessage `json:"runtime_evidence"`
	Report            json.RawMessage `json:"report"`
	EngineVersion     string          `json:"engine_version"`
	CreatedAt         time.Time       `json:"created_at"`
}
type Manifest struct {
	SHA256    string    `json:"sha256"`
	Bytes     []byte    `json:"bytes"`
	CreatedAt time.Time `json:"created_at"`
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

func (s *Store) ListAnalyses(ctx context.Context, page Page) ([]AnalysisJob, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, network_id, status, COALESCE(engine_version,''), COALESCE(error_message,''), current_artifact_id, candidate_artifact_id, created_by, created_at, started_at, finished_at FROM analysis_jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}
	defer rows.Close()
	values := []AnalysisJob{}
	for rows.Next() {
		var value AnalysisJob
		if err := rows.Scan(&value.ID, &value.Network, &value.Status, &value.EngineVersion, &value.ErrorMessage, &value.CurrentArtifactID, &value.CandidateArtifactID, &value.CreatedBy, &value.CreatedAt, &value.StartedAt, &value.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan analysis: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
