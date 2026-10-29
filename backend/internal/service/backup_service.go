// Package service 实现备份监控的业务编排：采集上报、状态判定、告警与巡检。
package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
)

// BackupService 备份监控业务服务。
type BackupService struct {
	repo        *repository.BackupRepository
	hub         *AgentHub
	dataDir     string
	lastCleanup time.Time
	mu          sync.Mutex
}

// SetDataDir 指定 Agent 二进制与运行时数据目录。
func (s *BackupService) SetDataDir(dir string) {
	s.dataDir = dir
}

// NewBackupService 创建备份监控服务。
func NewBackupService(repo *repository.BackupRepository) *BackupService {
	return &BackupService{repo: repo}
}

// SetHub 注入 Agent WebSocket 连接中心（启动顺序决定）。
func (s *BackupService) SetHub(hub *AgentHub) {
	s.hub = hub
}

// Repo 暴露仓储供管理端处理函数使用。
func (s *BackupService) Repo() *repository.BackupRepository {
	return s.repo
}

// Settings 读取基础参数。
func (s *BackupService) Settings() domain.AppSettings {
	settings, err := s.repo.LoadBackupSettings()
	if err != nil {
		log.Printf("Load backup settings failed: %s", err)
		return domain.DefaultAppSettings()
	}
	return settings
}

// SaveSettings 保存基础参数。
func (s *BackupService) SaveSettings(settings domain.AppSettings) error {
	if settings.OfflineTimeoutMinutes < settings.ReportTimeoutMinutes {
		return fmt.Errorf("主机离线阈值不能小于主机上报超时")
	}
	return s.repo.SaveBackupSettings(settings)
}

// MatchProfile 按主机名匹配采集配置：优先级降序 → 精确匹配优先 → 字面量更具体优先 → 名称。
func (s *BackupService) MatchProfile(hostname string) (*domain.AgentProfile, error) {
	profiles, err := s.repo.ListAgentProfiles()
	if err != nil {
		return nil, err
	}
	return matchProfileFrom(profiles, hostname), nil
}

func globMatch(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(value)
}

func patternScore(pattern string) int {
	exact := !strings.ContainsAny(pattern, "*?")
	literal := len(strings.NewReplacer("*", "", "?", "").Replace(pattern))
	score := literal
	if exact {
		score += 100000
	}
	return score
}

// RecomputeHostStatus 按最后上报时间重算主机状态。
func (s *BackupService) RecomputeHostStatus(hostID string, now time.Time) string {
	var lastReport sql.NullString
	if err := s.repo.DB().QueryRow(`SELECT last_report_time FROM hosts WHERE id=?`, hostID).Scan(&lastReport); err != nil {
		return string(domain.HostOffline)
	}
	settings := s.Settings()
	status := string(domain.HostOffline)
	if lastReport.Valid && strings.TrimSpace(lastReport.String) != "" {
		reported, err := appcommon.ParseTimestamp(lastReport.String)
		if err != nil {
			status = string(domain.HostOffline)
		} else {
			elapsed := now.Sub(reported)
			switch {
			case elapsed > time.Duration(settings.OfflineTimeoutMinutes)*time.Minute:
				status = string(domain.HostOffline)
			case elapsed > time.Duration(settings.ReportTimeoutMinutes)*time.Minute:
				status = string(domain.HostAbnormal)
			default:
				status = string(domain.HostNormal)
			}
		}
	}
	_, _ = s.repo.DB().Exec(`UPDATE hosts SET status=? WHERE id=?`, status, hostID)
	return status
}

// RecomputeTaskStatus 综合计划与最新执行重算任务状态。
func (s *BackupService) RecomputeTaskStatus(taskID string, now time.Time) string {
	task, err := s.repo.GetTask(taskID)
	if err != nil || task == nil {
		return string(domain.TaskNoData)
	}
	latest, err := s.repo.LatestExecution(taskID)
	if err != nil {
		return string(domain.TaskNoData)
	}
	lastExecTime := ""
	if latest != nil {
		lastExecTime = preferredExecutionTime(latest.StartTime, latest.CreationTime)
	}
	status := string(domain.TaskNoData)
	schedule := task.ScheduleCron
	if schedule == "" {
		schedule = CronFromHuman(task.Schedule)
	}
	if scheduled, ok := LastScheduled(schedule, now); ok {
		grace := time.Duration(s.Settings().NotRunGraceMinutes) * time.Minute
		if now.Sub(scheduled) <= grace {
			if latest != nil {
				status = executionStateToTaskStatus(latest.State)
			}
		} else {
			window := time.Duration(s.Settings().ScheduleMatchWindowMinutes) * time.Minute
			if window <= 0 {
				window = 10 * time.Minute
			}
			windowStart := scheduled.Add(-window)
			windowEnd := scheduled.Add(7 * 24 * time.Hour)
			if next, ok := NextScheduled(schedule, scheduled); ok {
				windowEnd = next
			}
			if latest != nil {
				if created, err := parseTime(latest.CreationTime); err == nil && !created.Before(windowStart) && created.Before(windowEnd) {
					status = executionStateToTaskStatus(latest.State)
				} else {
					status = string(domain.TaskNotRun)
				}
			} else {
				status = string(domain.TaskNotRun)
			}
		}
	} else if latest != nil {
		status = executionStateToTaskStatus(latest.State)
	}
	_, _ = s.repo.DB().Exec(`UPDATE tasks SET status=?, last_execution_time=? WHERE id=?`, status, lastExecTime, taskID)
	return status
}

func executionStateToTaskStatus(state string) string {
	switch state {
	case "success":
		return string(domain.TaskNormal)
	case "running":
		return string(domain.TaskRunning)
	}
	return string(domain.TaskFailed)
}

func parseTime(value string) (time.Time, error) {
	return appcommon.ParseTimestamp(value)
}

const dedupeOffSuppress = time.Hour

type alertParams struct {
	Type     string
	Level    string
	HostID   string
	TaskID   string
	HostName string
	TaskName string
	Message  string
}

// RaiseAlert 创建或更新告警，并在新告警产生时异步发送通知。
func (s *BackupService) RaiseAlert(params alertParams) {
	settings := s.Settings()
	if settings.DedupeAlerts {
		id, level, message, err := s.repo.AlertExistsActive(params.Type, params.TaskID, params.HostID)
		if err != nil {
			log.Printf("Query active alert failed: %s", err)
		}
		if id != "" {
			if level != params.Level || message != params.Message {
				_, _ = s.repo.DB().Exec(`UPDATE alerts SET level=?, message=? WHERE id=?`, params.Level, params.Message, id)
			}
			return
		}
	} else {
		since := appcommon.FormatTimestamp(time.Now().Add(-dedupeOffSuppress))
		recent, err := s.repo.AlertExistsRecent(params.Type, params.TaskID, params.HostID, since)
		if err == nil && recent {
			return
		}
	}
	now := appcommon.NowTimestamp()
	_, err := s.repo.DB().Exec(`INSERT INTO alerts(id,host_id,task_id,host_name,task_name,type,level,status,created_at,message)
		VALUES(?,?,?,?,?,?,?, 'active', ?,?)`,
		newUUID(), params.HostID, nullableString(params.TaskID), params.HostName, nullableString(params.TaskName),
		params.Type, params.Level, now, params.Message)
	if err != nil {
		log.Printf("Insert alert failed: %s", err)
		return
	}
	hostIP := ""
	if host, err := s.repo.GetHost(params.HostID); err == nil && host != nil {
		hostIP = host.IP
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("Notify alert panic: %v", recovered)
			}
		}()
		_ = s.NotifyAlert(NotifyInput{Type: params.Type, Level: params.Level, Message: params.Message,
			HostName: params.HostName, HostIP: hostIP, TaskName: params.TaskName, Status: "未处理"})
	}()
}

type resolvedAlertRow struct {
	HostID   string
	HostName string
	TaskName string
	Level    string
	Message  string
}

// ResolveAlerts 恢复指定范围的未处理告警，并按需推送恢复通知。
func (s *BackupService) ResolveAlerts(alertType, taskID, hostID string) {
	query := `SELECT host_id,host_name,COALESCE(task_name,''),level,COALESCE(message,'') FROM alerts WHERE type=? AND status='active'`
	args := []any{alertType}
	if taskID != "" {
		query += ` AND task_id=?`
		args = append(args, taskID)
	} else {
		query += ` AND host_id=?`
		args = append(args, hostID)
	}
	rows, err := s.repo.DB().Query(query, args...)
	if err != nil {
		return
	}
	var resolved []resolvedAlertRow
	for rows.Next() {
		var row resolvedAlertRow
		if rows.Scan(&row.HostID, &row.HostName, &row.TaskName, &row.Level, &row.Message) == nil {
			resolved = append(resolved, row)
		}
	}
	rows.Close()
	if len(resolved) == 0 {
		return
	}
	now := appcommon.NowTimestamp()
	if taskID != "" {
		_, _ = s.repo.DB().Exec(`UPDATE alerts SET status='resolved', resolved_at=? WHERE type=? AND task_id=? AND status='active'`, now, alertType, taskID)
	} else {
		_, _ = s.repo.DB().Exec(`UPDATE alerts SET status='resolved', resolved_at=? WHERE type=? AND host_id=? AND status='active'`, now, alertType, hostID)
	}
	for _, row := range resolved {
		hostIP := ""
		if host, err := s.repo.GetHost(row.HostID); err == nil && host != nil {
			hostIP = host.IP
		}
		go func(row resolvedAlertRow, hostIP string) {
			_ = s.NotifyRecover(NotifyInput{Type: alertType, Level: row.Level, Message: row.Message,
				HostName: row.HostName, HostIP: hostIP, TaskName: row.TaskName})
		}(row, hostIP)
	}
}

// ResolveHostTaskAlerts 抑制主机失联期间的任务告警（不推送恢复通知）。
func (s *BackupService) ResolveHostTaskAlerts(hostID string, types []string) {
	for _, alertType := range types {
		_, _ = s.repo.DB().Exec(`UPDATE alerts SET status='resolved' WHERE type=? AND host_id=? AND status='active'`,
			alertType, hostID)
	}
}

func (s *BackupService) alertBackupFailed(task *domain.BackupTask, execution *domain.BackupExecution) {
	message := "备份执行失败：执行状态为 failed"
	if strings.TrimSpace(execution.Message) != "" {
		message = "备份执行失败：" + execution.Message
	}
	s.RaiseAlert(alertParams{Type: "backup_failed", Level: "critical", HostID: task.HostID, TaskID: task.ID,
		HostName: task.HostName, TaskName: task.Name, Message: message})
}

func (s *BackupService) alertNotRun(task *domain.BackupTask, scheduled time.Time) {
	s.RaiseAlert(alertParams{Type: "not_run", Level: "critical", HostID: task.HostID, TaskID: task.ID,
		HostName: task.HostName, TaskName: task.Name,
		Message: fmt.Sprintf("任务 %s 计划于 %s 执行，但未检测到对应执行记录", task.Name, scheduled.Format("15:04"))})
}

// AlertReportTimeout 产生主机上报超时/离线告警。
func (s *BackupService) AlertReportTimeout(hostID, hostname, status, reason string) {
	settings := s.Settings()
	offline := status == string(domain.HostOffline)
	message := fmt.Sprintf("主机 %s 上报异常：超过 %d 分钟未上报", hostname, settings.ReportTimeoutMinutes)
	if offline {
		if reason == "heartbeat" {
			message = fmt.Sprintf("主机 %s 心跳中断，已离线", hostname)
		} else {
			message = fmt.Sprintf("主机 %s 已离线：超过离线阈值 %d 分钟未上报", hostname, settings.OfflineTimeoutMinutes)
		}
	}
	level := "warning"
	if offline {
		level = "critical"
	}
	s.RaiseAlert(alertParams{Type: "report_timeout", Level: level, HostID: hostID, HostName: hostname, Message: message})
}

var (
	artifactTailPattern = regexp.MustCompile(`[;；]\s*产物\s+(.+?)(?:（[^）]*）)?\s*$`)
	artifactOnlyPattern = regexp.MustCompile(`检测到备份产物[:：]\s*(.+?)\s*$`)
)

var validStates = map[string]bool{"success": true, "running": true, "failed": true}
var validTypes = map[string]bool{"shell": true, "veeam": true, "powershell": true, "batch": true, "exe": true}
var validTriggers = map[string]bool{"scheduled": true, "manual": true}

// IngestResult 上报处理结果。
type IngestResult struct {
	HostID        string `json:"hostId"`
	NewExecutions int    `json:"newExecutions"`
}

// Ingest 处理一次 Agent 上报（事务原子写入）。
func (s *BackupService) Ingest(ctx context.Context, report domain.IngestReport) (IngestResult, error) {
	db := s.repo.DB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return IngestResult{}, err
	}
	defer tx.Rollback()

	tasks := make([]domain.IngestTask, 0, len(report.Tasks))
	for _, task := range report.Tasks {
		if strings.TrimSpace(task.Name) == "" || !validTypes[task.Type] {
			continue
		}
		tasks = append(tasks, task)
	}
	executions := make([]domain.IngestExecution, 0, len(report.Executions))
	for _, execution := range report.Executions {
		if strings.TrimSpace(execution.TaskName) == "" || !validStates[execution.State] {
			continue
		}
		if strings.TrimSpace(execution.CreationTime) == "" {
			continue
		}
		normalized, err := normalizeTime(execution.CreationTime)
		if err != nil {
			continue
		}
		execution.CreationTime = normalized
		executions = append(executions, execution)
	}

	hostID, err := resolveHost(tx, report.Host)
	if err != nil {
		return IngestResult{}, err
	}
	now := appcommon.NowTimestamp()
	if hostID == "" {
		hostID = newUUID()
		if _, err := tx.Exec(`INSERT INTO hosts(id,hostname,node_id,ip,os,arch,os_build,role,agent_version,agent_started_at,platform,status,last_report_time,created_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?, 'normal', ?, ?)`,
			hostID, report.Host.Hostname, nullableString(report.Host.NodeID), report.Host.IP, report.Host.OS,
			report.Host.Arch, report.Host.OSBuild, report.Host.Role, report.Host.AgentVersion, report.Host.AgentStartedAt, platformFromOS(report.Host.OS), now, now); err != nil {
			return IngestResult{}, err
		}
	} else {
		hostname := report.Host.Hostname
		if report.Host.NodeID != "" {
			var conflictID, conflictNode sql.NullString
			err := tx.QueryRow(`SELECT id,node_id FROM hosts WHERE hostname=? AND id<>?`, hostname, hostID).Scan(&conflictID, &conflictNode)
			if err == nil {
				if !conflictNode.Valid || conflictNode.String == "" {
					_ = deleteHostTx(tx, conflictID.String)
				} else if current, err := hostName(tx, hostID); err == nil {
					hostname = current
				}
			}
		}
		if _, err := tx.Exec(`UPDATE hosts SET hostname=?, ip=?, node_id=COALESCE(NULLIF(?,''),node_id), os=COALESCE(NULLIF(?,''),os),
			arch=COALESCE(NULLIF(?,''),arch),
			os_build=CASE WHEN ?>0 THEN ? ELSE os_build END,
			agent_version=COALESCE(NULLIF(?,''),agent_version),
			agent_started_at=COALESCE(NULLIF(?,''),agent_started_at),
			platform=COALESCE(NULLIF(?,''),platform), last_report_time=? WHERE id=?`,
			hostname, report.Host.IP, report.Host.NodeID, report.Host.OS, report.Host.Arch,
			report.Host.OSBuild, report.Host.OSBuild, report.Host.AgentVersion, report.Host.AgentStartedAt,
			platformFromOS(report.Host.OS), now, hostID); err != nil {
			return IngestResult{}, err
		}
		if _, err := tx.Exec(`UPDATE tasks SET host_name=? WHERE host_id=?`, hostname, hostID); err != nil {
			return IngestResult{}, err
		}
	}
	if s.hub != nil {
		s.hub.ClearDisconnected(hostID)
	}

	taskMap, err := upsertTasks(tx, hostID, report.Host.Hostname, tasks)
	if err != nil {
		return IngestResult{}, err
	}
	newExecutions, err := insertExecutions(tx, taskMap, executions)
	if err != nil {
		return IngestResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return IngestResult{}, err
	}

	s.RecomputeHostStatus(hostID, time.Now())
	s.ResolveAlerts("report_timeout", "", hostID)
	for _, task := range taskMap {
		status := s.RecomputeTaskStatus(task.ID, time.Now())
		latest, _ := s.repo.LatestExecution(task.ID)
		if latest != nil {
			switch latest.State {
			case "failed":
				s.alertBackupFailed(&task, latest)
			case "success":
				s.ResolveAlerts("backup_failed", task.ID, "")
				s.ResolveAlerts("not_run", task.ID, "")
			default:
				s.ResolveAlerts("not_run", task.ID, "")
			}
		}
		if status == string(domain.TaskNotRun) {
			schedule := task.ScheduleCron
			if schedule == "" {
				schedule = CronFromHuman(task.Schedule)
			}
			if scheduled, ok := LastScheduled(schedule, time.Now()); ok {
				s.alertNotRun(&task, scheduled)
			}
		} else {
			s.ResolveAlerts("not_run", task.ID, "")
		}
	}
	return IngestResult{HostID: hostID, NewExecutions: newExecutions}, nil
}

func resolveHost(tx *sql.Tx, host domain.IngestHost) (string, error) {
	if strings.TrimSpace(host.NodeID) != "" {
		var id string
		err := tx.QueryRow(`SELECT id FROM hosts WHERE node_id=?`, host.NodeID).Scan(&id)
		if err == nil {
			return id, nil
		}
		if err != sql.ErrNoRows {
			return "", err
		}
	}
	var id string
	err := tx.QueryRow(`SELECT id FROM hosts WHERE hostname=?`, host.Hostname).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

func hostName(tx *sql.Tx, hostID string) (string, error) {
	var name string
	err := tx.QueryRow(`SELECT hostname FROM hosts WHERE id=?`, hostID).Scan(&name)
	return name, err
}

func deleteHostTx(tx *sql.Tx, hostID string) error {
	if _, err := tx.Exec(`DELETE FROM executions WHERE task_id IN (SELECT id FROM tasks WHERE host_id=?)`, hostID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM alerts WHERE host_id=? OR task_id IN (SELECT id FROM tasks WHERE host_id=?)`, hostID, hostID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tasks WHERE host_id=?`, hostID); err != nil {
		return err
	}
	_, err := tx.Exec(`DELETE FROM hosts WHERE id=?`, hostID)
	return err
}

func upsertTasks(tx *sql.Tx, hostID, hostname string, tasks []domain.IngestTask) (map[string]domain.BackupTask, error) {
	taskMap := map[string]domain.BackupTask{}
	seenKeys := map[string]bool{}
	seenNames := map[string]bool{}
	for _, input := range tasks {
		name := strings.TrimSpace(input.Name)
		key := strings.TrimSpace(input.Key)
		if seenNames[name] {
			continue
		}
		if key != "" && seenKeys[key] {
			continue
		}
		seenNames[name] = true
		if key != "" {
			seenKeys[key] = true
		}
		config := normalizeTaskConfig(input.Config)
		effectiveType := deriveScriptMethod(input.Type, config)
		schedule, scheduleDisabled := normalizeIncomingSchedule(input.Schedule, input.ScheduleCron)
		if effectiveType == "veeam" {
			if schedule == "" {
				delete(config, "schedule")
			} else {
				config["schedule"] = schedule
			}
		}
		configJSON, _ := marshalJSON(config)

		var existing domain.BackupTask
		found := false
		if key != "" {
			existing, found = queryTask(tx, `SELECT id,host_id,host_name,name,COALESCE(task_key,''),type,COALESCE(tool,''),
				COALESCE(schedule,''),COALESCE(schedule_cron,''),schedule_locked,status,COALESCE(last_execution_time,''),config,'' 
				FROM tasks WHERE host_id=? AND task_key=?`, hostID, key)
		}
		if !found {
			existing, found = queryTask(tx, `SELECT id,host_id,host_name,name,COALESCE(task_key,''),type,COALESCE(tool,''),
				COALESCE(schedule,''),COALESCE(schedule_cron,''),schedule_locked,status,COALESCE(last_execution_time,''),config,'' 
				FROM tasks WHERE host_id=? AND name=?`, hostID, name)
		}
		if !found {
			existing, found = findLegacyLabelTask(tx, hostID, name, effectiveType, input.ScheduleCron)
		}
		if found {
			nextName := name
			var conflict int
			_ = tx.QueryRow(`SELECT COUNT(*) FROM tasks WHERE host_id=? AND name=? AND id<>?`, hostID, nextName, existing.ID).Scan(&conflict)
			if conflict > 0 {
				nextName = existing.Name
			}
			if _, err := tx.Exec(`UPDATE tasks SET name=?, task_key=COALESCE(NULLIF(?,''),task_key), type=?, tool=COALESCE(NULLIF(?,''),tool),
				schedule=COALESCE(NULLIF(?,''),schedule), schedule_cron=COALESCE(NULLIF(?,''),schedule_cron), config=? WHERE id=?`,
				nextName, key, effectiveType, input.Tool, schedule, input.ScheduleCron, configJSON, existing.ID); err != nil {
				return nil, err
			}
			if scheduleDisabled && schedule == "" {
				if _, err := tx.Exec(`UPDATE tasks SET schedule='' WHERE id=?`, existing.ID); err != nil {
					return nil, err
				}
			}
			existing.Name = nextName
			existing.TaskKey = key
			existing.Type = effectiveType
			existing.Tool = input.Tool
			existing.Schedule = schedule
			existing.ScheduleCron = input.ScheduleCron
			existing.Config = config
			taskMap[name] = existing
			continue
		}
		id := newUUID()
		if _, err := tx.Exec(`INSERT INTO tasks(id,host_id,host_name,name,task_key,type,tool,schedule,schedule_cron,status,config,created_at)
			VALUES(?,?,?,?,?,?,?,?,?, 'no_data', ?,?)`,
			id, hostID, hostname, name, nullableString(key), effectiveType, input.Tool, schedule, input.ScheduleCron, configJSON,
			appcommon.NowTimestamp()); err != nil {
			return nil, err
		}
		taskMap[name] = domain.BackupTask{ID: id, HostID: hostID, HostName: hostname, Name: name, TaskKey: key, Type: effectiveType,
			Tool: input.Tool, Schedule: schedule, ScheduleCron: input.ScheduleCron, Config: config}
	}
	return taskMap, nil
}

func queryTask(tx *sql.Tx, query string, args ...any) (domain.BackupTask, bool) {
	var task domain.BackupTask
	var locked int
	var configRaw string
	err := tx.QueryRow(query, args...).Scan(&task.ID, &task.HostID, &task.HostName, &task.Name, &task.TaskKey, &task.Type,
		&task.Tool, &task.Schedule, &task.ScheduleCron, &locked, &task.Status, &task.LastExecutionTime, &configRaw, &task.HostIP)
	if err != nil {
		return task, false
	}
	task.ScheduleLocked = locked == 1
	task.Config = domain.ParseJSONObject(configRaw)
	return task, true
}

func insertExecutions(tx *sql.Tx, taskMap map[string]domain.BackupTask, executions []domain.IngestExecution) (int, error) {
	sort.SliceStable(executions, func(i, j int) bool { return executions[i].CreationTime < executions[j].CreationTime })
	inserted := 0
	for _, execution := range executions {
		task, ok := taskMap[execution.TaskName]
		if !ok {
			continue
		}
		creationTime := execution.CreationTime
		startTime, _ := normalizeOptionalTime(execution.StartTime)
		endTime, _ := normalizeOptionalTime(execution.EndTime)
		artifactPath := strings.TrimSpace(execution.ArtifactPath)
		if artifactPath == "" {
			artifactPath = parseArtifactPath(execution.Message)
		}
		var existing string
		if execution.SessionID != "" {
			_ = tx.QueryRow(`SELECT id FROM executions WHERE task_id=? AND session_id=?`, task.ID, execution.SessionID).Scan(&existing)
		} else {
			_ = tx.QueryRow(`SELECT id FROM executions WHERE task_id=? AND creation_time=?`, task.ID, creationTime).Scan(&existing)
		}
		trigger := ""
		if validTriggers[execution.Trigger] {
			trigger = execution.Trigger
		}
		if existing != "" {
			if _, err := tx.Exec(`UPDATE executions SET state=?, trigger=COALESCE(NULLIF(?,''),trigger), end_time=COALESCE(NULLIF(?,''),end_time), exit_code=COALESCE(?,exit_code),
				message=COALESCE(NULLIF(?,''),message), processed_size=COALESCE(NULLIF(?,''),processed_size),
				transferred_size=COALESCE(NULLIF(?,''),transferred_size), artifact_path=COALESCE(NULLIF(?,''),artifact_path),
				job_id=COALESCE(NULLIF(?,''),job_id), oib_id=COALESCE(NULLIF(?,''),oib_id), attempt=COALESCE(?,attempt),
				max_attempts=COALESCE(?,max_attempts), is_last_attempt=COALESCE(?,is_last_attempt) WHERE id=?`,
				execution.State, trigger, endTime, execution.ExitCode, execution.Message, execution.ProcessedSize, execution.TransferredSize,
				artifactPath, execution.JobID, execution.OIBID, execution.Attempt, execution.MaxAttempts, boolToIntPtr(execution.IsLastAttempt), existing); err != nil {
				return inserted, err
			}
			continue
		}
		if _, err := tx.Exec(`INSERT INTO executions(id,task_id,session_id,job_id,oib_id,state,trigger,creation_time,start_time,end_time,
			processed_size,transferred_size,artifact_path,exit_code,attempt,max_attempts,is_last_attempt,message)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			newUUID(), task.ID, nullableString(execution.SessionID), nullableString(execution.JobID), nullableString(execution.OIBID),
			execution.State, nullableString(trigger), creationTime, startTime, endTime, execution.ProcessedSize, execution.TransferredSize,
			nullableString(artifactPath), execution.ExitCode, execution.Attempt, execution.MaxAttempts, boolToIntPtr(execution.IsLastAttempt),
			execution.Message); err != nil {
			return inserted, err
		}
		inserted++
		if execution.SessionID != "" {
			_, _ = tx.Exec(`UPDATE executions SET state='failed', message='执行中断（未收到结束状态）'
				WHERE task_id=? AND state='running' AND creation_time<? AND (session_id IS NULL OR session_id<>?)`,
				task.ID, creationTime, execution.SessionID)
		} else {
			_, _ = tx.Exec(`UPDATE executions SET state='failed', message='执行中断（未收到结束状态）'
				WHERE task_id=? AND state='running' AND creation_time<?`, task.ID, creationTime)
		}
	}
	return inserted, nil
}

func normalizeTaskConfig(config map[string]any) map[string]any {
	if config == nil {
		return map[string]any{}
	}
	if source, _ := config["source"].(string); source != "schtasks" {
		return config
	}
	if script, _ := config["script"].(string); strings.TrimSpace(script) != "" {
		return config
	}
	command, _ := config["command"].(string)
	launcher := firstToken(command)
	if launcher == "" {
		return config
	}
	out := map[string]any{}
	for key, value := range config {
		out[key] = value
	}
	out["script"] = launcher
	return out
}

func deriveScriptMethod(taskType string, config map[string]any) string {
	if taskType != "shell" {
		return taskType
	}
	script, _ := config["script"].(string)
	command, _ := config["command"].(string)
	ext := extensionOf(script)
	if ext == "" {
		ext = extensionOf(firstToken(command))
	}
	switch ext {
	case "ps1":
		return "powershell"
	case "bat", "cmd":
		return "batch"
	case "exe":
		return "exe"
	}
	return "shell"
}

func extensionOf(pathLike string) string {
	text := strings.Trim(strings.TrimSpace(pathLike), `"`)
	idx := strings.LastIndex(text, ".")
	if idx < 0 || idx == len(text)-1 {
		return ""
	}
	ext := text[idx+1:]
	for _, r := range ext {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return ""
		}
	}
	return strings.ToLower(ext)
}

func firstToken(command string) string {
	text := strings.TrimSpace(command)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, `"`) {
		if end := strings.Index(text[1:], `"`); end >= 0 {
			return text[1 : end+1]
		}
		return text[1:]
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func parseArtifactPath(message string) string {
	if message == "" {
		return ""
	}
	matches := artifactTailPattern.FindStringSubmatch(message)
	if matches == nil {
		matches = artifactOnlyPattern.FindStringSubmatch(message)
	}
	if matches == nil {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func normalizeTime(value string) (string, error) {
	parsed, err := parseTime(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	return appcommon.FormatTimestamp(parsed), nil
}

func normalizeOptionalTime(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return normalizeTime(value)
}

func platformFromOS(osName string) string {
	name := strings.ToLower(strings.TrimSpace(osName))
	if name == "" {
		return ""
	}
	if strings.Contains(name, "win") {
		return "windows"
	}
	return "linux"
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func boolToIntPtr(value *bool) any {
	if value == nil {
		return nil
	}
	if *value {
		return 1
	}
	return 0
}
