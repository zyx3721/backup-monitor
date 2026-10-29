package service_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/settings"
)

func newTestService(t *testing.T) *service.BackupService {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "backup-monitor.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("enable wal: %v", err)
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure settings schema: %v", err)
	}
	if err := repository.EnsureBackupSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	svc := service.NewBackupService(repository.NewBackupRepository(db))
	svc.SetDataDir(t.TempDir())
	return svc
}

func sampleReport() domain.IngestReport {
	base := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	return domain.IngestReport{
		Host: domain.IngestHost{Hostname: "db-01", NodeID: "node-1", IP: "10.0.0.11", OS: "linux", Role: "database", AgentVersion: "1.0.0"},
		Tasks: []domain.IngestTask{{
			Name: "db-backup", Key: "cron:root:0 23 * * *", Type: "shell", Schedule: "每天 23:00", ScheduleCron: "0 23 * * *",
			Config: map[string]any{"source": "crontab", "script": "/opt/backup/db.sh"},
		}},
		Executions: []domain.IngestExecution{{
			TaskName: "db-backup", SessionID: "session-1", State: "success", Trigger: "scheduled",
			CreationTime: base.Format(time.RFC3339), StartTime: base.Add(time.Minute).Format(time.RFC3339),
			EndTime:         base.Add(8 * time.Minute).Format(time.RFC3339),
			TransferredSize: "1.1 GB", Message: "状态文件：/data/backup/db.status；产物 /data/backup/db.tar.gz（1.1 GB）",
		}},
	}
}

func TestResolveHostTaskAlertsLeavesResolvedAtEmpty(t *testing.T) {
	svc := newTestService(t)
	db := svc.Repo().DB()
	if _, err := db.Exec(
		`INSERT INTO alerts(id,host_id,host_name,type,level,status,created_at,message)
		 VALUES('alert-1','host-1','db-01','not_run','critical','active',?,'任务未执行')`,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	svc.ResolveHostTaskAlerts("host-1", []string{"not_run"})

	var status string
	var resolvedAt sql.NullString
	if err := db.QueryRow(`SELECT status,resolved_at FROM alerts WHERE id='alert-1'`).Scan(&status, &resolvedAt); err != nil {
		t.Fatalf("query alert: %v", err)
	}
	if status != "resolved" {
		t.Fatalf("status = %q, want resolved", status)
	}
	if resolvedAt.Valid {
		t.Fatalf("resolved_at = %q, want NULL", resolvedAt.String)
	}
}

func TestIngestCreatesHostTaskAndExecution(t *testing.T) {
	svc := newTestService(t)
	result, err := svc.Ingest(t.Context(), sampleReport())
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if result.HostID == "" || result.NewExecutions != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	hosts, err := svc.Repo().ListHosts()
	if err != nil || len(hosts) != 1 {
		t.Fatalf("hosts = %d, err = %v", len(hosts), err)
	}
	if hosts[0].Hostname != "db-01" || hosts[0].TaskCount != 1 || hosts[0].Status != "normal" {
		t.Fatalf("unexpected host: %+v", hosts[0])
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	if tasks[0].Status != "normal" || tasks[0].Type != "shell" {
		t.Fatalf("unexpected task: %+v", tasks[0])
	}
	executions, err := svc.Repo().ListExecutionsByTask(tasks[0].ID)
	if err != nil || len(executions) != 1 {
		t.Fatalf("executions = %d, err = %v", len(executions), err)
	}
	if executions[0].ArtifactPath != "/data/backup/db.tar.gz" {
		t.Fatalf("artifact = %q", executions[0].ArtifactPath)
	}
}

func TestIngestIsIdempotentPerSession(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Ingest(t.Context(), sampleReport()); err != nil {
		t.Fatal(err)
	}
	report := sampleReport()
	report.Executions[0].State = "failed"
	report.Executions[0].Message = "退出码 1"
	again, err := svc.Ingest(t.Context(), report)
	if err != nil {
		t.Fatal(err)
	}
	if again.NewExecutions != 0 {
		t.Fatalf("expected no new execution, got %d", again.NewExecutions)
	}
	tasks, _ := svc.Repo().ListTasks()
	executions, _ := svc.Repo().ListExecutionsByTask(tasks[0].ID)
	if len(executions) != 1 || executions[0].State != "failed" {
		t.Fatalf("unexpected executions: %+v", executions)
	}
	alerts, _ := svc.Repo().ListAlerts()
	if len(alerts) != 1 || alerts[0].Type != "backup_failed" || alerts[0].Status != "active" {
		t.Fatalf("unexpected alerts: %+v", alerts)
	}
}

func TestSuccessResolvesFailureAlert(t *testing.T) {
	svc := newTestService(t)
	failed := sampleReport()
	failed.Executions[0].State = "failed"
	if _, err := svc.Ingest(t.Context(), failed); err != nil {
		t.Fatal(err)
	}
	success := sampleReport()
	success.Executions[0].SessionID = "session-2"
	successTime := time.Now().UTC().Add(-time.Minute)
	success.Executions[0].CreationTime = successTime.Format(time.RFC3339)
	success.Executions[0].StartTime = successTime.Add(-time.Minute).Format(time.RFC3339)
	success.Executions[0].EndTime = successTime.Format(time.RFC3339)
	if _, err := svc.Ingest(t.Context(), success); err != nil {
		t.Fatal(err)
	}
	alerts, _ := svc.Repo().ListAlerts()
	if len(alerts) != 1 || alerts[0].Status != "resolved" {
		t.Fatalf("expected resolved alert, got %+v", alerts)
	}
}

func TestTaskWithoutExecutionIsNotRun(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	report.Executions = nil
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatal(err)
	}
	tasks, _ := svc.Repo().ListTasks()
	status := svc.RecomputeTaskStatus(tasks[0].ID, time.Date(2026, 10, 21, 23, 45, 0, 0, time.Local))
	if status != string(domain.TaskNotRun) {
		t.Fatalf("status = %s, want not_run", status)
	}
}

func TestHostStatusThresholds(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	report.Host.IP = "10.0.0.12"
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatal(err)
	}
	hosts, _ := svc.Repo().ListHosts()
	now := time.Now().UTC()
	if got := svc.RecomputeHostStatus(hosts[0].ID, now.Add(2*time.Minute)); got != "normal" {
		t.Fatalf("2min status = %s", got)
	}
	if got := svc.RecomputeHostStatus(hosts[0].ID, now.Add(10*time.Minute)); got != "offline" {
		t.Fatalf("10min status = %s", got)
	}
}

func TestSettingsDefaultsSeededAndPersisted(t *testing.T) {
	svc := newTestService(t)
	defaults := domain.DefaultAppSettings()
	if got := svc.Settings(); got != defaults {
		t.Fatalf("seeded settings = %+v, want %+v", got, defaults)
	}
	next := defaults
	next.ReportTimeoutMinutes = 10
	next.OfflineTimeoutMinutes = 30
	next.RetentionDays = 120
	next.Agent.ReportIntervalSec = 120
	if err := svc.SaveSettings(next); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if got := svc.Settings(); got != next {
		t.Fatalf("reloaded settings = %+v, want %+v", got, next)
	}
}

func TestSaveSettingsRejectsOfflineBelowReport(t *testing.T) {
	svc := newTestService(t)
	next := domain.DefaultAppSettings()
	next.ReportTimeoutMinutes = 3
	next.OfflineTimeoutMinutes = 2
	err := svc.SaveSettings(next)
	if err == nil || err.Error() != "主机离线阈值不能小于主机上报超时" {
		t.Fatalf("save error = %v", err)
	}
	if got := svc.Settings(); got.OfflineTimeoutMinutes < got.ReportTimeoutMinutes {
		t.Fatalf("invalid settings persisted: %+v", got)
	}
}

func TestIngestAdoptsLegacyLabelTask(t *testing.T) {
	svc := newTestService(t)
	legacy := sampleReport()
	legacy.Tasks[0].Name = "rsync-mys-app key=rsync-mys-file status=/var/log/backup/rsync-mys-file.status"
	legacy.Tasks[0].Key = "cron:root:10 0 * * *"
	legacy.Tasks[0].Schedule = "每天 00:10"
	legacy.Tasks[0].ScheduleCron = "10 0 * * *"
	legacy.Executions = nil
	if _, err := svc.Ingest(t.Context(), legacy); err != nil {
		t.Fatalf("ingest legacy: %v", err)
	}

	clean := sampleReport()
	clean.Tasks[0].Name = "rsync-mys-app"
	clean.Tasks[0].Key = "rsync-mys-file"
	clean.Tasks[0].Schedule = "每天 00:10"
	clean.Tasks[0].ScheduleCron = "10 0 * * *"
	clean.Tasks[0].Config = map[string]any{
		"source":     "crontab",
		"script":     "/root/rsync-file.sh",
		"statusFile": "/var/log/backup/rsync-mys-file.status",
	}
	clean.Executions = nil
	if _, err := svc.Ingest(t.Context(), clean); err != nil {
		t.Fatalf("ingest clean: %v", err)
	}

	tasks, err := svc.Repo().ListTasks()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1: %+v", len(tasks), tasks)
	}
	if tasks[0].Name != "rsync-mys-app" || tasks[0].TaskKey != "rsync-mys-file" {
		t.Fatalf("adopted task = %+v", tasks[0])
	}
}

func TestIngestUsesExecutionStartTime(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	want, err := time.Parse(time.RFC3339, report.Executions[0].StartTime)
	if err != nil {
		t.Fatalf("parse start time: %v", err)
	}
	got, err := time.Parse(time.RFC3339, tasks[0].LastExecutionTime)
	if err != nil {
		t.Fatalf("last execution time = %q: %v", tasks[0].LastExecutionTime, err)
	}
	if !got.Equal(want) {
		t.Fatalf("last execution = %s, want %s", got, want)
	}
}

func TestIngestUnknownOSLeavesPlatformEmpty(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	report.Host.OS = ""
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	hosts, err := svc.Repo().ListHosts()
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("host count = %d, want 1", len(hosts))
	}
	if hosts[0].Platform != "" {
		t.Fatalf("platform = %q, want empty", hosts[0].Platform)
	}
}

func TestIngestWindowsOSResolvesPlatform(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	report.Host.OS = "Windows Server 2019"
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	hosts, err := svc.Repo().ListHosts()
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Platform != "windows" {
		t.Fatalf("hosts = %+v, want windows platform", hosts)
	}
}

func TestIngestKeepsExistingHostRole(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	report.Host.Role = "web"
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	hosts, err := svc.Repo().ListHosts()
	if err != nil || len(hosts) != 1 {
		t.Fatalf("hosts = %d, err = %v", len(hosts), err)
	}
	if hosts[0].Role != "database" {
		t.Fatalf("role = %q, want database", hosts[0].Role)
	}
}
