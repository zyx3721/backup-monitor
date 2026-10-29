package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  int
	}{
		{"1.2.33", "1.2.32", 1},
		{"1.2.32", "1.2.33", -1},
		{"1.2.33", "1.2.33", 0},
		{"v1.2.33", "1.2.33", 0},
		{"1.2", "1.2.0", 0},
		{"1.10.0", "1.2.0", 1},
		{"2.0.0", "2.0.0-beta", 1},
		{"2.0.0-beta", "2.0.0", -1},
		{"2.0.0-beta.10", "2.0.0-beta.2", 1},
		{"2.0.0-beta.2", "2.0.0-beta.10", -1},
		{"2.0.0+build.5", "2.0.0", 0},
		{"", "0.0.1", -1},
		{"0.0.1", "", 1},
		{"", "", 0},
		{"1.2.3", "1.2.3.0", 0},
		{"1.2.3.1", "1.2.3", 1},
	}
	for _, item := range cases {
		if got := CompareVersion(item.left, item.right); got != item.want {
			t.Errorf("CompareVersion(%q, %q) = %d, want %d", item.left, item.right, got, item.want)
		}
	}
}

func TestClearBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "backup-agent.bak")
	if err := os.WriteFile(target, []byte("old"), 0o444); err != nil {
		t.Fatalf("prepare backup: %v", err)
	}
	if err := clearBackup(target); err != nil {
		t.Fatalf("clearBackup returned error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("backup still exists: %v", err)
	}
}

func TestClearBackupMissing(t *testing.T) {
	if err := clearBackup(filepath.Join(t.TempDir(), "missing.bak")); err != nil {
		t.Fatalf("clearBackup on missing file returned error: %v", err)
	}
}

func TestRestoreExecutable(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "backup-agent")
	backup := filepath.Join(dir, "backup-agent.bak")
	if err := os.WriteFile(executable, []byte("new"), 0o755); err != nil {
		t.Fatalf("prepare executable: %v", err)
	}
	if err := os.WriteFile(backup, []byte("old"), 0o755); err != nil {
		t.Fatalf("prepare backup: %v", err)
	}
	if err := restoreExecutable(executable, backup); err != nil {
		t.Fatalf("restoreExecutable returned error: %v", err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatalf("read restored executable: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("restored content = %q, want old", data)
	}
}
