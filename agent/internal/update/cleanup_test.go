package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.bak")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if !removeFile(path) {
		t.Fatal("removeFile(existing) = false, want true")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("backup still exists: %v", err)
	}
	if removeFile(path) {
		t.Fatal("removeFile(missing) = true, want false")
	}
}
