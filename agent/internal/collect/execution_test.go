package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/cron"
	"backup-monitor-agent/internal/discover"
	"backup-monitor-agent/internal/statusfile"
	"backup-monitor-agent/internal/veeam"
)

func TestProcExecution(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	scheduled := time.Date(2026, 10, 21, 23, 0, 0, 0, zone)
	task := discover.Task{Name: "db", ProcKeyword: " mysqldump "}

	execution, ok := procExecution(task, scheduled)
	if !ok {
		t.Fatal("expected a proc execution")
	}
	if execution.State != "running" {
		t.Errorf("state = %q", execution.State)
	}
	if execution.Trigger != "scheduled" {
		t.Errorf("trigger = %q", execution.Trigger)
	}
	if want := fmt.Sprintf("{proc:db:%d}", scheduled.UnixMilli()); execution.SessionID != want {
		t.Errorf("sessionId = %q, want %q", execution.SessionID, want)
	}
	if !strings.Contains(execution.Message, "mysqldump") {
		t.Errorf("message = %q", execution.Message)
	}
	if execution.IsLastAttempt == nil || !*execution.IsLastAttempt {
		t.Error("proc execution must be last attempt")
	}
	if _, ok := procExecution(task, time.Time{}); ok {
		t.Error("zero scheduled time must not produce an execution")
	}
}

func TestStateFromExit(t *testing.T) {
	cases := map[int]string{
		0:      "success",
		267009: "running",
		267011: "",
		267012: "",
		267014: "failed",
		267015: "",
		1:      "failed",
		-1:     "failed",
	}
	for code, want := range cases {
		if got := stateFromExit(code); got != want {
			t.Errorf("stateFromExit(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestTriggerFor(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	scheduled := time.Date(2026, 10, 20, 23, 0, 0, 0, zone)
	if got := triggerFor(scheduled.Add(5*time.Minute), scheduled, true, 30); got != "scheduled" {
		t.Errorf("near schedule = %q", got)
	}
	if got := triggerFor(scheduled.Add(40*time.Minute), scheduled, true, 30); got != "manual" {
		t.Errorf("deviation = %q", got)
	}
	if got := triggerFor(scheduled, time.Time{}, false, 30); got != "scheduled" {
		t.Errorf("missing schedule = %q", got)
	}
}

func TestExecutionFromStatus(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "db.status")
	content := "start=2026-10-20T23:00:01+08:00\nend=2026-10-20T23:10:00+08:00\nsize=2 GB\nexit=0\nstatus=success\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := statusfile.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	zone := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 10, 21, 10, 0, 0, 0, zone)
	scheduled, ok := cron.GetLastScheduledTime("0 23 * * *", now)
	if !ok {
		t.Fatal("expected scheduled time")
	}
	task := discover.Task{
		Name:         "db",
		ScheduleCron: "0 23 * * *",
		Script:       "/opt/backup/db.sh",
	}
	execution, valid := executionFromStatus(task, path, info, now, scheduled, true, false, false, api.DefaultProfileConfig())
	if !valid {
		t.Fatal("expected a valid execution")
	}
	if execution.State != "success" {
		t.Errorf("state = %q", execution.State)
	}
	if execution.ProcessedSize != "2.0 GB" {
		t.Errorf("size = %q", execution.ProcessedSize)
	}
	if execution.Message != "状态文件："+path {
		t.Errorf("message = %q", execution.Message)
	}
	wantSessionID := fmt.Sprintf(
		"{status:db:%d}",
		time.Date(2026, 10, 20, 23, 0, 1, 0, zone).UnixMilli(),
	)
	if execution.SessionID != wantSessionID {
		t.Errorf("session ID = %q, want %q", execution.SessionID, wantSessionID)
	}
	if execution.Trigger != "scheduled" {
		t.Errorf("trigger = %q", execution.Trigger)
	}
	if execution.ExitCode == nil || *execution.ExitCode != 0 {
		t.Errorf("exit code = %#v", execution.ExitCode)
	}
}

func TestExecutionFromStatusMissingEnd(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	start := time.Date(2026, 11, 5, 23, 30, 1, 0, zone)
	path := "/var/log/backup/mongodb-backup.status"
	info := statusfile.Info{Start: start, HasStart: true}
	task := discover.Task{Name: "mongodb-backup"}
	profile := api.DefaultProfileConfig()

	cases := []struct {
		name          string
		now           time.Time
		running       bool
		procAvailable bool
		wantState     string
		wantMessage   string
		wantEnd       bool
	}{
		{"process running", start.Add(3 * time.Hour), true, false, "running", "执行中", false},
		{"process available but stopped", start.Add(time.Minute), false, true, "failed", "执行中断（缺少结束时间）（" + path + "）", false},
		{"no proc keyword within grace", start.Add(5 * time.Minute), false, false, "running", "执行中（进程探测不可用，按时间窗口判定）", false},
		{"no proc keyword beyond grace", start.Add(20 * time.Minute), false, false, "failed", "执行中断（缺少结束时间）（" + path + "）", false},
	}
	for _, tc := range cases {
		execution, ok := executionFromStatus(task, path, info, tc.now, time.Time{}, false, tc.running, tc.procAvailable, profile)
		if !ok {
			t.Fatalf("%s: expected an execution", tc.name)
		}
		if execution.State != tc.wantState {
			t.Errorf("%s: state = %q, want %q", tc.name, execution.State, tc.wantState)
		}
		if execution.Message != tc.wantMessage {
			t.Errorf("%s: message = %q, want %q", tc.name, execution.Message, tc.wantMessage)
		}
		if (execution.EndTime != "") != tc.wantEnd {
			t.Errorf("%s: endTime = %q", tc.name, execution.EndTime)
		}
	}
}

func TestExecutionFromStatusRunningStatusMissingEnd(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	start := time.Date(2026, 11, 5, 23, 30, 1, 0, zone)
	path := "/var/log/backup/mongodb-backup.status"
	info := statusfile.Info{Start: start, HasStart: true, Status: "running"}
	task := discover.Task{Name: "mongodb-backup"}

	execution, ok := executionFromStatus(task, path, info, start.Add(20*time.Minute), time.Time{}, false, false, false, api.DefaultProfileConfig())
	if !ok {
		t.Fatal("expected an execution")
	}
	if execution.State != "failed" {
		t.Errorf("state = %q, want failed", execution.State)
	}
	if execution.Message != "执行中断（缺少结束时间）（"+path+"）" {
		t.Errorf("message = %q", execution.Message)
	}
}

func TestNormalizeExecutionState(t *testing.T) {
	cases := map[string]string{
		"SUCCESS":    "success",
		"failed":     "failed",
		"InProgress": "running",
		"aborted":    "failed",
	}
	for input, want := range cases {
		if got := normalizeExecutionState(input); got != want {
			t.Errorf("normalizeExecutionState(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeSessionPreservesRetryFields(t *testing.T) {
	base := veeam.Session{ID: "s1", State: "success"}
	info := veeam.Session{ID: "s1", OIBID: "{oib}", Attempt: 1, HasAttempt: true, MaxAttempts: 4, HasMaxAttempts: true, IsLastAttempt: true, HasLastAttempt: true}
	merged := mergeSession(base, info)
	if merged.OIBID != "{oib}" || !merged.HasAttempt || merged.Attempt != 1 || !merged.HasMaxAttempts || merged.MaxAttempts != 4 || !merged.HasLastAttempt || !merged.IsLastAttempt {
		t.Fatalf("merged session lost retry fields: %#v", merged)
	}
}
