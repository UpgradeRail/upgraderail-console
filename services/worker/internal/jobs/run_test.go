package jobs

import (
	"path/filepath"
	"testing"
)

func TestArtifactPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := artifactPath(root, "../secret.wasm"); err == nil {
		t.Fatal("expected traversal to fail")
	}
	path, err := artifactPath(root, "ab/hash.wasm")
	if err != nil || path != filepath.Join(root, "ab", "hash.wasm") {
		t.Fatalf("got %q, %v", path, err)
	}
}
