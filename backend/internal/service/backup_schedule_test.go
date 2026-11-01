package service_test

import (
	"testing"
	"time"

	"backup-monitor-backend/internal/domain"
)

func TestDecorateTasksScheduleState(t *testing.T) {
	svc := newTestService(t)
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, nil)
	report.Tasks = []domain.IngestTask{
		{Name: "monthly", Key: "cron:root:monthly", Type: "shell", Schedule: "每月 15 日 06:00", Config: map[string]any{"source": "crontab"}},
		{Name: "reboot", Key: "cron:root:reboot", Type: "shell", Schedule: "开机时", Config: map[string]any{"source": "schtasks"}},
		{Name: "disabled", Key: "cron:root:disabled", Type: "shell", Schedule: "未启用", Config: map[string]any{"source": "crontab"}},
	}
	report.Executions = nil
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	byName := map[string]domain.BackupTask{}
	for _, task := range svc.DecorateTasks(tasks) {
		byName[task.Name] = task
	}
	if !byName["monthly"].ScheduleParsed || byName["monthly"].ScheduleDisabled {
		t.Fatalf("monthly schedule = %+v", byName["monthly"])
	}
	if byName["reboot"].ScheduleParsed || byName["reboot"].ScheduleDisabled {
		t.Fatalf("reboot schedule = %+v", byName["reboot"])
	}
	if byName["disabled"].ScheduleParsed || !byName["disabled"].ScheduleDisabled {
		t.Fatalf("disabled schedule = %+v", byName["disabled"])
	}
}

func TestManualServerScheduleSurvivesIngest(t *testing.T) {
	svc := newTestService(t)
	plan := time.Date(2026, 10, 21, 23, 0, 0, 0, time.Local)
	report := windowReport(plan, nil)
	report.Tasks[0].Schedule = "开机时"
	report.Tasks[0].ScheduleCron = ""
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	taskID := tasks[0].ID
	if err := svc.Repo().UpdateTaskSchedule(taskID, "0 23 * * *", true); err != nil {
		t.Fatalf("update schedule: %v", err)
	}

	again := windowReport(plan, nil)
	again.Tasks[0].Schedule = "登录时"
	again.Tasks[0].ScheduleCron = ""
	if _, err := svc.Ingest(t.Context(), again); err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	task, err := svc.Repo().GetTask(taskID)
	if err != nil || task == nil {
		t.Fatalf("get task: %v", err)
	}
	if task.ScheduleCron != "0 23 * * *" || !task.ScheduleLocked {
		t.Fatalf("locked schedule = %q locked=%v", task.ScheduleCron, task.ScheduleLocked)
	}
	decoded := svc.DecorateTasks([]domain.BackupTask{*task})[0]
	if !decoded.ScheduleParsed || decoded.ScheduleDisabled {
		t.Fatalf("decoded task = %+v", decoded)
	}
}
