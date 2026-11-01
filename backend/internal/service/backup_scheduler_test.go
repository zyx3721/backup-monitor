package service_test

import (
	"testing"
	"time"

	"backup-monitor-backend/internal/domain"
)

func TestOfflineHostKeepsNotRunAlert(t *testing.T) {
	svc := newTestService(t)
	report := sampleReport()
	report.Executions = nil
	if _, err := svc.Ingest(t.Context(), report); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	tasks, err := svc.Repo().ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d, err = %v", len(tasks), err)
	}
	taskID, hostID := tasks[0].ID, tasks[0].HostID
	now := time.Now().UTC()
	for _, item := range []struct{ id, alertType string }{
		{id: "alert-not-run", alertType: "not_run"},
		{id: "alert-failed", alertType: "backup_failed"},
	} {
		if _, err := svc.Repo().DB().Exec(`INSERT INTO alerts(id,host_id,task_id,host_name,task_name,type,level,status,created_at,message)
			VALUES(?,?,?, 'db-01','db-backup',?,'critical','active',?,'测试告警')`, item.id, hostID, taskID, item.alertType,
			now.Format(time.RFC3339)); err != nil {
			t.Fatalf("insert %s: %v", item.alertType, err)
		}
	}
	if _, err := svc.Repo().DB().Exec(`UPDATE hosts SET last_report_time=? WHERE id=?`,
		now.Add(-2*time.Hour).Format(time.RFC3339), hostID); err != nil {
		t.Fatalf("expire host report: %v", err)
	}

	svc.RunScheduler(now)

	counts := map[string]int{}
	rows, err := svc.Repo().DB().Query(`SELECT type, COUNT(*) FROM alerts WHERE status='active' GROUP BY type`)
	if err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var alertType string
		var count int
		if rows.Scan(&alertType, &count) == nil {
			counts[alertType] = count
		}
	}
	if counts["not_run"] != 1 {
		t.Fatalf("active not_run = %d, want 1 (counts=%v)", counts["not_run"], counts)
	}
	if counts["backup_failed"] != 0 {
		t.Fatalf("active backup_failed = %d, want 0 (counts=%v)", counts["backup_failed"], counts)
	}
}

func TestCleanupNotifyRetentionIndependent(t *testing.T) {
	cases := []struct {
		name      string
		retention int
		audit     int
		notify    int
		want      int
	}{
		{name: "其它保留为 0 时仍清理发送记录", retention: 0, audit: 0, notify: 90, want: 0},
		{name: "全部为 0 时不清理", retention: 0, audit: 0, notify: 0, want: 1},
		{name: "执行保留开启时清理发送记录", retention: 90, audit: 0, notify: 90, want: 0},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			svc := newTestService(t)
			settings := domain.DefaultAppSettings()
			settings.RetentionDays = item.retention
			settings.AuditRetentionDays = item.audit
			settings.NotifyRetentionDays = item.notify
			if err := svc.SaveSettings(settings); err != nil {
				t.Fatalf("save settings: %v", err)
			}
			old := time.Now().UTC().Add(-200 * 24 * time.Hour).Format(time.RFC3339)
			if _, err := svc.Repo().DB().Exec(`INSERT INTO notifications(id,channel,type,level,status,created_at)
				VALUES('n1','mail','backup_failed','critical','sent',?)`, old); err != nil {
				t.Fatalf("insert notification: %v", err)
			}
			svc.RunScheduler(time.Now().UTC())
			var count int
			if err := svc.Repo().DB().QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&count); err != nil {
				t.Fatalf("count notifications: %v", err)
			}
			if count != item.want {
				t.Fatalf("notifications = %d, want %d", count, item.want)
			}
		})
	}
}
