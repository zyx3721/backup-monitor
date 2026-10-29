//go:build windows

package discover

import (
	"bytes"
	"context"
	"testing"
)

func TestRunSchtasks(t *testing.T) {
	output, err := runSchtasks(context.Background())
	if err != nil {
		t.Fatalf("runSchtasks: %v", err)
	}
	if !bytes.Contains(output, []byte("TaskName")) {
		t.Fatalf("unexpected schtasks output: %q", string(output))
	}
	tasks, err := ParseSchtasksCSV(output, defaultFilter())
	if err != nil {
		t.Fatalf("ParseSchtasksCSV: %v", err)
	}
	t.Logf("schtasks tasks matched: %d", len(tasks))
}
