package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/cron"
	"backup-monitor-agent/internal/discover"
	"backup-monitor-agent/internal/proc"
	"backup-monitor-agent/internal/statusfile"
	"backup-monitor-agent/internal/veeam"
)

func (c *Collector) executionsFor(task discover.Task, profile api.ProfileConfig, now time.Time) []api.IngestExecution {
	var scheduled time.Time
	hasScheduled := false
	if strings.TrimSpace(task.ScheduleCron) != "" {
		scheduled, hasScheduled = cron.GetLastScheduledTime(task.ScheduleCron, now)
	}
	running := c.procRunning(task)

	executions := make([]api.IngestExecution, 0, 2)
	statusExec := c.statusExecution(task, profile, now, scheduled, hasScheduled, running)
	if statusExec != nil {
		executions = append(executions, *statusExec)
	}

	if task.HasLastRun {
		if statusExec == nil || !sameRun(statusExec.CreationTime, task.LastRun) {
			if execution, ok := lastRunExecution(task, scheduled, hasScheduled, profile); ok {
				executions = append(executions, execution)
			}
		}
	}

	artifact := c.findArtifact(task, profile)
	if artifact != nil {
		if len(executions) > 0 {
			attachArtifact(&executions[len(executions)-1], artifact, profile.ArtifactPreToleranceMinutes)
		} else if profile.ArtifactCheck && artifactEligible(artifact, task, profile, scheduled, hasScheduled, now) {
			executions = append(executions, artifactExecution(artifact, task, scheduled, hasScheduled, profile))
		}
	}
	if statusExec == nil && running && hasScheduled {
		if execution, ok := procExecution(task, scheduled); ok {
			executions = append(executions, execution)
		}
	}
	return dedupeExecutions(executions)
}

func (c *Collector) procRunning(task discover.Task) bool {
	keyword := proc.Keyword(task.ProcKeyword)
	if keyword == "" {
		return false
	}
	if !proc.Running(keyword) {
		return false
	}
	c.logf("任务 %s 正在执行：匹配进程 %q", task.Name, keyword)
	return true
}

func procExecution(task discover.Task, scheduled time.Time) (api.IngestExecution, bool) {
	if scheduled.IsZero() {
		return api.IngestExecution{}, false
	}
	return api.IngestExecution{
		TaskName:      task.Name,
		SessionID:     fmt.Sprintf("{proc:%s:%d}", task.Name, scheduled.UnixMilli()),
		IsLastAttempt: lastAttemptFlag(),
		State:         "running",
		Trigger:       "scheduled",
		CreationTime:  formatTime(scheduled),
		StartTime:     formatTime(scheduled),
		Message:       fmt.Sprintf("进程运行中（proc=%s）", proc.Keyword(task.ProcKeyword)),
	}, true
}

func (c *Collector) statusExecution(task discover.Task, profile api.ProfileConfig, now, scheduled time.Time, hasScheduled, running bool) *api.IngestExecution {
	paths := statusFileCandidates(task)
	for _, path := range paths {
		info, err := statusfile.ParseFile(path)
		if err != nil {
			c.logf("读取状态文件 %s 失败：%v", path, err)
			continue
		}
		execution, ok := executionFromStatus(task, path, info, now, scheduled, hasScheduled, running, profile)
		if ok {
			return &execution
		}
	}
	return nil
}

func executionFromStatus(task discover.Task, path string, info statusfile.Info, now, scheduled time.Time, hasScheduled, running bool, profile api.ProfileConfig) (api.IngestExecution, bool) {
	state := normalizeExecutionState(info.Status)
	if state == "" && info.HasExit {
		state = stateFromExit(info.ExitCode)
	}
	if state == "" {
		if info.HasStart && !info.HasEnd {
			state = "running"
		} else if info.HasEnd {
			state = "success"
		}
	}
	if state == "" && running {
		state = "running"
	}
	if state == "" {
		return api.IngestExecution{}, false
	}
	if state == "success" && info.HasExit && info.ExitCode != 0 {
		state = "failed"
	}

	creation := firstTime(info.Start, info.Time, info.End)
	if creation.IsZero() {
		if fileInfo, err := os.Stat(path); err == nil {
			creation = fileInfo.ModTime()
		}
	}
	if creation.IsZero() {
		creation = now
	}
	start := info.Start
	if start.IsZero() {
		start = creation
	}
	end := info.End
	if end.IsZero() && state != "running" {
		end = creation
	}
	if state == "running" {
		end = time.Time{}
	}

	execution := api.IngestExecution{
		TaskName:      task.Name,
		SessionID:     fmt.Sprintf("{status:%s:%d}", task.Name, creation.UnixMilli()),
		IsLastAttempt: lastAttemptFlag(),
		State:         state,
		Trigger:       triggerForStart(start, task.ScheduleCron, profile.ManualTriggerMinutes),
		CreationTime:  formatTime(creation),
		StartTime:     formatTime(start),
		EndTime:       formatTime(end),
		Message:       "状态文件：" + path,
	}
	if info.HasSize {
		execution.ProcessedSize = statusfile.FormatSize(info.SizeBytes)
	}
	if info.HasTransferred {
		execution.TransferredSize = statusfile.FormatSize(info.TransferredBytes)
	}
	if info.HasExit {
		code := info.ExitCode
		execution.ExitCode = &code
	}
	return execution, true
}

func lastRunExecution(task discover.Task, scheduled time.Time, hasScheduled bool, profile api.ProfileConfig) (api.IngestExecution, bool) {
	if !task.HasLastRun {
		return api.IngestExecution{}, false
	}
	state := stateFromExit(task.LastResult)
	running := strings.Contains(strings.ToLower(task.State), "running") || strings.Contains(task.State, "运行")
	if state == "" && !running {
		return api.IngestExecution{}, false
	}
	if running {
		state = "running"
	}
	start := task.LastRun
	end := start
	if state == "running" {
		end = time.Time{}
	}
	execution := api.IngestExecution{
		TaskName:      task.Name,
		SessionID:     fmt.Sprintf("schtasks:%d", start.Unix()),
		IsLastAttempt: lastAttemptFlag(),
		State:         state,
		Trigger:       triggerForStart(start, task.ScheduleCron, profile.ManualTriggerMinutes),
		CreationTime:  formatTime(start),
		StartTime:     formatTime(start),
		EndTime:       formatTime(end),
		Message:       fmt.Sprintf("计划任务最近一次运行，结果码 %d", task.LastResult),
	}
	if task.HasLastResult {
		code := task.LastResult
		execution.ExitCode = &code
	}
	if state == "running" {
		execution.Message = "计划任务正在运行"
	}
	return execution, true
}

func buildVeeamExecution(taskName, jobID string, session veeam.Session, scheduleCron string, profile api.ProfileConfig, now time.Time) (api.IngestExecution, bool) {
	state := session.State
	if state == "" {
		state = veeam.NormalizeState(session.Result)
	}
	if state == "" {
		if session.HasEnd {
			state = "failed"
		} else {
			state = "running"
		}
	}

	creation := session.Start
	if creation.IsZero() {
		creation = session.Creation
	}
	if creation.IsZero() {
		creation = session.End
	}
	if creation.IsZero() {
		creation = now
	}
	start := session.Start
	if start.IsZero() {
		start = creation
	}
	end := session.End
	if end.IsZero() && state != "running" {
		end = start
	}
	if state == "running" {
		end = time.Time{}
	}

	message := "Veeam 会话"
	if strings.TrimSpace(session.ID) != "" {
		message += "：" + strings.TrimSpace(session.ID)
	}
	execution := api.IngestExecution{
		TaskName:        taskName,
		SessionID:       strings.TrimSpace(session.ID),
		JobID:           jobID,
		OibID:           strings.TrimSpace(session.OIBID),
		State:           state,
		CreationTime:    formatTime(creation),
		StartTime:       formatTime(start),
		EndTime:         formatTime(end),
		ProcessedSize:   strings.TrimSpace(session.ProcessedSize),
		TransferredSize: strings.TrimSpace(session.TransferredSize),
		Message:         message,
	}
	if session.HasAttempt {
		attempt := session.Attempt
		execution.Attempt = &attempt
	}
	if session.HasMaxAttempts {
		maxAttempts := session.MaxAttempts
		execution.MaxAttempts = &maxAttempts
	}
	if session.HasLastAttempt {
		isLastAttempt := session.IsLastAttempt
		execution.IsLastAttempt = &isLastAttempt
	}
	if scheduleCron != "" {
		execution.Trigger = triggerForStart(start, scheduleCron, profile.ManualTriggerMinutes)
	}
	if execution.Trigger == "" {
		execution.Trigger = "scheduled"
	}
	return execution, true
}

func normalizeExecutionState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "success", "succeeded", "ok", "completed", "done", "finished":
		return "success"
	case "failed", "failure", "error", "fail":
		return "failed"
	case "running", "inprogress", "in-progress", "started", "working":
		return "running"
	default:
		lower := strings.ToLower(strings.TrimSpace(value))
		switch {
		case strings.Contains(lower, "fail"), strings.Contains(lower, "error"), strings.Contains(lower, "失败"):
			return "failed"
		case strings.Contains(lower, "abort"), strings.Contains(lower, "cancel"), strings.Contains(lower, "terminat"), strings.Contains(lower, "取消"), strings.Contains(lower, "终止"):
			return "failed"
		case strings.Contains(lower, "success"), strings.Contains(lower, "complete"), strings.Contains(lower, "成功"), strings.Contains(lower, "完成"):
			return "success"
		case strings.Contains(lower, "run"), strings.Contains(lower, "progress"), strings.Contains(lower, "运行"):
			return "running"
		default:
			return ""
		}
	}
}

func stateFromExit(code int) string {
	switch code {
	case 0:
		return "success"
	case 0x41301:
		return "running"
	case 0x41302, 0x41303, 0x41304, 0x41305, 0x41307, 0x41308:
		return ""
	case 0x41306:
		return "failed"
	case 0x41300:
		return ""
	default:
		return "failed"
	}
}

func triggerForStart(start time.Time, cronExpr string, manualMinutes int) string {
	if scheduled, ok := cron.GetLastScheduledTime(cronExpr, start); ok {
		return triggerFor(start, scheduled, true, manualMinutes)
	}
	return triggerFor(start, time.Time{}, false, manualMinutes)
}

func triggerFor(start, scheduled time.Time, hasScheduled bool, manualMinutes int) string {
	if !hasScheduled || scheduled.IsZero() || start.IsZero() || manualMinutes <= 0 {
		return "scheduled"
	}
	deviation := start.Sub(scheduled)
	if deviation < 0 {
		deviation = -deviation
	}
	if deviation > time.Duration(manualMinutes)*time.Minute {
		return "manual"
	}
	return "scheduled"
}

func statusFileCandidates(task discover.Task) []string {
	candidates := make([]string, 0, 3)
	for _, key := range []string{"statusFile", "status"} {
		if value := configString(task.Config, key); value != "" {
			candidates = append(candidates, value)
		}
	}
	script := configString(task.Config, "script")
	if script == "" {
		script = task.Script
	}
	if script != "" {
		candidates = append(candidates, script+".status")
		extension := filepath.Ext(script)
		candidates = append(candidates, strings.TrimSuffix(script, extension)+".status")
	}

	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) && script != "" {
			candidate = filepath.Join(filepath.Dir(script), candidate)
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			result = append(result, candidate)
		}
	}
	return result
}

func configString(config map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := config[key]
		if !exists {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func firstTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parseReportTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func sameRun(creation string, lastRun time.Time) bool {
	parsed := parseReportTime(creation)
	if parsed.IsZero() || lastRun.IsZero() {
		return false
	}
	deviation := parsed.Sub(lastRun)
	if deviation < 0 {
		deviation = -deviation
	}
	return deviation <= 5*time.Minute
}

func attachArtifact(execution *api.IngestExecution, artifact *artifactInfo, preToleranceMinutes int) bool {
	if execution == nil || artifact == nil {
		return false
	}
	if execution.State == "running" {
		return false
	}
	start := parseReportTime(execution.StartTime)
	if start.IsZero() {
		start = parseReportTime(execution.CreationTime)
	}
	if !start.IsZero() && artifact.ModTime.Before(start.Add(-time.Duration(preToleranceMinutes)*time.Minute)) {
		return false
	}
	sizeText := statusfile.FormatSize(artifact.Size)
	execution.ArtifactPath = artifact.Path
	if execution.ProcessedSize == "" {
		execution.ProcessedSize = sizeText
	}
	if execution.Message == "" {
		execution.Message = "检测到备份产物：" + artifact.Path
	} else {
		execution.Message += fmt.Sprintf("；产物 %s（%s）", artifact.Path, sizeText)
	}
	return true
}

func artifactEligible(artifact *artifactInfo, task discover.Task, profile api.ProfileConfig, scheduled time.Time, hasScheduled bool, now time.Time) bool {
	if artifact == nil {
		return false
	}
	if !hasScheduled {
		maxAge := time.Duration(profile.MaxArtifactAgeHours) * time.Hour
		if maxAge > 0 && now.Sub(artifact.ModTime) > maxAge {
			return false
		}
	}
	if hasScheduled && !scheduled.IsZero() {
		tolerance := time.Duration(profile.ArtifactPreToleranceMinutes) * time.Minute
		if artifact.ModTime.Before(scheduled.Add(-tolerance)) {
			return false
		}
	}
	if task.HasLastRun {
		tolerance := time.Duration(profile.ArtifactPreToleranceMinutes) * time.Minute
		if artifact.ModTime.Before(task.LastRun.Add(-tolerance)) {
			return false
		}
	}
	return true
}

func artifactExecution(artifact *artifactInfo, task discover.Task, scheduled time.Time, hasScheduled bool, profile api.ProfileConfig) api.IngestExecution {
	sizeText := statusfile.FormatSize(artifact.Size)
	state := "success"
	return api.IngestExecution{
		TaskName:      task.Name,
		IsLastAttempt: lastAttemptFlag(),
		State:         state,
		Trigger:       triggerForStart(artifact.ModTime, task.ScheduleCron, profile.ManualTriggerMinutes),
		CreationTime:  formatTime(artifact.ModTime),
		StartTime:     formatTime(artifact.ModTime),
		EndTime:       formatTime(artifact.ModTime),
		ProcessedSize: sizeText,
		ArtifactPath:  artifact.Path,
		Message:       "检测到备份产物：" + artifact.Path,
	}
}

func dedupeExecutions(executions []api.IngestExecution) []api.IngestExecution {
	seen := make(map[string]struct{}, len(executions))
	result := make([]api.IngestExecution, 0, len(executions))
	for _, execution := range executions {
		identity := execution.TaskName + "|" + execution.SessionID
		if execution.SessionID == "" {
			identity = execution.TaskName + "|" + execution.CreationTime
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		result = append(result, execution)
	}
	return result
}

func lastAttemptFlag() *bool {
	value := true
	return &value
}
