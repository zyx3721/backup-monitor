package collect

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/discover"
)

func TestNewestArtifactAndMessage(t *testing.T) {
	directory := t.TempDir()
	oldPath := filepath.Join(directory, "old.bak")
	newPath := filepath.Join(directory, "new.bak")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now.Add(-1*time.Hour), now.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	profile := api.DefaultProfileConfig()
	profile.ArtifactCheck = true
	profile.ArtifactMaxDepth = 2
	collector := &Collector{}
	task := discover.Task{
		Name: "db",
		Key:  "db-key",
		Config: map[string]any{
			"artifactPath": directory,
		},
	}
	artifact := collector.findArtifact(task, profile)
	if artifact == nil {
		t.Fatal("expected artifact")
	}
	if artifact.Path != newPath {
		t.Fatalf("artifact path = %q, want %q", artifact.Path, newPath)
	}
	if artifact.Size != int64(len("new-data")) {
		t.Errorf("artifact size = %d", artifact.Size)
	}

	execution := artifactExecution(artifact, task, time.Time{}, false, profile)
	if execution.State != "success" {
		t.Errorf("state = %q", execution.State)
	}
	if execution.Message != "检测到备份产物："+newPath {
		t.Errorf("message = %q", execution.Message)
	}
	if execution.ArtifactPath != newPath {
		t.Errorf("artifact path = %q", execution.ArtifactPath)
	}
	if execution.ProcessedSize != "8 B" {
		t.Errorf("size = %q", execution.ProcessedSize)
	}
}
