package store

import (
	"context"
	"fmt"
	"time"
)

// Artifact is the metadata row for a stored WASM artifact. The bytes
// themselves live in the artifact storage backend (filesystem today),
// addressed by StorageKey; this row is what the API and worker use to
// look an artifact up by a stable ID instead of a content hash.
type Artifact struct {
	ID          string    `json:"id"`
	SHA256      string    `json:"sha256"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentType string    `json:"content_type"`
	Uploader    *string   `json:"uploader"`
	CreatedAt   time.Time `json:"created_at"`
	// StorageKey locates the artifact's bytes in the storage backend. It is
	// an internal implementation detail (equivalent information to SHA256,
	// since the key is derived from it), so it is deliberately excluded
	// from the JSON the API returns to clients.
	StorageKey string `json:"-"`
}

type ArtifactInput struct {
	ID, SHA256, StorageKey, ContentType string
	SizeBytes                           int64
	Uploader                            *string
}

// CreateArtifact persists artifact metadata. Artifacts are content
// addressed by SHA-256 (enforced by the artifacts.sha256 unique
// constraint): if a row for this hash already exists, that existing row
// is returned unchanged rather than inserting a duplicate, since the same
// bytes already live at the same storage key.
func (s *Store) CreateArtifact(ctx context.Context, input ArtifactInput) (Artifact, error) {
	var value Artifact
	err := s.pool.QueryRow(ctx, `
		INSERT INTO artifacts (id, sha256, size_bytes, storage_key, content_type, uploader)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
		RETURNING id, sha256, size_bytes, content_type, uploader, created_at, storage_key
	`, input.ID, input.SHA256, input.SizeBytes, input.StorageKey, input.ContentType, input.Uploader).
		Scan(&value.ID, &value.SHA256, &value.SizeBytes, &value.ContentType, &value.Uploader, &value.CreatedAt, &value.StorageKey)
	if err != nil {
		return Artifact{}, fmt.Errorf("create artifact: %w", err)
	}
	return value, nil
}

func (s *Store) GetArtifact(ctx context.Context, id string) (Artifact, error) {
	var value Artifact
	err := s.pool.QueryRow(ctx, `SELECT id, sha256, size_bytes, content_type, uploader, created_at, storage_key FROM artifacts WHERE id = $1`, id).
		Scan(&value.ID, &value.SHA256, &value.SizeBytes, &value.ContentType, &value.Uploader, &value.CreatedAt, &value.StorageKey)
	if err != nil {
		return Artifact{}, fmt.Errorf("get artifact: %w", err)
	}
	return value, nil
}
