package artifactstore

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

func TestFilesystemStoreRejectsEmptyArtifact(t *testing.T) {
	_, err := (Filesystem{Directory: t.TempDir(), MaxBytes: 16}).Put(context.Background(), bytes.NewReader([]byte{}))
	if !errors.Is(err, ErrEmpty) {
		t.Fatalf("got %v", err)
	}
}

func TestFilesystemOpenAndVerifyHashRoundTrip(t *testing.T) {
	store := Filesystem{Directory: t.TempDir(), MaxBytes: 16}
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("wasm")))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyHash(stored.Key, stored.SHA256); err != nil {
		t.Fatalf("expected hash to verify, got %v", err)
	}
	if err := store.VerifyHash(stored.Key, "not-the-real-hash"); err == nil {
		t.Fatal("expected a mismatched hash to fail verification")
	}
	file, err := store.Open(stored.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "wasm" {
		t.Fatalf("unexpected stored bytes: %q", data)
	}
}

func TestFilesystemOpenRejectsPathTraversal(t *testing.T) {
	store := Filesystem{Directory: t.TempDir(), MaxBytes: 16}
	if _, err := store.Open("../../etc/passwd"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
	if _, err := store.Open("/etc/passwd"); err == nil {
		t.Fatal("expected absolute paths to be rejected")
	}
}
