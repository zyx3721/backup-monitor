package swagger

// swaggerHost 备份主机。
type swaggerHost struct {
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
}

// swaggerHostDetail 主机详情（含任务列表）。
type swaggerHostDetail struct {
	swaggerHost
	Tasks []swaggerTask `json:"tasks"`
}

// swaggerHostRefreshGroup 同一上报间隔下的一组主机。
type swaggerHostRefreshGroup struct {
	IntervalSec int      `json:"intervalSec"`
	HostIDs     []string `json:"hostIds"`
}

// swaggerHostRefreshPlan 主机刷新计划。
type swaggerHostRefreshPlan struct {
	DefaultIntervalSec int                       `json:"defaultIntervalSec"`
	MinIntervalSec     int                       `json:"minIntervalSec"`
	Groups             []swaggerHostRefreshGroup `json:"groups"`
}

// swaggerTask 备份任务。
type swaggerTask struct {
	ID           string `json:"id"`
	HostID       string `json:"hostId"`
	HostName     string `json:"hostName"`
	HostIP       string `json:"hostIp"`
	Name         string `json:"name"`
	TaskKey      string `json:"taskKey,omitempty"`
	Type         string `json:"type"`
	Tool         string `json:"tool,omitempty"`
	Schedule     string `json:"schedule"`
	ScheduleCron string `json:"scheduleCron,omitempty"`
	Status       string `json:"status"`
}

// swaggerExecution 执行记录。
type swaggerExecution struct {
	ID              string `json:"id"`
	TaskID          string `json:"taskId"`
	TaskName        string `json:"taskName"`
	HostID          string `json:"hostId"`
	HostName        string `json:"hostName"`
	TaskType        string `json:"taskType"`
	SessionID       string `json:"sessionId,omitempty"`
	State           string `json:"state"`
	Trigger         string `json:"trigger"`
	CreationTime    string `json:"creationTime"`
	StartTime       string `json:"startTime,omitempty"`
	EndTime         string `json:"endTime,omitempty"`
	ProcessedSize   string `json:"processedSize,omitempty"`
	TransferredSize string `json:"transferredSize,omitempty"`
	ArtifactPath    string `json:"artifactPath,omitempty"`
	ExitCode        int    `json:"exitCode"`
	Message         string `json:"message,omitempty"`
}

// swaggerAlert 告警事件。
type swaggerAlert struct {
	ID         string `json:"id"`
	HostID     string `json:"hostId"`
	HostName   string `json:"hostName"`
	TaskID     string `json:"taskId,omitempty"`
	TaskName   string `json:"taskName,omitempty"`
	TaskType   string `json:"taskType,omitempty"`
	Type       string `json:"type"`
	Level      string `json:"level"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	CreatedAt  string `json:"createdAt"`
	ResolvedAt string `json:"resolvedAt,omitempty"`
}

// swaggerAgentProfile 采集配置。
type swaggerAgentProfile struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Hostname    string                    `json:"hostname"`
	Description string                    `json:"description,omitempty"`
	Enabled     bool                      `json:"enabled"`
	Priority    int                       `json:"priority"`
	Config      swaggerAgentProfileConfig `json:"config"`
	CreatedAt   string                    `json:"createdAt"`
	UpdatedAt   string                    `json:"updatedAt"`
}

// swaggerAgentProfileConfig 采集配置明细。
type swaggerAgentProfileConfig struct {
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

// swaggerAgentToken Agent 访问令牌。
type swaggerAgentToken struct {
	ID         string `json:"id"`
	Token      string `json:"token,omitempty"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname,omitempty"`
	Enabled    bool   `json:"enabled"`
	LastSeenAt string `json:"lastSeenAt,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// swaggerNotifyChannel 通知渠道。
type swaggerNotifyChannel struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	EventType    []string               `json:"eventType"`
	Level        string                 `json:"level"`
	RemainDays   int                    `json:"remainDays"`
	TriggerGroup string                 `json:"triggerGroup"`
	Remark       string                 `json:"remark,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Recover      bool                   `json:"recover"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

// swaggerNotificationRecord 通知发送记录。
type swaggerNotificationRecord struct {
	ID          string `json:"id"`
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

// swaggerAgentBinaryMeta 托管的 Agent 二进制元数据。
type swaggerAgentBinaryMeta struct {
	Platform   string `json:"platform"`
	Arch       string `json:"arch"`
	Variant    string `json:"variant"`
	GoVersion  string `json:"goVersion"`
	Version    string `json:"version"`
	Sha256     string `json:"sha256"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

// swaggerHostRoleRequest 修改主机角色请求。
type swaggerHostRoleRequest struct {
	Role string `json:"role"`
}

// swaggerTaskScheduleRequest 修改执行计划请求。
type swaggerTaskScheduleRequest struct {
	ScheduleCron string `json:"scheduleCron"`
}

// swaggerAgentProfileRequest 创建或更新采集配置请求。
type swaggerAgentProfileRequest struct {
	Name        string                 `json:"name"`
	Hostname    string                 `json:"hostname"`
	Description string                 `json:"description"`
	Enabled     *bool                  `json:"enabled"`
	Priority    *int                   `json:"priority"`
	Config      map[string]interface{} `json:"config"`
}

// swaggerAgentTokenRequest 更新 Agent 令牌请求。
type swaggerAgentTokenRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Enabled  *bool  `json:"enabled"`
}

// swaggerAgentTokenCreateRequest 创建 Agent 令牌请求，token 留空时自动生成。
type swaggerAgentTokenCreateRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Token    string `json:"token"`
}

// swaggerAgentTokenCreateResponse 创建 Agent 令牌响应（含明文令牌）。
type swaggerAgentTokenCreateResponse struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

// swaggerAgentBinaryLatestResponse 最新 Agent 二进制信息。
type swaggerAgentBinaryLatestResponse struct {
	Version   string `json:"version"`
	Sha256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Arch      string `json:"arch"`
	Variant   string `json:"variant"`
	GoVersion string `json:"goVersion"`
}

// swaggerAgentBinaryPushRequest 推送 Agent 自更新请求。
type swaggerAgentBinaryPushRequest struct {
	Platform string `json:"platform"`
	All      bool   `json:"all"`
	IPs      string `json:"ips,omitempty"`
}

// swaggerHistoryExportRequest 数据导出审计请求。
type swaggerHistoryExportRequest struct {
	Module string `json:"module"`
	Count  int    `json:"count"`
}
