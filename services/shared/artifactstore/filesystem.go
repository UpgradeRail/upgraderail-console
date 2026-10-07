// Package artifactstore provides immutable, hash-addressed artifact storage
// shared by the API (which accepts uploads) and the worker (which reads
// uploaded artifacts to run the Engine). Both services import this same
// module so there is exactly one storage abstraction, not one per service.
package artifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrTooLarge is returned when an uploaded artifact exceeds MaxBytes.
	ErrTooLarge = errors.New("artifact exceeds configured size limit")
	// ErrEmpty is returned when an uploaded artifact contains zero bytes.
	ErrEmpty = errors.New("artifact is empty")
)

// Stored describes an artifact after it has been committed to storage.
type Stored struct {
	SHA256, Key, ContentType string
	Size                     int64
}

// Filesystem stores artifacts as content-addressed files under Directory,
// sharded by the first two hex characters of the SHA-256 hash.
type Filesystem struct {
	Directory string
	MaxBytes  int64
}

// Put streams input to a temporary file, hashing as it goes, then commits
// the file to its content-addressed path. The SHA-256 returned is always
// computed server-side from the bytes actually written; callers must never
// substitute a client-supplied hash for it.
func (s Filesystem) Put(ctx context.Context, input io.Reader) (Stored, error) {
	if !filepath.IsAbs(s.Directory) {
		return Stored{}, errors.New("artifact directory must be absolute")
	}
	if s.MaxBytes < 1 {
		return Stored{}, errors.New("maximum artifact size must be positive")
	}
	if err := os.MkdirAll(s.Directory, 0700); err != nil {
		return Stored{}, fmt.Errorf("create artifact directory: %w", err)
	}
	temporary, err := os.CreateTemp(s.Directory, ".upload-*")
	if err != nil {
		return Stored{}, fmt.Errorf("create temporary artifact: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	defer temporary.Close()

	hash := sha256.New()
	reader := &contextReader{ctx: ctx, reader: io.LimitReader(input, s.MaxBytes+1)}
	size, err := io.Copy(io.MultiWriter(temporary, hash), reader)
	if err != nil {
		return Stored{}, fmt.Errorf("write artifact: %w", err)
	}
	if size > s.MaxBytes {
		return Stored{}, ErrTooLarge
	}
	if size == 0 {
		return Stored{}, ErrEmpty
	}
	if err := temporary.Chmod(0600); err != nil {
		return Stored{}, fmt.Errorf("protect artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Stored{}, fmt.Errorf("close artifact: %w", err)
	}
	encodedHash := hex.EncodeToString(hash.Sum(nil))
	key := filepath.Join(encodedHash[:2], encodedHash)
	path := filepath.Join(s.Directory, key)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return Stored{}, fmt.Errorf("create artifact shard: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return Stored{}, fmt.Errorf("commit artifact: %w", err)
	}
	return Stored{SHA256: encodedHash, Key: key, Size: size, ContentType: "application/wasm"}, nil
}

// Open returns a reader for a previously stored artifact identified by its
// storage key (as returned in Stored.Key). The caller must Close it.
func (s Filesystem) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", err)
	}
	return file, nil
}

func (s Filesystem) path(key string) (string, error) {
	if !filepath.IsAbs(s.Directory) {
		return "", errors.New("artifact directory must be absolute")
	}
	cleaned := filepath.Clean(key)
	if filepath.IsAbs(cleaned) || cleaned == ".." || cleaned == "." || hasDotDotPrefix(cleaned) {
		return "", errors.New("artifact storage key is invalid")
	}
	return filepath.Join(s.Directory, cleaned), nil
}

func hasDotDotPrefix(cleaned string) bool {
	return len(cleaned) >= 3 && cleaned[0] == '.' && cleaned[1] == '.' && os.IsPathSeparator(cleaned[2])
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(value []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(value)
}
