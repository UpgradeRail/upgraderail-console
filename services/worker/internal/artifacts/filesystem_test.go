package artifacts

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemStoreUsesContentHash(t *testing.T) {
	directory := t.TempDir()
	stored, err := (Filesystem{Directory: directory, MaxBytes: 16}).Put(context.Background(), bytes.NewReader([]byte("wasm")))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.SHA256) != 64 {
		t.Fatalf("expected SHA-256, got %s", stored.SHA256)
	}
	if _, err := os.Stat(filepath.Join(directory, stored.Key)); err != nil {
		t.Fatal(err)
	}
}

func TestFilesystemStoreRejectsOversizedArtifact(t *testing.T) {
	_, err := (Filesystem{Directory: t.TempDir(), MaxBytes: 3}).Put(context.Background(), bytes.NewReader([]byte("wasm")))
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
}
