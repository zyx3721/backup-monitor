package service

import (
	"context"
	"log"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
)

// StartScheduler 启动周期巡检；每次循环按最新设置的间隔重排。
func (s *BackupService) StartScheduler(ctx context.Context) {
	go func() {
		for {
			s.RunScheduler(time.Now())
			interval := time.Duration(s.Settings().SchedulerIntervalSec) * time.Second
			if interval < 5*time.Second {
				interval = 5 * time.Second
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}()
}

// RunScheduler 执行一轮巡检：主机在线状态、任务漏执行、过期数据清理。
func (s *BackupService) RunScheduler(now time.Time) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Scheduler panic: %v", recovered)
		}
	}()
	hosts, err := s.repo.ListHosts()
	if err != nil {
		log.Printf("Scheduler list hosts failed: %s", err)
		return
	}
	unreachable := map[string]bool{}
	for _, host := range hosts {
		status := s.RecomputeHostStatus(host.ID, now)
		reason := "timeout"
		if s.hub != nil && s.hub.IsDisconnected(host.ID) {
			status = string(domain.HostOffline)
			reason = "heartbeat"
			_, _ = s.repo.DB().Exec(`UPDATE hosts SET status='offline' WHERE id=?`, host.ID)
		}
		if status == string(domain.HostAbnormal) || status == string(domain.HostOffline) {
			s.AlertReportTimeout(host.ID, host.Hostname, status, reason)
			unreachable[host.ID] = true
			s.ResolveHostTaskAlerts(host.ID, []string{"backup_failed", "not_run"})
		} else {
			s.ResolveAlerts("report_timeout", "", host.ID)
		}
	}

	tasks, err := s.repo.ListTasks()
	if err != nil {
		log.Printf("Scheduler list tasks failed: %s", err)
		return
	}
	for i := range tasks {
		task := &tasks[i]
		if unreachable[task.HostID] {
			continue
		}
		status := s.RecomputeTaskStatus(task.ID, now)
		if status == string(domain.TaskNotRun) {
			schedule := task.ScheduleCron
			if schedule == "" {
				schedule = CronFromHuman(task.Schedule)
			}
			if scheduled, ok := LastScheduled(schedule, now); ok {
				s.alertNotRun(task, scheduled)
			}
		} else {
			s.ResolveAlerts("not_run", task.ID, "")
		}
	}
	s.cleanup(now)
}

func (s *BackupService) cleanup(now time.Time) {
	settings := s.Settings()
	if settings.RetentionDays <= 0 && settings.AuditRetentionDays <= 0 {
		return
	}
	s.mu.Lock()
	if !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < time.Hour {
		s.mu.Unlock()
		return
	}
	s.lastCleanup = now
	s.mu.Unlock()

	if settings.RetentionDays > 0 {
		cutoff := appcommon.FormatTimestamp(now.Add(-time.Duration(settings.RetentionDays) * 24 * time.Hour))
		for _, statement := range []string{
			`DELETE FROM executions WHERE creation_time < ?`,
			`DELETE FROM alerts WHERE status='resolved' AND created_at < ?`,
		} {
			if _, err := s.repo.DB().Exec(statement, cutoff); err != nil {
				log.Printf("Cleanup failed: %s", err)
			}
		}
	}
	if settings.NotifyRetentionDays > 0 {
		cutoff := appcommon.FormatTimestamp(now.Add(-time.Duration(settings.NotifyRetentionDays) * 24 * time.Hour))
		if _, err := s.repo.DB().Exec(`DELETE FROM notifications WHERE created_at < ?`, cutoff); err != nil {
			log.Printf("Notification cleanup failed: %s", err)
		}
	}
	if settings.AuditRetentionDays > 0 {
		cutoff := appcommon.FormatTimestamp(now.Add(-time.Duration(settings.AuditRetentionDays) * 24 * time.Hour))
		if _, err := s.repo.DB().Exec(`DELETE FROM audit_logs WHERE date < ?`, cutoff); err != nil {
			log.Printf("Audit cleanup failed: %s", err)
		}
	}
	if _, err := s.repo.DB().Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		log.Printf("WAL checkpoint failed: %s", err)
	}
}
