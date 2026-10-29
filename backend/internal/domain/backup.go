// Package domain 定义备份监控域的 API 契约与归一化规则。
package domain

import (
	"encoding/json"
	"strings"
)

// HostStatus 主机状态。
type HostStatus string

const (
	HostNormal   HostStatus = "normal"
	HostAbnormal HostStatus = "abnormal"
	HostOffline  HostStatus = "offline"
)

// TaskStatus 任务状态。
type TaskStatus string

const (
	TaskNormal  TaskStatus = "normal"
	TaskRunning TaskStatus = "running"
	TaskFailed  TaskStatus = "failed"
	TaskNotRun  TaskStatus = "not_run"
	TaskNoData  TaskStatus = "no_data"
)

// ValidHostRoles 合法主机角色。
var ValidHostRoles = map[string]bool{
	"database": true, "web": true, "application": true, "file": true, "virtualization": true,
}

// Host 备份主机。
type Host struct {
	ID             string `json:"id"`
	Hostname       string `json:"hostname"`
	NodeID         string `json:"nodeId,omitempty"`
	IP             string `json:"ip"`
	OS             string `json:"os,omitempty"`
	Role           string `json:"role,omitempty"`
	AgentVersion   string `json:"agentVersion,omitempty"`
	AgentStartedAt string `json:"agentStartedAt,omitempty"`
	Status         string `json:"status"`
	LastReportTime string `json:"lastReportTime"`
	TaskCount      int    `json:"taskCount"`
	Platform       string `json:"platform,omitempty"`
	Arch           string `json:"arch,omitempty"`
	OSBuild        int    `json:"osBuild,omitempty"`
}

// HostDetail 主机详情（含任务）。
type HostDetail struct {
	Host
	Tasks []BackupTask `json:"tasks"`
}

// HostRefreshGroup 同一上报间隔下的一组主机。
type HostRefreshGroup struct {
	IntervalSec int      `json:"intervalSec"`
	HostIDs     []string `json:"hostIds"`
}

// HostRefreshPlan 管理端按主机实际上报间隔生成的刷新计划。
type HostRefreshPlan struct {
	DefaultIntervalSec int                `json:"defaultIntervalSec"`
	MinIntervalSec     int                `json:"minIntervalSec"`
	Groups             []HostRefreshGroup `json:"groups"`
}

// BackupTask 备份任务。
type BackupTask struct {
	ID                  string         `json:"id"`
	HostID              string         `json:"hostId"`
	HostName            string         `json:"hostName"`
	HostIP              string         `json:"hostIp"`
	Name                string         `json:"name"`
	TaskKey             string         `json:"taskKey,omitempty"`
	Type                string         `json:"type"`
	Tool                string         `json:"tool,omitempty"`
	Schedule            string         `json:"schedule"`
	ScheduleCron        string         `json:"scheduleCron,omitempty"`
	ScheduleLocked      bool           `json:"scheduleLocked"`
	Status              string         `json:"status"`
	LastExecutionTime   string         `json:"lastExecutionTime,omitempty"`
	LastSuccessTime     string         `json:"lastSuccessTime,omitempty"`
	ConsecutiveFailures int            `json:"consecutiveFailures"`
	SuccessRate         int            `json:"successRate"`
	LatestSize          string         `json:"latestSize,omitempty"`
	Retention           string         `json:"retention,omitempty"`
	Config              map[string]any `json:"config"`
}

// BackupExecution 备份执行记录。
type BackupExecution struct {
	ID              string `json:"id"`
	TaskID          string `json:"taskId"`
	TaskName        string `json:"taskName,omitempty"`
	HostID          string `json:"hostId,omitempty"`
	HostName        string `json:"hostName,omitempty"`
	TaskType        string `json:"taskType,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
	JobID           string `json:"jobId,omitempty"`
	OIBID           string `json:"oibId,omitempty"`
	State           string `json:"state"`
	Trigger         string `json:"trigger,omitempty"`
	CreationTime    string `json:"creationTime"`
	StartTime       string `json:"startTime,omitempty"`
	EndTime         string `json:"endTime,omitempty"`
	ProcessedSize   string `json:"processedSize,omitempty"`
	TransferredSize string `json:"transferredSize,omitempty"`
	ArtifactPath    string `json:"artifactPath,omitempty"`
	ExitCode        *int   `json:"exitCode,omitempty"`
	Attempt         *int   `json:"attempt,omitempty"`
	MaxAttempts     *int   `json:"maxAttempts,omitempty"`
	IsLastAttempt   *bool  `json:"isLastAttempt,omitempty"`
	Message         string `json:"message,omitempty"`
}

// Alert 告警。
type Alert struct {
	ID         string `json:"id"`
	HostID     string `json:"hostId"`
	TaskID     string `json:"taskId,omitempty"`
	HostName   string `json:"hostName"`
	TaskName   string `json:"taskName,omitempty"`
	TaskType   string `json:"taskType,omitempty"`
	Type       string `json:"type"`
	Level      string `json:"level"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	ResolvedAt string `json:"resolvedAt,omitempty"`
	Message    string `json:"message"`
}

// AgentGlobalConfig 下发给 Agent 的全局参数。
type AgentGlobalConfig struct {
	ReportIntervalSec int `json:"reportIntervalSec"`
	HTTPTimeoutSec    int `json:"httpTimeoutSec"`
	WSBackoffMinSec   int `json:"wsBackoffMinSec"`
	WSBackoffMaxSec   int `json:"wsBackoffMaxSec"`
}

// AppSettings 备份监控基础参数。
type AppSettings struct {
	ReportTimeoutMinutes       int               `json:"reportTimeoutMinutes"`
	OfflineTimeoutMinutes      int               `json:"offlineTimeoutMinutes"`
	NotRunGraceMinutes         int               `json:"notRunGraceMinutes"`
	ScheduleMatchWindowMinutes int               `json:"scheduleMatchWindowMinutes"`
	SchedulerIntervalSec       int               `json:"schedulerIntervalSec"`
	DedupeAlerts               bool              `json:"dedupeAlerts"`
	RetentionDays              int               `json:"retentionDays"`
	AuditRetentionDays         int               `json:"auditRetentionDays"`
	NotifyRetentionDays        int               `json:"notifyRetentionDays"`
	RequireAgentToken          bool              `json:"requireAgentToken"`
	Agent                      AgentGlobalConfig `json:"agent"`
}

// AgentFacingSettings 可安全下发给 Agent 的设置子集。
type AgentFacingSettings struct {
	ReportTimeoutMinutes       int               `json:"reportTimeoutMinutes"`
	NotRunGraceMinutes         int               `json:"notRunGraceMinutes"`
	ScheduleMatchWindowMinutes int               `json:"scheduleMatchWindowMinutes"`
	SchedulerIntervalSec       int               `json:"schedulerIntervalSec"`
	DedupeAlerts               bool              `json:"dedupeAlerts"`
	RetentionDays              int               `json:"retentionDays"`
	RequireAgentToken          bool              `json:"requireAgentToken"`
	Agent                      AgentGlobalConfig `json:"agent"`
}

// DefaultAppSettings 返回基础参数默认值。
func DefaultAppSettings() AppSettings {
	return AppSettings{
		ReportTimeoutMinutes: 3, OfflineTimeoutMinutes: 6, NotRunGraceMinutes: 30,
		ScheduleMatchWindowMinutes: 60, SchedulerIntervalSec: 30,
		DedupeAlerts: true, RetentionDays: 90, AuditRetentionDays: 90, NotifyRetentionDays: 90, RequireAgentToken: false,
		Agent: AgentGlobalConfig{ReportIntervalSec: 60},
	}
}

// Sections 按基础配置区块拆分基础参数，供 basic_config 各行分别存储。
func (s AppSettings) Sections() map[string]map[string]any {
	return map[string]map[string]any{
		BaseSectionThresholds: {
			"reportTimeoutMinutes":       s.ReportTimeoutMinutes,
			"offlineTimeoutMinutes":      s.OfflineTimeoutMinutes,
			"notRunGraceMinutes":         s.NotRunGraceMinutes,
			"scheduleMatchWindowMinutes": s.ScheduleMatchWindowMinutes,
			"schedulerIntervalSec":       s.SchedulerIntervalSec,
			"retentionDays":              s.RetentionDays,
			"auditRetentionDays":         s.AuditRetentionDays,
			"notifyRetentionDays":        s.NotifyRetentionDays,
		},
		BaseSectionAgent: {
			"reportIntervalSec": s.Agent.ReportIntervalSec,
			"httpTimeoutSec":    s.Agent.HTTPTimeoutSec,
			"wsBackoffMinSec":   s.Agent.WSBackoffMinSec,
			"wsBackoffMaxSec":   s.Agent.WSBackoffMaxSec,
		},
		BaseSectionAlarm: {
			"dedupeAlerts":      s.DedupeAlerts,
			"requireAgentToken": s.RequireAgentToken,
		},
	}
}

// MergeAppSettingsSections 将基础配置各区块合并为基础参数，缺失字段回填默认值。
func MergeAppSettingsSections(sections map[string]map[string]any) AppSettings {
	merged := map[string]any{}
	for name, section := range sections {
		if name == BaseSectionAgent {
			merged["agent"] = section
			continue
		}
		for key, value := range section {
			merged[key] = value
		}
	}
	return NormalizeAppSettings(merged)
}

const (
	maxReportTimeoutMinutes       = 1440
	maxOfflineTimeoutMinutes      = 4320
	maxNotRunGraceMinutes         = 1440
	maxScheduleMatchWindowMinutes = 1440
	maxSchedulerIntervalSec       = 600
	maxRetentionDays              = 3650
	maxAuditRetentionDays         = 365
	maxAgentReportIntervalSec     = 86400
	maxAgentHTTPTimeoutSec        = 600
	maxAgentWSBackoffMinSec       = 120
	maxAgentWSBackoffMaxSec       = 600
)

// NormalizeAppSettings 将任意输入归一化为合法设置。
func NormalizeAppSettings(raw map[string]any) AppSettings {
	d := DefaultAppSettings()
	out := d
	out.ReportTimeoutMinutes = boundedInt(raw["reportTimeoutMinutes"], d.ReportTimeoutMinutes, 1, maxReportTimeoutMinutes)
	out.OfflineTimeoutMinutes = boundedInt(raw["offlineTimeoutMinutes"], d.OfflineTimeoutMinutes, 1, maxOfflineTimeoutMinutes)
	if out.OfflineTimeoutMinutes < out.ReportTimeoutMinutes {
		out.OfflineTimeoutMinutes = out.ReportTimeoutMinutes * 3
		if out.OfflineTimeoutMinutes > maxOfflineTimeoutMinutes {
			out.OfflineTimeoutMinutes = maxOfflineTimeoutMinutes
		}
	}
	out.NotRunGraceMinutes = boundedInt(raw["notRunGraceMinutes"], d.NotRunGraceMinutes, 1, maxNotRunGraceMinutes)
	out.ScheduleMatchWindowMinutes = boundedInt(raw["scheduleMatchWindowMinutes"], d.ScheduleMatchWindowMinutes, 1, maxScheduleMatchWindowMinutes)
	out.SchedulerIntervalSec = boundedInt(raw["schedulerIntervalSec"], d.SchedulerIntervalSec, 5, maxSchedulerIntervalSec)
	out.RetentionDays = boundedInt(raw["retentionDays"], d.RetentionDays, 0, maxRetentionDays)
	out.AuditRetentionDays = boundedInt(raw["auditRetentionDays"], d.AuditRetentionDays, 0, maxAuditRetentionDays)
	out.NotifyRetentionDays = boundedInt(raw["notifyRetentionDays"], d.NotifyRetentionDays, 0, maxRetentionDays)
	out.DedupeAlerts = boolValue(raw["dedupeAlerts"], d.DedupeAlerts)
	out.RequireAgentToken = boolValue(raw["requireAgentToken"], d.RequireAgentToken)
	if agent, ok := raw["agent"].(map[string]any); ok {
		out.Agent.ReportIntervalSec = boundedInt(agent["reportIntervalSec"], d.Agent.ReportIntervalSec, 10, maxAgentReportIntervalSec)
		out.Agent.HTTPTimeoutSec = boundedInt(agent["httpTimeoutSec"], 0, 0, maxAgentHTTPTimeoutSec)
		out.Agent.WSBackoffMinSec = boundedInt(agent["wsBackoffMinSec"], 0, 0, maxAgentWSBackoffMinSec)
		out.Agent.WSBackoffMaxSec = boundedInt(agent["wsBackoffMaxSec"], 0, 0, maxAgentWSBackoffMaxSec)
	}
	if out.Agent.WSBackoffMaxSec > 0 && out.Agent.WSBackoffMaxSec < out.Agent.WSBackoffMinSec {
		out.Agent.WSBackoffMaxSec = out.Agent.WSBackoffMinSec
	}
	return out
}

// ValidateAppSettings 校验保存前的语义约束，返回空字符串表示通过。
func ValidateAppSettings(raw map[string]any) string {
	d := DefaultAppSettings()
	report := boundedInt(raw["reportTimeoutMinutes"], d.ReportTimeoutMinutes, 1, maxReportTimeoutMinutes)
	offline := boundedInt(raw["offlineTimeoutMinutes"], d.OfflineTimeoutMinutes, 1, maxOfflineTimeoutMinutes)
	if offline < report {
		return "主机离线阈值不能小于主机上报超时"
	}
	if agentRaw, ok := raw["agent"].(map[string]any); ok {
		wsMin := nonNegativeInt(agentRaw["wsBackoffMinSec"], d.Agent.WSBackoffMinSec)
		wsMax := nonNegativeInt(agentRaw["wsBackoffMaxSec"], d.Agent.WSBackoffMaxSec)
		if wsMin == 0 {
			wsMin = d.Agent.WSBackoffMinSec
		}
		if wsMax == 0 {
			wsMax = d.Agent.WSBackoffMaxSec
		}
		if wsMax < wsMin {
			return "WS 重连退避上限不能小于下限"
		}
	}
	return ""
}

func boundedInt(value any, fallback, min, max int) int {
	n := positiveInt(value, fallback, min)
	if n > max {
		return max
	}
	return n
}

// AgentFacing 提取可下发给 Agent 的字段。
func (s AppSettings) AgentFacing() AgentFacingSettings {
	return AgentFacingSettings{
		ReportTimeoutMinutes: s.ReportTimeoutMinutes, NotRunGraceMinutes: s.NotRunGraceMinutes,
		ScheduleMatchWindowMinutes: s.ScheduleMatchWindowMinutes, SchedulerIntervalSec: s.SchedulerIntervalSec,
		DedupeAlerts: s.DedupeAlerts, RetentionDays: s.RetentionDays,
		RequireAgentToken: s.RequireAgentToken, Agent: s.Agent,
	}
}

// AgentProfileConfig 采集配置项。
type AgentProfileConfig struct {
	ReportIntervalSec           int      `json:"reportIntervalSec"`
	CollectCrontab              bool     `json:"collectCrontab"`
	CollectVeeam                bool     `json:"collectVeeam"`
	ArtifactCheck               bool     `json:"artifactCheck"`
	LabelPrefix                 string   `json:"labelPrefix"`
	IncludeKeywords             []string `json:"includeKeywords"`
	IncludePaths                []string `json:"includePaths"`
	ExcludeKeywords             []string `json:"excludeKeywords"`
	MaxArtifactAgeHours         int      `json:"maxArtifactAgeHours"`
	HeartbeatIntervalSec        int      `json:"heartbeatIntervalSec"`
	ArtifactMaxDepth            int      `json:"artifactMaxDepth"`
	MaxSessionsPerJob           int      `json:"maxSessionsPerJob"`
	MaxSessionInfoCalls         int      `json:"maxSessionInfoCalls"`
	MaxJobInfoCalls             int      `json:"maxJobInfoCalls"`
	ManualTriggerMinutes        int      `json:"manualTriggerMinutes"`
	ArtifactPreToleranceMinutes int      `json:"artifactPreToleranceMinutes"`
	AutoUpdateMinutes           int      `json:"autoUpdateMinutes"`
}

// DefaultProfileConfig 默认采集配置。
func DefaultProfileConfig() AgentProfileConfig {
	return AgentProfileConfig{
		CollectCrontab: true, CollectVeeam: true, ArtifactCheck: true,
		LabelPrefix:         "# backup-monitor:",
		IncludeKeywords:     []string{"backup", "dump", "mysqldump", "xtrabackup", "pg_dump", "rsync", "rclone", "borg", "tar", "veeam"},
		IncludePaths:        []string{"/data/scripts", "/opt/backup", "/usr/local/bin"},
		ExcludeKeywords:     []string{"logrotate", "ntpdate", "certbot", "apt-get", "yum ", "/usr/lib"},
		MaxArtifactAgeHours: 30, HeartbeatIntervalSec: 30, ArtifactMaxDepth: 3,
		MaxSessionsPerJob: 5, MaxSessionInfoCalls: 10, MaxJobInfoCalls: 20,
		ManualTriggerMinutes: 30, ArtifactPreToleranceMinutes: 10,
	}
}

// NormalizeProfileConfig 归一化采集配置。
func NormalizeProfileConfig(raw map[string]any) AgentProfileConfig {
	d := DefaultProfileConfig()
	out := d
	out.ReportIntervalSec = nonNegativeInt(raw["reportIntervalSec"], d.ReportIntervalSec)
	out.CollectCrontab = boolValue(raw["collectCrontab"], d.CollectCrontab)
	out.CollectVeeam = boolValue(raw["collectVeeam"], d.CollectVeeam)
	out.ArtifactCheck = boolValue(raw["artifactCheck"], d.ArtifactCheck)
	if v, ok := raw["labelPrefix"].(string); ok {
		out.LabelPrefix = v
	}
	out.IncludeKeywords = stringSlice(raw["includeKeywords"], d.IncludeKeywords)
	out.IncludePaths = stringSlice(raw["includePaths"], d.IncludePaths)
	out.ExcludeKeywords = stringSlice(raw["excludeKeywords"], d.ExcludeKeywords)
	out.MaxArtifactAgeHours = positiveInt(raw["maxArtifactAgeHours"], d.MaxArtifactAgeHours, 1)
	out.HeartbeatIntervalSec = positiveInt(raw["heartbeatIntervalSec"], d.HeartbeatIntervalSec, 1)
	out.ArtifactMaxDepth = positiveInt(raw["artifactMaxDepth"], d.ArtifactMaxDepth, 1)
	out.MaxSessionsPerJob = nonNegativeInt(raw["maxSessionsPerJob"], d.MaxSessionsPerJob)
	out.MaxSessionInfoCalls = nonNegativeInt(raw["maxSessionInfoCalls"], d.MaxSessionInfoCalls)
	out.MaxJobInfoCalls = nonNegativeInt(raw["maxJobInfoCalls"], d.MaxJobInfoCalls)
	out.ManualTriggerMinutes = nonNegativeInt(raw["manualTriggerMinutes"], d.ManualTriggerMinutes)
	out.ArtifactPreToleranceMinutes = nonNegativeInt(raw["artifactPreToleranceMinutes"], d.ArtifactPreToleranceMinutes)
	out.AutoUpdateMinutes = nonNegativeInt(raw["autoUpdateMinutes"], d.AutoUpdateMinutes)
	return out
}

// AgentProfile 采集配置。
type AgentProfile struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Hostname    string             `json:"hostname"`
	Description string             `json:"description,omitempty"`
	Enabled     bool               `json:"enabled"`
	Priority    int                `json:"priority"`
	Config      AgentProfileConfig `json:"config"`
	CreatedAt   string             `json:"createdAt"`
	UpdatedAt   string             `json:"updatedAt"`
}

// AgentToken Agent 访问令牌。
type AgentToken struct {
	ID         string `json:"id"`
	Token      string `json:"token"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname,omitempty"`
	Enabled    bool   `json:"enabled"`
	LastSeenAt string `json:"lastSeenAt,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// NotifyChannel 通知渠道。
type NotifyChannel struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	EventType    []string       `json:"eventType"`
	Level        string         `json:"level"`
	RemainDays   int            `json:"remainDays"`
	TriggerGroup string         `json:"triggerGroup"`
	Remark       string         `json:"remark,omitempty"`
	Enabled      bool           `json:"enabled"`
	Recover      bool           `json:"recover"`
	Config       map[string]any `json:"config"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

// NotificationRecord 通知发送记录。
type NotificationRecord struct {
	ID          string `json:"id"`
	ChannelID   string `json:"channelId,omitempty"`
	Channel     string `json:"channel"`
	ChannelName string `json:"channelName,omitempty"`
	Type        string `json:"type"`
	Level       string `json:"level,omitempty"`
	Recipients  string `json:"recipients,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	HostName    string `json:"hostName,omitempty"`
	TaskName    string `json:"taskName,omitempty"`
	Message     string `json:"message,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// StatusSummary 任务状态分布。
type StatusSummary struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// MethodSummary 备份方式汇总。
type MethodSummary struct {
	Method string `json:"method"`
	Total  int    `json:"total"`
	Normal int    `json:"normal"`
	Failed int    `json:"failed"`
	NotRun int    `json:"notRun"`
}

// TrendPoint 每日执行趋势。
type TrendPoint struct {
	Date    string `json:"date"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
	Other   int    `json:"other"`
}

// Dashboard 概览数据。
type Dashboard struct {
	HostCount           int             `json:"hostCount"`
	HostNormal          int             `json:"hostNormal"`
	HostAbnormal        int             `json:"hostAbnormal"`
	HostOffline         int             `json:"hostOffline"`
	TaskCount           int             `json:"taskCount"`
	TodaySuccess        int             `json:"todaySuccess"`
	TodayFailed         int             `json:"todayFailed"`
	TodayTotal          int             `json:"todayTotal"`
	SuccessRate         int             `json:"successRate"`
	AbnormalTasks       int             `json:"abnormalTasks"`
	NotRunTasks         int             `json:"notRunTasks"`
	ActiveAlerts        int             `json:"activeAlerts"`
	StatusSummary       []StatusSummary `json:"statusSummary"`
	MethodSummary       []MethodSummary `json:"methodSummary"`
	Trend               []TrendPoint    `json:"trend"`
	RecentAbnormalTasks []BackupTask    `json:"recentAbnormalTasks"`
	RecentAlerts        []Alert         `json:"recentAlerts"`
}

// IngestHost Agent 上报的主机信息。
type IngestHost struct {
	Hostname       string `json:"hostname"`
	NodeID         string `json:"nodeId"`
	IP             string `json:"ip"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	OSBuild        int    `json:"osBuild"`
	Role           string `json:"role"`
	AgentVersion   string `json:"agentVersion"`
	AgentStartedAt string `json:"agentStartedAt"`
}

// IngestTask Agent 上报的任务。
type IngestTask struct {
	Name         string         `json:"name"`
	Key          string         `json:"key"`
	Type         string         `json:"type"`
	Tool         string         `json:"tool"`
	Schedule     string         `json:"schedule"`
	ScheduleCron string         `json:"scheduleCron"`
	Config       map[string]any `json:"config"`
}

// IngestExecution Agent 上报的执行记录。
type IngestExecution struct {
	TaskName        string `json:"taskName"`
	SessionID       string `json:"sessionId"`
	JobID           string `json:"jobId"`
	OIBID           string `json:"oibId"`
	State           string `json:"state"`
	Trigger         string `json:"trigger"`
	CreationTime    string `json:"creationTime"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
	ProcessedSize   string `json:"processedSize"`
	TransferredSize string `json:"transferredSize"`
	ArtifactPath    string `json:"artifactPath"`
	ExitCode        *int   `json:"exitCode"`
	Attempt         *int   `json:"attempt"`
	MaxAttempts     *int   `json:"maxAttempts"`
	IsLastAttempt   *bool  `json:"isLastAttempt"`
	Message         string `json:"message"`
}

// IngestReport 上报请求体。
type IngestReport struct {
	Host       IngestHost        `json:"host"`
	Tasks      []IngestTask      `json:"tasks"`
	Executions []IngestExecution `json:"executions"`
}

// AgentConfigResponse 配置拉取响应。
type AgentConfigResponse struct {
	Hostname   string              `json:"hostname"`
	ServerTime string              `json:"serverTime"`
	Settings   AgentFacingSettings `json:"settings"`
	Profile    *AgentProfile       `json:"profile"`
}

// ParseJSONObject 解析 JSON 对象，失败返回空对象。
func ParseJSONObject(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func positiveInt(value any, fallback, min int) int {
	n, ok := numberValue(value)
	if !ok || n < float64(min) {
		return fallback
	}
	return int(n)
}

func nonNegativeInt(value any, fallback int) int {
	n, ok := numberValue(value)
	if !ok || n < 0 {
		return fallback
	}
	return int(n)
}

func numberValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

func boolValue(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return fallback
}

func stringSlice(value any, fallback []string) []string {
	list, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return typed
		}
		return fallback
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}
