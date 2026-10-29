package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFailureBlockedUsesVersionAndBackoff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.update-state")
	now := time.Now()

	if failureBlocked(path, "1.0.32", now) {
		t.Fatal("缺少状态文件时不应退避")
	}
	if err := writeFailure(path, "1.0.32", now.Add(-time.Minute)); err != nil {
		t.Fatalf("writeFailure: %v", err)
	}
	if !failureBlocked(path, "1.0.32", now) {
		t.Fatal("同版本失败后应退避")
	}
	if failureBlocked(path, "1.0.33", now) {
		t.Fatal("其它版本不应退避")
	}
	if failureBlocked(path, "1.0.32", now.Add(FailureBackoff+time.Minute)) {
		t.Fatal("超过退避时长后应允许重试")
	}
}

func TestFailureBlockedIgnoresBrokenState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.update-state")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("write broken state: %v", err)
	}
	if failureBlocked(path, "1.0.32", time.Now()) {
		t.Fatal("状态文件损坏时不应退避")
	}
}
