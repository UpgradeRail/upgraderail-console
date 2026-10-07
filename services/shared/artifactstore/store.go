package artifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// Store is the storage abstraction shared by the API (Put) and the worker
// (Open). Implementations are content-addressed: the returned key is derived
// from the SHA-256 of the bytes the store actually received.
type Store interface {
	// Put reads input once, enforcing the configured size limit, and returns
	// the server-computed hash and the storage key.
	Put(ctx context.Context, input io.Reader) (Stored, error)
	// Open returns a reader for a key previously returned by Put. The caller
	// must Close it. A missing object wraps fs.ErrNotExist.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// VerifyHash re-reads a stored artifact and recomputes its SHA-256,
// returning an error if it no longer matches the expected hash. It is used
// where a caller wants to confirm the stored bytes still match what was
// recorded at upload time, rather than trusting the recorded hash.
func VerifyHash(ctx context.Context, store Store, key, expectedSHA256 string) error {
	reader, err := store.Open(ctx, key)
	if err != nil {
		return err
	}
	defer reader.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return fmt.Errorf("read artifact for verification: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expectedSHA256 {
		return fmt.Errorf("artifact hash mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}
