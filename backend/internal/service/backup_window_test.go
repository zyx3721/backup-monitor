package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
)

func windowReport(plan time.Time, created *time.Time) domain.IngestReport {
	cron := fmt.Sprintf("%d %d * * *", plan.Minute(), plan.Hour())
	report := domain.IngestReport{
		Host: domain.IngestHost{Hostname: "window-01", NodeID: "node-window", IP: "10.0.0.60", OS: "linux", Role: "database", AgentVersion: "1.0.0"},
		Tasks: []domain.IngestTask{{
			Name: "window-task", Key: "cron:root:" + cron, Type: "shell",
			Schedule: "每天 " + plan.Format("15:04"), ScheduleCron: cron,
			Config: map[string]any{"source": "crontab"},
		}},
	}
	if created != nil {
		start := created.Add(time.Minute)
		report.Executions = []domain.IngestExecution{{
			TaskName: "window-task", SessionID: "window-session", State: "success", Trigger: "scheduled",
			CreationTime: created.UTC().Format(time.RFC3339),
			StartTime:    start.UTC().Format(time.RFC3339),
			EndTime:      start.Add(time.Minute).UTC().Format(time.RFC3339),
		}}
	}
	return report
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestTaskMatchWindowIsSymmetric(t *testing.T) {
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	cases := []struct {
		name    string
		created *time.Time
		now     time.Time
		window  int
		want    string
	}{
		{"早 30 分钟窗口 60", ptrTime(plan.Add(-30 * time.Minute)), plan.Add(45 * time.Minute), 60, "normal"},
		{"早 30 分钟窗口 10", ptrTime(plan.Add(-30 * time.Minute)), plan.Add(45 * time.Minute), 10, "not_run"},
		{"晚 40 分钟窗口 60", ptrTime(plan.Add(40 * time.Minute)), plan.Add(45 * time.Minute), 60, "normal"},
		{"晚 40 分钟窗口 10", ptrTime(plan.Add(40 * time.Minute)), plan.Add(45 * time.Minute), 10, "not_run"},
		{"晚 4 小时窗口 60", ptrTime(plan.Add(4 * time.Hour)), plan.Add(4*time.Hour + 10*time.Minute), 60, "not_run"},
		{"无执行记录", nil, plan.Add(45 * time.Minute), 60, "not_run"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			svc := newTestService(t)
			settings := domain.DefaultAppSettings()
			settings.ScheduleMatchWindowMinutes = item.window
			if err := svc.SaveSettings(settings); err != nil {
				t.Fatalf("save settings: %v", err)
			}
			if _, err := svc.Ingest(t.Context(), windowReport(plan, item.created)); err != nil {
				t.Fatalf("ingest: %v", err)
			}
			tasks, err := svc.Repo().ListTasks()
			if err != nil || len(tasks) != 1 {
				t.Fatalf("tasks = %d, err = %v", len(tasks), err)
			}
			if got := svc.RecomputeTaskStatus(tasks[0].ID, item.now); got != item.want {
				t.Fatalf("status = %s, want %s", got, item.want)
			}
		})
	}
}

func TestNotRunAlertResolvesOnlyForOwnPlan(t *testing.T) {
	svc := newTestService(t)
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	if _, err := svc.Ingest(t.Context(), windowReport(plan, nil)); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	if _, err := svc.Repo().DB().Exec(`DELETE FROM alerts`); err != nil {
		t.Fatalf("clear alerts: %v", err)
	}
	message := fmt.Sprintf("任务 window-task 计划于 %s 执行，但未检测到对应执行记录", plan.Format("2006-01-02 15:04"))
	if _, err := svc.Repo().DB().Exec(`INSERT INTO alerts(id,host_id,task_id,host_name,task_name,type,level,status,created_at,message)
		VALUES('alert-plan-1',?,?,'window-01','window-task','not_run','critical','active',?,?)`,
		tasks[0].HostID, tasks[0].ID, time.Now().UTC().Format(time.RFC3339), message); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	svc.ResolveNotRunAlertsForPlan(tasks[0].ID, plan.Add(24*time.Hour))
	if got := activeNotRunCount(t, svc, tasks[0].ID); got != 1 {
		t.Fatalf("active alerts after next plan = %d, want 1", got)
	}

	svc.ResolveNotRunAlertsForPlan(tasks[0].ID, plan)
	if got := activeNotRunCount(t, svc, tasks[0].ID); got != 0 {
		t.Fatalf("active alerts after own plan = %d, want 0", got)
	}
}

func TestSchedulerNotRunAlertCarriesPlanDate(t *testing.T) {
	svc := newTestService(t)
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	if _, err := svc.Ingest(t.Context(), windowReport(plan, nil)); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if _, err := svc.Repo().DB().Exec(`DELETE FROM alerts`); err != nil {
		t.Fatalf("clear alerts: %v", err)
	}
	svc.RunScheduler(plan.Add(31 * time.Minute))
	alerts, err := svc.Repo().ListAlerts()
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	want := plan.Format("2006-01-02 15:04")
	for _, alert := range alerts {
		if alert.Type == "not_run" && strings.Contains(alert.Message, want) {
			return
		}
	}
	t.Fatalf("alerts = %+v, want not_run containing %q", alerts, want)
}

func activeNotRunCount(t *testing.T, svc *service.BackupService, taskID string) int {
	t.Helper()
	var count int
	if err := svc.Repo().DB().QueryRow(`SELECT COUNT(*) FROM alerts WHERE type='not_run' AND task_id=? AND status='active'`, taskID).Scan(&count); err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	return count
}
