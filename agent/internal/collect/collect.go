// Package collect 把计划任务、状态文件、备份产物与 Veeam 会话转换为上报数据。
package collect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sort"
	"strings"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/config"
	"backup-monitor-agent/internal/cron"
	"backup-monitor-agent/internal/discover"
	"backup-monitor-agent/internal/sysinfo"
	"backup-monitor-agent/internal/veeam"
)

// Collector 按配置采集本机备份任务与执行记录。
type Collector struct {
	Hostname  string
	NodeID    string
	Role      string
	Version   string
	StartedAt time.Time
	Logger    *log.Logger
	Now       func() time.Time
}

// Options 是采集所需的配置快照。
type Options struct {
	Profile  api.ProfileConfig
	Settings api.Settings
}

type taskEntry struct {
	task       api.IngestTask
	executions []api.IngestExecution
}

// Collect 生成一份完整上报数据，单项采集失败只记录日志。
func (c *Collector) Collect(ctx context.Context, options Options) (api.IngestReport, error) {
	now := c.now()
	report := api.IngestReport{
		Host:       c.hostInfo(),
		Tasks:      []api.IngestTask{},
		Executions: []api.IngestExecution{},
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}

	profile := normalizeProfile(options.Profile)
	entries := make([]taskEntry, 0)

	if profile.CollectCrontab {
		filter := discover.Filter{
			LabelPrefix:     profile.LabelPrefix,
			IncludeKeywords: profile.IncludeKeywords,
			IncludePaths:    profile.IncludePaths,
			ExcludeKeywords: profile.ExcludeKeywords,
		}
		found, err := discover.List(ctx, filter)
		if err != nil {
			c.logf("计划任务采集完成，存在错误：%v", err)
		}
		for _, task := range found {
			entry := taskEntry{
				task:       toIngestTask(task),
				executions: c.executionsFor(task, profile, now),
			}
			entries = append(entries, entry)
		}
	}

	veeamTasks := []taskEntry(nil)
	if profile.CollectVeeam {
		veeamTasks = c.veeamEntries(ctx, profile, now)
		entries = append(entries, veeamTasks...)
	}

	entries = append(entries, c.platformEntries(ctx, profile, now, len(veeamTasks) > 0)...)
	entries = dedupeEntries(entries)
	ensureUniqueNames(entries)

	for _, entry := range entries {
		if strings.TrimSpace(entry.task.Name) == "" {
			continue
		}
		report.Tasks = append(report.Tasks, entry.task)
		report.Executions = append(report.Executions, entry.executions...)
	}

	sort.Slice(report.Tasks, func(i, j int) bool {
		return report.Tasks[i].Name < report.Tasks[j].Name
	})
	sort.Slice(report.Executions, func(i, j int) bool {
		if report.Executions[i].CreationTime == report.Executions[j].CreationTime {
			return report.Executions[i].TaskName < report.Executions[j].TaskName
		}
		return report.Executions[i].CreationTime < report.Executions[j].CreationTime
	})

	if err := ctx.Err(); err != nil {
		return report, err
	}
	return report, nil
}

func (c *Collector) hostInfo() api.IngestHost {
	hostname := strings.TrimSpace(c.Hostname)
	if hostname == "" {
		hostname = sysinfo.Hostname()
	}
	role := strings.TrimSpace(c.Role)
	if role == "" {
		role = config.RoleDatabase
	}
	return api.IngestHost{
		Hostname:       hostname,
		NodeID:         strings.TrimSpace(c.NodeID),
		IP:             sysinfo.PrivateIP(),
		OS:             sysinfo.OSName(),
		Arch:           sysinfo.Arch(),
		OSBuild:        sysinfo.OSBuild(),
		Role:           role,
		AgentVersion:   c.Version,
		AgentStartedAt: c.startedAtText(),
	}
}

// startedAtText 返回进程启动时间，未设置时返回空串。
func (c *Collector) startedAtText() string {
	if c.StartedAt.IsZero() {
		return ""
	}
	return c.StartedAt.UTC().Format(time.RFC3339Nano)
}

func (c *Collector) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Collector) logf(format string, args ...any) {
	if c.Logger != nil {
		c.Logger.Printf(format, args...)
	}
}

func toIngestTask(task discover.Task) api.IngestTask {
	taskType := strings.TrimSpace(task.Type)
	if taskType == "" {
		taskType = "shell"
	}
	configCopy := make(map[string]any, len(task.Config))
	for key, value := range task.Config {
		configCopy[key] = value
	}
	return api.IngestTask{
		Name:         task.Name,
		Key:          task.Key,
		Type:         taskType,
		Tool:         task.Tool,
		Schedule:     task.Schedule,
		ScheduleCron: task.ScheduleCron,
		Config:       configCopy,
	}
}

func normalizeProfile(profile api.ProfileConfig) api.ProfileConfig {
	defaults := api.DefaultProfileConfig()
	if strings.TrimSpace(profile.LabelPrefix) == "" {
		profile.LabelPrefix = defaults.LabelPrefix
	}
	if len(profile.IncludeKeywords) == 0 {
		profile.IncludeKeywords = defaults.IncludeKeywords
	}
	if len(profile.IncludePaths) == 0 {
		profile.IncludePaths = defaults.IncludePaths
	}
	if profile.ExcludeKeywords == nil {
		profile.ExcludeKeywords = defaults.ExcludeKeywords
	}
	if profile.MaxArtifactAgeHours <= 0 {
		profile.MaxArtifactAgeHours = defaults.MaxArtifactAgeHours
	}
	if profile.HeartbeatIntervalSec <= 0 {
		profile.HeartbeatIntervalSec = defaults.HeartbeatIntervalSec
	}
	if profile.ArtifactMaxDepth <= 0 {
		profile.ArtifactMaxDepth = defaults.ArtifactMaxDepth
	}
	if profile.MaxSessionsPerJob <= 0 {
		profile.MaxSessionsPerJob = defaults.MaxSessionsPerJob
	}
	if profile.MaxSessionInfoCalls <= 0 {
		profile.MaxSessionInfoCalls = defaults.MaxSessionInfoCalls
	}
	if profile.MaxJobInfoCalls <= 0 {
		profile.MaxJobInfoCalls = defaults.MaxJobInfoCalls
	}
	if profile.ManualTriggerMinutes <= 0 {
		profile.ManualTriggerMinutes = defaults.ManualTriggerMinutes
	}
	if profile.ArtifactPreToleranceMinutes <= 0 {
		profile.ArtifactPreToleranceMinutes = defaults.ArtifactPreToleranceMinutes
	}
	return profile
}

func (c *Collector) veeamEntries(ctx context.Context, profile api.ProfileConfig, now time.Time) []taskEntry {
	client := veeam.New()
	if !client.Available() {
		return nil
	}
	jobs, err := client.ListJobs(ctx)
	if err != nil {
		c.logf("读取 Veeam 作业列表失败：%v", err)
		return nil
	}
	sessions, err := client.ListSessions(ctx, "")
	if err != nil {
		c.logf("读取 Veeam 会话列表失败：%v", err)
		sessions = nil
	}

	byJobID := make(map[string][]veeam.Session)
	byJobName := make(map[string][]veeam.Session)
	for _, session := range sessions {
		if session.JobID != "" {
			byJobID[strings.ToLower(session.JobID)] = append(byJobID[strings.ToLower(session.JobID)], session)
		}
		if session.JobName != "" {
			byJobName[strings.ToLower(session.JobName)] = append(byJobName[strings.ToLower(session.JobName)], session)
		}
	}

	infoCalls := 0
	jobInfoCalls := 0
	entries := make([]taskEntry, 0, len(jobs))
	for _, job := range jobs {
		name := strings.TrimSpace(job.Name)
		if name == "" {
			name = strings.TrimSpace(job.ID)
		}
		if name == "" {
			continue
		}
		key := "veeam:" + strings.TrimSpace(job.ID)
		if strings.TrimSpace(job.ID) == "" {
			key = "veeam:" + name
		}
		schedule := strings.TrimSpace(job.Schedule)
		scheduleCron := ""
		if schedule != "" && cron.Validate(schedule) {
			scheduleCron = strings.Join(strings.Fields(schedule), " ")
			schedule = cron.ToHuman(scheduleCron)
		}
		configMap := map[string]any{
			"source": "veeam",
			"jobId":  job.ID,
		}
		if job.Type != "" {
			configMap["type"] = job.Type
		}
		if job.Status != "" {
			configMap["status"] = job.Status
		}
		jobInfoSchedule := ""
		if jobInfoCalls < profile.MaxJobInfoCalls && strings.TrimSpace(job.ID) != "" {
			jobInfo, jobInfoErr := client.JobInfo(ctx, job.ID)
			if jobInfoErr != nil {
				c.logf("读取 Veeam 作业详情 %s 失败：%v", job.ID, jobInfoErr)
			} else {
				jobInfoCalls++
				mergeJobInfoConfig(configMap, jobInfo)
				jobInfoSchedule = jobInfo.Schedule
			}
		}

		matched := byJobID[strings.ToLower(job.ID)]
		if len(matched) == 0 {
			matched = byJobName[strings.ToLower(name)]
		}
		matched = append([]veeam.Session(nil), matched...)
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].Creation.After(matched[j].Creation)
		})
		limit := profile.MaxSessionsPerJob
		if limit > 0 && len(matched) > limit {
			matched = matched[:limit]
		}

		if schedule == "" {
			if derivedCron, derivedHuman := deriveVeeamSchedule(matched); derivedCron != "" {
				scheduleCron, schedule = derivedCron, derivedHuman
			}
		}
		if jobInfoSchedule != "" {
			schedule = jobInfoSchedule
		}
		if schedule != "" {
			configMap["schedule"] = schedule
		}

		entry := taskEntry{
			task: api.IngestTask{
				Name:         name,
				Key:          key,
				Type:         "veeam",
				Tool:         "Veeam",
				Schedule:     schedule,
				ScheduleCron: scheduleCron,
				Config:       configMap,
			},
		}
		for index, session := range matched {
			if cached, ok := veeam.CachedSession(session.ID); ok {
				session = mergeSession(session, cached)
			} else if (index == 0 || infoCalls < profile.MaxSessionInfoCalls) && strings.TrimSpace(session.ID) != "" {
				info, infoErr := client.SessionInfo(ctx, session.ID)
				if infoErr != nil {
					c.logf("读取 Veeam 会话 %s 失败：%v", session.ID, infoErr)
				} else {
					session = mergeSession(session, info)
					veeam.RememberSession(info)
					infoCalls++
				}
			}
			execution, ok := buildVeeamExecution(name, job.ID, session, scheduleCron, profile, now)
			if ok {
				entry.executions = append(entry.executions, execution)
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

func mergeSession(base, info veeam.Session) veeam.Session {
	if info.ID != "" {
		base.ID = info.ID
	}
	if info.JobID != "" {
		base.JobID = info.JobID
	}
	if info.JobName != "" {
		base.JobName = info.JobName
	}
	if info.State != "" {
		base.State = info.State
	}
	if info.Result != "" {
		base.Result = info.Result
	}
	if info.HasStart {
		base.Start, base.HasStart = info.Start, true
	}
	if info.HasEnd {
		base.End, base.HasEnd = info.End, true
	}
	if info.ProcessedSize != "" {
		base.ProcessedSize = info.ProcessedSize
	}
	if info.OIBID != "" {
		base.OIBID = info.OIBID
	}
	if info.HasAttempt {
		base.Attempt, base.HasAttempt = info.Attempt, true
	}
	if info.HasMaxAttempts {
		base.MaxAttempts, base.HasMaxAttempts = info.MaxAttempts, true
	}
	if info.HasLastAttempt {
		base.IsLastAttempt, base.HasLastAttempt = info.IsLastAttempt, true
	}
	if info.TransferredSize != "" {
		base.TransferredSize = info.TransferredSize
	}
	if base.Creation.IsZero() && !info.Creation.IsZero() {
		base.Creation = info.Creation
	}
	return base
}

func dedupeEntries(entries []taskEntry) []taskEntry {
	seen := make(map[string]int, len(entries))
	result := make([]taskEntry, 0, len(entries))
	for _, entry := range entries {
		identity := strings.TrimSpace(entry.task.Key)
		if identity == "" {
			identity = strings.ToLower(strings.TrimSpace(entry.task.Name))
		}
		if index, exists := seen[identity]; exists {
			result[index].executions = append(result[index].executions, entry.executions...)
			continue
		}
		seen[identity] = len(result)
		result = append(result, entry)
	}
	return result
}

func ensureUniqueNames(entries []taskEntry) {
	used := make(map[string]struct{}, len(entries))
	for index := range entries {
		name := strings.TrimSpace(entries[index].task.Name)
		if name == "" {
			name = "backup-task"
		}
		if _, exists := used[name]; exists {
			suffix := shortKey(entries[index].task.Key)
			name = name + " (" + suffix + ")"
			for {
				if _, conflict := used[name]; !conflict {
					break
				}
				name += "-2"
			}
		}
		used[name] = struct{}{}
		entries[index].task.Name = name
		for execution := range entries[index].executions {
			entries[index].executions[execution].TaskName = name
		}
	}
}

func shortKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:4])
}

func mergeJobInfoConfig(config map[string]any, info veeam.JobInfo) {
	if info.Repository != "" {
		config["repository"] = info.Repository
	}
	if info.ActiveFull != "" {
		config["activeFull"] = info.ActiveFull
	}
	if info.Retention != "" {
		config["retention"] = info.Retention
	}
	if info.Compression != "" {
		config["compression"] = info.Compression
	}
	if len(info.Objects) > 0 {
		config["objects"] = info.Objects
	}
	if info.JobType != "" {
		config["jobType"] = info.JobType
	}
}
