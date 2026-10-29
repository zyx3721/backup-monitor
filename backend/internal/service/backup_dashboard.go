package service

import (
	"sort"
	"time"

	"backup-monitor-backend/internal/domain"
)

// TaskStats 任务执行统计。
type TaskStats struct {
	ConsecutiveFailures int
	SuccessRate         int
	LastSuccessTime     string
	LatestSize          string
}

// BuildTaskStats 基于执行记录汇总每个任务的连续失败、成功率、最近成功与最新体积。
func (s *BackupService) BuildTaskStats() (map[string]TaskStats, error) {
	grouped, err := s.repo.ExecutionStates()
	if err != nil {
		return nil, err
	}
	stats := map[string]TaskStats{}
	for taskID, executions := range grouped {
		item := TaskStats{}
		for _, execution := range executions {
			if execution.State == "failed" {
				item.ConsecutiveFailures++
			} else {
				break
			}
		}
		window := executions
		if len(window) > 30 {
			window = window[:30]
		}
		success, finished := 0, 0
		lastSuccess := ""
		for _, execution := range window {
			if execution.State == "success" {
				success++
				if lastSuccess == "" {
					lastSuccess = preferredExecutionTime(execution.StartTime, execution.CreationTime)
				}
			}
			if execution.State != "running" {
				finished++
			}
		}
		if finished > 0 {
			item.SuccessRate = success * 100 / finished
		}
		item.LastSuccessTime = lastSuccess
		if len(executions) > 0 {
			latest := executions[0]
			if latest.TransferredSize != "" {
				item.LatestSize = latest.TransferredSize
			} else {
				item.LatestSize = latest.ProcessedSize
			}
		}
		stats[taskID] = item
	}
	return stats, nil
}

// DecorateTasks 为任务补充执行统计。
func (s *BackupService) DecorateTasks(tasks []domain.BackupTask) []domain.BackupTask {
	stats, err := s.BuildTaskStats()
	if err != nil {
		return tasks
	}
	for i := range tasks {
		if item, ok := stats[tasks[i].ID]; ok {
			tasks[i].ConsecutiveFailures = item.ConsecutiveFailures
			tasks[i].SuccessRate = item.SuccessRate
			tasks[i].LastSuccessTime = item.LastSuccessTime
			tasks[i].LatestSize = item.LatestSize
		}
	}
	return tasks
}

// BuildDashboard 汇总概览数据。
func (s *BackupService) BuildDashboard() (domain.Dashboard, error) {
	hosts, err := s.repo.ListHosts()
	if err != nil {
		return domain.Dashboard{}, err
	}
	tasks, err := s.repo.ListTasks()
	if err != nil {
		return domain.Dashboard{}, err
	}
	tasks = s.DecorateTasks(tasks)
	executions, err := s.repo.ListExecutions(0)
	if err != nil {
		return domain.Dashboard{}, err
	}
	alerts, err := s.repo.ListAlerts()
	if err != nil {
		return domain.Dashboard{}, err
	}

	out := domain.Dashboard{HostCount: len(hosts), TaskCount: len(tasks)}
	for _, host := range hosts {
		switch host.Status {
		case string(domain.HostNormal):
			out.HostNormal++
		case string(domain.HostAbnormal):
			out.HostAbnormal++
		default:
			out.HostOffline++
		}
	}
	statusOrder := []string{"normal", "running", "failed", "not_run", "no_data"}
	statusCount := map[string]int{}
	for _, task := range tasks {
		statusCount[task.Status]++
	}
	for _, status := range statusOrder {
		out.StatusSummary = append(out.StatusSummary, domain.StatusSummary{Status: status, Count: statusCount[status]})
	}
	out.AbnormalTasks = statusCount["failed"]
	out.NotRunTasks = statusCount["not_run"]

	today := time.Now().Format("2006-01-02")
	trendBuckets := map[string]*domain.TrendPoint{}
	days := make([]string, 0, 14)
	for i := 13; i >= 0; i-- {
		key := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		days = append(days, key)
		trendBuckets[key] = &domain.TrendPoint{Date: key}
	}
	for _, execution := range executions {
		created, err := parseTime(execution.CreationTime)
		if err != nil {
			continue
		}
		if created.Local().Format("2006-01-02") == today {
			out.TodayTotal++
			switch execution.State {
			case "success":
				out.TodaySuccess++
			case "failed":
				out.TodayFailed++
			}
		}
		if bucket, ok := trendBuckets[created.Local().Format("2006-01-02")]; ok {
			switch execution.State {
			case "success":
				bucket.Success++
			case "failed":
				bucket.Failed++
			default:
				bucket.Other++
			}
		}
	}
	if out.TodayTotal > 0 {
		out.SuccessRate = out.TodaySuccess * 100 / out.TodayTotal
	}
	for _, day := range days {
		out.Trend = append(out.Trend, *trendBuckets[day])
	}

	for _, method := range []string{"veeam", "shell", "powershell", "batch", "exe"} {
		item := domain.MethodSummary{Method: method}
		for _, task := range tasks {
			if task.Type != method {
				continue
			}
			item.Total++
			switch task.Status {
			case string(domain.TaskNormal):
				item.Normal++
			case string(domain.TaskFailed):
				item.Failed++
			case string(domain.TaskNotRun):
				item.NotRun++
			}
		}
		out.MethodSummary = append(out.MethodSummary, item)
	}

	abnormal := []domain.BackupTask{}
	for _, task := range tasks {
		if task.Status == string(domain.TaskFailed) || task.Status == string(domain.TaskNotRun) {
			abnormal = append(abnormal, task)
		}
	}
	sort.SliceStable(abnormal, func(i, j int) bool { return abnormal[i].LastExecutionTime > abnormal[j].LastExecutionTime })
	if len(abnormal) > 10 {
		abnormal = abnormal[:10]
	}
	out.RecentAbnormalTasks = abnormal
	for _, alert := range alerts {
		if alert.Status == "active" {
			out.ActiveAlerts++
		}
	}
	out.RecentAlerts = alerts
	if len(out.RecentAlerts) > 8 {
		out.RecentAlerts = out.RecentAlerts[:8]
	}
	return out, nil
}
