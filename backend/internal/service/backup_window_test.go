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

func TestTaskMatchWindowOnlyLimitsPlanEnd(t *testing.T) {
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	cases := []struct {
		name    string
		created *time.Time
		now     time.Time
		want    string
	}{
		{"计划时刻执行", ptrTime(plan), plan.Add(5 * time.Minute), "normal"},
		{"提前 4 分钟（时钟容差内）", ptrTime(plan.Add(-4 * time.Minute)), plan.Add(5 * time.Minute), "normal"},
		{"提前 30 分钟（超出时钟容差）", ptrTime(plan.Add(-30 * time.Minute)), plan.Add(5 * time.Minute), "not_run"},
		{"晚 40 分钟（窗口内）", ptrTime(plan.Add(40 * time.Minute)), plan.Add(45 * time.Minute), "normal"},
		{"晚 70 分钟（超出窗口）", ptrTime(plan.Add(70 * time.Minute)), plan.Add(80 * time.Minute), "not_run"},
		{"无执行记录", nil, plan.Add(45 * time.Minute), "not_run"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			svc := newTestService(t)
			settings := domain.DefaultAppSettings()
			settings.ScheduleMatchWindowMinutes = 60
			settings.NotRunGraceMinutes = 1
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

func TestManualExecutionDoesNotSatisfyPlan(t *testing.T) {
	svc := newTestService(t)
	settings := domain.DefaultAppSettings()
	settings.ScheduleMatchWindowMinutes = 60
	settings.NotRunGraceMinutes = 1
	if err := svc.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, nil)
	report.Executions = []domain.IngestExecution{{
		TaskName: "window-task", SessionID: "manual-1", State: "success", Trigger: "manual",
		CreationTime: plan.Add(30 * time.Minute).UTC().Format(time.RFC3339),
		StartTime:    plan.Add(31 * time.Minute).UTC().Format(time.RFC3339),
		EndTime:      plan.Add(32 * time.Minute).UTC().Format(time.RFC3339),
	}}
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	if got := svc.RecomputeTaskStatus(tasks[0].ID, plan.Add(40*time.Minute)); got != "not_run" {
		t.Fatalf("status = %s, want not_run", got)
	}
}

func TestScheduledExecutionSatisfiesPlanAlongsideManual(t *testing.T) {
	svc := newTestService(t)
	settings := domain.DefaultAppSettings()
	settings.ScheduleMatchWindowMinutes = 60
	settings.NotRunGraceMinutes = 1
	if err := svc.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, nil)
	report.Executions = []domain.IngestExecution{
		{
			TaskName: "window-task", SessionID: "sched-1", State: "success", Trigger: "scheduled",
			CreationTime: plan.Add(5 * time.Minute).UTC().Format(time.RFC3339),
			StartTime:    plan.Add(6 * time.Minute).UTC().Format(time.RFC3339),
			EndTime:      plan.Add(7 * time.Minute).UTC().Format(time.RFC3339),
		},
		{
			TaskName: "window-task", SessionID: "manual-1", State: "success", Trigger: "manual",
			CreationTime: plan.Add(30 * time.Minute).UTC().Format(time.RFC3339),
			StartTime:    plan.Add(31 * time.Minute).UTC().Format(time.RFC3339),
			EndTime:      plan.Add(32 * time.Minute).UTC().Format(time.RFC3339),
		},
	}
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	if got := svc.RecomputeTaskStatus(tasks[0].ID, plan.Add(40*time.Minute)); got != "normal" {
		t.Fatalf("status = %s, want normal", got)
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

func TestNextPlanExecutionIngestedEarlyKeepsPreviousPlan(t *testing.T) {
	svc := newTestService(t)
	settings := domain.DefaultAppSettings()
	settings.ScheduleMatchWindowMinutes = 60
	settings.NotRunGraceMinutes = 30
	if err := svc.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	next := plan.Add(24 * time.Hour)
	report := windowReport(plan, ptrTime(plan.Add(3*time.Second)))
	report.Executions = append(report.Executions, domain.IngestExecution{
		TaskName: "window-task", SessionID: "next-plan", State: "success", Trigger: "scheduled",
		CreationTime: next.Add(3 * time.Second).UTC().Format(time.RFC3339),
		StartTime:    next.Add(3 * time.Second).UTC().Format(time.RFC3339),
		EndTime:      next.Add(53 * time.Second).UTC().Format(time.RFC3339),
	})
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	evaluation := svc.EvaluateTask(tasks[0].ID, next.Add(-time.Minute))
	if !evaluation.Plan.Equal(plan) {
		t.Fatalf("plan = %s, want %s", evaluation.Plan, plan)
	}
	if evaluation.Status != "normal" {
		t.Fatalf("status = %s, want normal（下一周期记录提前入库不应污染本次判定）", evaluation.Status)
	}
}

func TestLaterScheduledExecutionKeepsWindowRecord(t *testing.T) {
	svc := newTestService(t)
	settings := domain.DefaultAppSettings()
	settings.ScheduleMatchWindowMinutes = 60
	settings.NotRunGraceMinutes = 30
	if err := svc.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, ptrTime(plan.Add(3*time.Second)))
	report.Executions = append(report.Executions, domain.IngestExecution{
		TaskName: "window-task", SessionID: "late-retry", State: "success", Trigger: "scheduled",
		CreationTime: plan.Add(5 * time.Hour).UTC().Format(time.RFC3339),
		StartTime:    plan.Add(5 * time.Hour).UTC().Format(time.RFC3339),
		EndTime:      plan.Add(5*time.Hour + time.Minute).UTC().Format(time.RFC3339),
	})
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	evaluation := svc.EvaluateTask(tasks[0].ID, plan.Add(6*time.Hour))
	if evaluation.Status != "normal" {
		t.Fatalf("status = %s, want normal（窗口内成功记录不应被更晚记录顶掉）", evaluation.Status)
	}
}

func TestLateOnlyExecutionStillReportsLate(t *testing.T) {
	svc := newTestService(t)
	settings := domain.DefaultAppSettings()
	settings.ScheduleMatchWindowMinutes = 60
	settings.NotRunGraceMinutes = 30
	if err := svc.SaveSettings(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, nil)
	report.Executions = []domain.IngestExecution{{
		TaskName: "window-task", SessionID: "late-only", State: "success", Trigger: "scheduled",
		CreationTime: plan.Add(5 * time.Hour).UTC().Format(time.RFC3339),
		StartTime:    plan.Add(5 * time.Hour).UTC().Format(time.RFC3339),
		EndTime:      plan.Add(5*time.Hour + time.Minute).UTC().Format(time.RFC3339),
	}}
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	evaluation := svc.EvaluateTask(tasks[0].ID, plan.Add(6*time.Hour))
	if evaluation.Status != "not_run" || evaluation.Reason != "late" {
		t.Fatalf("status = %s, reason = %s, want not_run/late", evaluation.Status, evaluation.Reason)
	}
}

func activeNotRunCount(t *testing.T, svc *service.BackupService, taskID string) int {
	t.Helper()
	var count int
	if err := svc.Repo().DB().QueryRow(`SELECT COUNT(*) FROM alerts WHERE type='not_run' AND task_id=? AND status='active'`, taskID).Scan(&count); err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	return count
}
