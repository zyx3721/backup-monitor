package service_test

import (
	"testing"
	"time"

	"backup-monitor-backend/internal/domain"
)

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
