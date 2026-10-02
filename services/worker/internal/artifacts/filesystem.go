// Package artifacts provides immutable, hash-addressed artifact storage.
package artifacts

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

var ErrTooLarge = errors.New("artifact exceeds configured size limit")

type Stored struct {
	SHA256, Key, ContentType string
	Size                     int64
}

type Filesystem struct {
	Directory string
	MaxBytes  int64
}

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
