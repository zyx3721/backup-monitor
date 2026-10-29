package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWindowsLogStartsWithBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	logger, closeFn, err := Setup(path, 1, 1)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	logger.Printf("中文日志")
	if err := closeFn(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if runtime.GOOS == "windows" {
		if !bytes.HasPrefix(data, []byte(utf8BOM)) {
			t.Fatalf("missing UTF-8 BOM, head=% X", data[:minInt(6, len(data))])
		}
	}
	if !bytes.Contains(data, []byte("中文日志")) {
		t.Fatalf("log content lost, got %q", string(data))
	}
}

func TestAppendDoesNotRepeatBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	first, closeFirst, err := Setup(path, 1, 1)
	if err != nil {
		t.Fatalf("Setup first: %v", err)
	}
	first.Printf("first")
	if err := closeFirst(); err != nil {
		t.Fatalf("Close first: %v", err)
	}
	second, closeSecond, err := Setup(path, 1, 1)
	if err != nil {
		t.Fatalf("Setup second: %v", err)
	}
	second.Printf("second")
	if err := closeSecond(); err != nil {
		t.Fatalf("Close second: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if runtime.GOOS == "windows" && bytes.Count(data, []byte(utf8BOM)) != 1 {
		t.Fatalf("BOM repeated, count=%d", bytes.Count(data, []byte(utf8BOM)))
	}
	if !bytes.Contains(data, []byte("first")) || !bytes.Contains(data, []byte("second")) {
		t.Fatalf("append content lost, got %q", string(data))
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
