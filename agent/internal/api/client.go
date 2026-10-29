// Package api 实现服务端 Agent 协议中的配置、上报与二进制下载请求。
package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backup-monitor-agent/internal/config"
)

const tokenHeader = "X-Agent-Token"

// ErrNoBinary 表示服务端尚未托管 Agent 二进制。
var ErrNoBinary = errors.New("no agent binary has been uploaded")

// HTTPError 表示服务端返回了非成功状态码。
type HTTPError struct {
	StatusCode int
	Body       string
}

// Error 返回英文错误描述。
func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("服务端返回 HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("服务端返回 HTTP %d：%s", e.StatusCode, e.Body)
}

// Client 是带令牌与超时的服务端 HTTP 客户端。
type Client struct {
	base   *url.URL
	token  string
	client *http.Client
}

// New 创建 HTTP 客户端，超时为零时使用默认值。
func New(serverURL, token string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil {
		return nil, fmt.Errorf("解析服务端地址失败：%w", err)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("服务端地址无效：必须是 http 或 https 绝对地址")
	}
	if timeout <= 0 {
		timeout = time.Duration(config.DefaultHTTPTimeoutSec) * time.Second
	}
	return &Client{
		base:   parsed,
		token:  token,
		client: &http.Client{Timeout: timeout},
	}, nil
}

// Config 拉取指定主机的采集配置。
func (c *Client) Config(ctx context.Context, hostname, platform string) (*ConfigResponse, error) {
	query := url.Values{}
	query.Set("hostname", hostname)
	query.Set("platform", platform)
	path := "/api/agent/config?" + query.Encode()

	var response ConfigResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, fmt.Errorf("拉取 Agent 配置失败：%w", err)
	}
	return &response, nil
}

// Report 上报主机、任务与执行记录。
func (c *Client) Report(ctx context.Context, report IngestReport) (*ReportResponse, error) {
	var response ReportResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/ingest/report", report, &response); err != nil {
		return nil, fmt.Errorf("提交上报失败：%w", err)
	}
	return &response, nil
}

// LatestBinary 查询服务端托管的最新二进制信息。
func (c *Client) LatestBinary(ctx context.Context, platform, arch string, osBuild int) (*BinaryInfo, error) {
	query := url.Values{}
	query.Set("platform", platform)
	if strings.TrimSpace(arch) != "" {
		query.Set("arch", arch)
	}
	if osBuild > 0 {
		query.Set("osBuild", strconv.Itoa(osBuild))
	}
	path := "/api/agent/binary/latest?" + query.Encode()

	request, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("查询最新 Agent 版本失败：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, ErrNoBinary
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, responseError(response)
	}
	var info BinaryInfo
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode latest binary response: %w", err)
	}
	return &info, nil
}

// Download 下载二进制到 dest 并返回版本、校验和与大小。
func (c *Client) Download(ctx context.Context, rawURL, dest string) (*DownloadResult, error) {
	request, err := c.newRequest(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("下载 Agent 程序失败：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, responseError(response)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, fmt.Errorf("prepare download directory: %w", err)
	}

	file, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("create download file: %w", err)
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(dest)
		return nil, fmt.Errorf("write download file: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(dest)
		return nil, fmt.Errorf("close download file: %w", closeErr)
	}

	return &DownloadResult{
		Version: response.Header.Get("X-Agent-Binary-Version"),
		SHA256:  hex.EncodeToString(hash.Sum(nil)),
		Size:    size,
		Header:  response.Header.Get("X-Agent-Binary-Sha256"),
	}, nil
}

func (c *Client) doJSON(ctx context.Context, method, rawURL string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := c.newRequest(ctx, method, rawURL, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return responseError(response)
	}
	if out == nil {
		io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) newRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	target, err := c.resolve(rawURL)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set(tokenHeader, c.token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "backup-monitor-agent")
	return request, nil
}

func (c *Client) resolve(rawURL string) (string, error) {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return "", fmt.Errorf("empty request URL")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse request URL: %w", err)
	}
	if parsed.IsAbs() {
		return parsed.String(), nil
	}
	return c.base.ResolveReference(parsed).String(), nil
}

func responseError(response *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return &HTTPError{StatusCode: response.StatusCode, Body: strings.TrimSpace(string(data))}
}

// ConfigResponse 是 GET /api/agent/config 的响应。
type ConfigResponse struct {
	Hostname   string   `json:"hostname"`
	ServerTime string   `json:"serverTime"`
	Settings   Settings `json:"settings"`
	Profile    *Profile `json:"profile"`
}

// Settings 是服务端下发的全局设置子集。
type Settings struct {
	ReportTimeoutMinutes       int           `json:"reportTimeoutMinutes"`
	NotRunGraceMinutes         int           `json:"notRunGraceMinutes"`
	ScheduleMatchWindowMinutes int           `json:"scheduleMatchWindowMinutes"`
	SchedulerIntervalSec       int           `json:"schedulerIntervalSec"`
	DedupeAlerts               bool          `json:"dedupeAlerts"`
	RetentionDays              int           `json:"retentionDays"`
	RequireAgentToken          bool          `json:"requireAgentToken"`
	Agent                      AgentSettings `json:"agent"`
}

// AgentSettings 是服务端下发的 Agent 全局参数。
type AgentSettings struct {
	ReportIntervalSec int `json:"reportIntervalSec"`
	HTTPTimeoutSec    int `json:"httpTimeoutSec"`
	WSBackoffMinSec   int `json:"wsBackoffMinSec"`
	WSBackoffMaxSec   int `json:"wsBackoffMaxSec"`
}

// Profile 是匹配当前主机的采集配置。
type Profile struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Hostname    string        `json:"hostname"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Priority    int           `json:"priority"`
	Config      ProfileConfig `json:"config"`
}

// ProfileConfig 是采集行为参数。
type ProfileConfig struct {
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

// DefaultProfileConfig 返回与服务端一致的采集配置默认值。
func DefaultProfileConfig() ProfileConfig {
	return ProfileConfig{
		CollectCrontab:              true,
		CollectVeeam:                true,
		ArtifactCheck:               true,
		LabelPrefix:                 "# backup-monitor:",
		IncludeKeywords:             []string{"backup", "dump", "mysqldump", "xtrabackup", "pg_dump", "rsync", "rclone", "borg", "tar", "veeam"},
		IncludePaths:                []string{"/data/scripts", "/opt/backup", "/usr/local/bin"},
		ExcludeKeywords:             []string{"logrotate", "ntpdate", "certbot", "apt-get", "yum ", "/usr/lib"},
		MaxArtifactAgeHours:         config.DefaultArtifactMaxAgeHours,
		HeartbeatIntervalSec:        config.DefaultHeartbeatIntervalSec,
		ArtifactMaxDepth:            config.DefaultArtifactMaxDepth,
		MaxSessionsPerJob:           config.DefaultMaxSessionsPerJob,
		MaxSessionInfoCalls:         config.DefaultMaxSessionInfoCalls,
		MaxJobInfoCalls:             config.DefaultMaxJobInfoCalls,
		ManualTriggerMinutes:        config.DefaultManualTriggerMinutes,
		ArtifactPreToleranceMinutes: config.DefaultPreToleranceMinutes,
	}
}

type profileConfigWire struct {
	ReportIntervalSec           *int      `json:"reportIntervalSec"`
	CollectCrontab              *bool     `json:"collectCrontab"`
	CollectVeeam                *bool     `json:"collectVeeam"`
	ArtifactCheck               *bool     `json:"artifactCheck"`
	LabelPrefix                 *string   `json:"labelPrefix"`
	IncludeKeywords             *[]string `json:"includeKeywords"`
	IncludePaths                *[]string `json:"includePaths"`
	ExcludeKeywords             *[]string `json:"excludeKeywords"`
	MaxArtifactAgeHours         *int      `json:"maxArtifactAgeHours"`
	HeartbeatIntervalSec        *int      `json:"heartbeatIntervalSec"`
	ArtifactMaxDepth            *int      `json:"artifactMaxDepth"`
	MaxSessionsPerJob           *int      `json:"maxSessionsPerJob"`
	MaxSessionInfoCalls         *int      `json:"maxSessionInfoCalls"`
	MaxJobInfoCalls             *int      `json:"maxJobInfoCalls"`
	ManualTriggerMinutes        *int      `json:"manualTriggerMinutes"`
	ArtifactPreToleranceMinutes *int      `json:"artifactPreToleranceMinutes"`
	AutoUpdateMinutes           *int      `json:"autoUpdateMinutes"`
}

// UnmarshalJSON 为缺失字段补齐默认值。
func (p *ProfileConfig) UnmarshalJSON(data []byte) error {
	var wire profileConfigWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	result := DefaultProfileConfig()
	if wire.ReportIntervalSec != nil {
		result.ReportIntervalSec = nonNegative(*wire.ReportIntervalSec)
	}
	if wire.CollectCrontab != nil {
		result.CollectCrontab = *wire.CollectCrontab
	}
	if wire.CollectVeeam != nil {
		result.CollectVeeam = *wire.CollectVeeam
	}
	if wire.ArtifactCheck != nil {
		result.ArtifactCheck = *wire.ArtifactCheck
	}
	if wire.LabelPrefix != nil {
		result.LabelPrefix = *wire.LabelPrefix
	}
	if wire.IncludeKeywords != nil {
		result.IncludeKeywords = cleanList(*wire.IncludeKeywords)
	}
	if wire.IncludePaths != nil {
		result.IncludePaths = cleanList(*wire.IncludePaths)
	}
	if wire.ExcludeKeywords != nil {
		result.ExcludeKeywords = cleanList(*wire.ExcludeKeywords)
	}
	if wire.MaxArtifactAgeHours != nil {
		result.MaxArtifactAgeHours = positive(*wire.MaxArtifactAgeHours, config.DefaultArtifactMaxAgeHours)
	}
	if wire.HeartbeatIntervalSec != nil {
		result.HeartbeatIntervalSec = positive(*wire.HeartbeatIntervalSec, config.DefaultHeartbeatIntervalSec)
	}
	if wire.ArtifactMaxDepth != nil {
		result.ArtifactMaxDepth = positive(*wire.ArtifactMaxDepth, config.DefaultArtifactMaxDepth)
	}
	if wire.MaxSessionsPerJob != nil {
		result.MaxSessionsPerJob = positive(*wire.MaxSessionsPerJob, config.DefaultMaxSessionsPerJob)
	}
	if wire.MaxSessionInfoCalls != nil {
		result.MaxSessionInfoCalls = positive(*wire.MaxSessionInfoCalls, config.DefaultMaxSessionInfoCalls)
	}
	if wire.MaxJobInfoCalls != nil {
		result.MaxJobInfoCalls = positive(*wire.MaxJobInfoCalls, config.DefaultMaxJobInfoCalls)
	}
	if wire.ManualTriggerMinutes != nil {
		result.ManualTriggerMinutes = positive(*wire.ManualTriggerMinutes, config.DefaultManualTriggerMinutes)
	}
	if wire.ArtifactPreToleranceMinutes != nil {
		result.ArtifactPreToleranceMinutes = positive(*wire.ArtifactPreToleranceMinutes, config.DefaultPreToleranceMinutes)
	}
	if wire.AutoUpdateMinutes != nil {
		result.AutoUpdateMinutes = nonNegative(*wire.AutoUpdateMinutes)
	}
	*p = result
	return nil
}

type settingsWire struct {
	ReportTimeoutMinutes       *int       `json:"reportTimeoutMinutes"`
	NotRunGraceMinutes         *int       `json:"notRunGraceMinutes"`
	ScheduleMatchWindowMinutes *int       `json:"scheduleMatchWindowMinutes"`
	SchedulerIntervalSec       *int       `json:"schedulerIntervalSec"`
	DedupeAlerts               *bool      `json:"dedupeAlerts"`
	RetentionDays              *int       `json:"retentionDays"`
	RequireAgentToken          *bool      `json:"requireAgentToken"`
	Agent                      *agentWire `json:"agent"`
}

type agentWire struct {
	ReportIntervalSec *int `json:"reportIntervalSec"`
	HTTPTimeoutSec    *int `json:"httpTimeoutSec"`
	WSBackoffMinSec   *int `json:"wsBackoffMinSec"`
	WSBackoffMaxSec   *int `json:"wsBackoffMaxSec"`
}

// UnmarshalJSON 为缺失的全局设置补齐默认值。
func (s *Settings) UnmarshalJSON(data []byte) error {
	var wire settingsWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	result := Settings{
		ReportTimeoutMinutes:       30,
		NotRunGraceMinutes:         30,
		ScheduleMatchWindowMinutes: 60,
		SchedulerIntervalSec:       30,
		DedupeAlerts:               true,
		RetentionDays:              90,
	}
	if wire.ReportTimeoutMinutes != nil {
		result.ReportTimeoutMinutes = nonNegative(*wire.ReportTimeoutMinutes)
	}
	if wire.NotRunGraceMinutes != nil {
		result.NotRunGraceMinutes = nonNegative(*wire.NotRunGraceMinutes)
	}
	if wire.ScheduleMatchWindowMinutes != nil {
		result.ScheduleMatchWindowMinutes = nonNegative(*wire.ScheduleMatchWindowMinutes)
	}
	if wire.SchedulerIntervalSec != nil {
		result.SchedulerIntervalSec = nonNegative(*wire.SchedulerIntervalSec)
	}
	if wire.DedupeAlerts != nil {
		result.DedupeAlerts = *wire.DedupeAlerts
	}
	if wire.RetentionDays != nil {
		result.RetentionDays = nonNegative(*wire.RetentionDays)
	}
	if wire.RequireAgentToken != nil {
		result.RequireAgentToken = *wire.RequireAgentToken
	}
	result.Agent = AgentSettings{}
	if wire.Agent != nil {
		if wire.Agent.ReportIntervalSec != nil {
			result.Agent.ReportIntervalSec = nonNegative(*wire.Agent.ReportIntervalSec)
		}
		if wire.Agent.HTTPTimeoutSec != nil {
			result.Agent.HTTPTimeoutSec = nonNegative(*wire.Agent.HTTPTimeoutSec)
		}
		if wire.Agent.WSBackoffMinSec != nil {
			result.Agent.WSBackoffMinSec = nonNegative(*wire.Agent.WSBackoffMinSec)
		}
		if wire.Agent.WSBackoffMaxSec != nil {
			result.Agent.WSBackoffMaxSec = nonNegative(*wire.Agent.WSBackoffMaxSec)
		}
	}
	*s = result
	return nil
}

// BinaryInfo 是服务端托管二进制的元信息。
type BinaryInfo struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

// DownloadResult 是二进制下载结果。
type DownloadResult struct {
	Version string
	SHA256  string
	Size    int64
	Header  string
}

// IngestHost 是上报的主机信息。
type IngestHost struct {
	Hostname       string `json:"hostname"`
	NodeID         string `json:"nodeId,omitempty"`
	IP             string `json:"ip"`
	OS             string `json:"os,omitempty"`
	Arch           string `json:"arch,omitempty"`
	OSBuild        int    `json:"osBuild,omitempty"`
	Role           string `json:"role,omitempty"`
	AgentVersion   string `json:"agentVersion,omitempty"`
	AgentStartedAt string `json:"agentStartedAt,omitempty"`
}

// IngestTask 是上报的备份任务。
type IngestTask struct {
	Name         string         `json:"name"`
	Key          string         `json:"key,omitempty"`
	Type         string         `json:"type"`
	Tool         string         `json:"tool,omitempty"`
	Schedule     string         `json:"schedule,omitempty"`
	ScheduleCron string         `json:"scheduleCron,omitempty"`
	Config       map[string]any `json:"config"`
}

// IngestExecution 是上报的执行记录。
type IngestExecution struct {
	TaskName        string `json:"taskName"`
	SessionID       string `json:"sessionId,omitempty"`
	JobID           string `json:"jobId,omitempty"`
	OibID           string `json:"oibId,omitempty"`
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

// IngestReport 是 POST /api/ingest/report 的请求体。
type IngestReport struct {
	Host       IngestHost        `json:"host"`
	Tasks      []IngestTask      `json:"tasks"`
	Executions []IngestExecution `json:"executions"`
}

// ReportResponse 是上报响应。
type ReportResponse struct {
	OK            bool   `json:"ok"`
	HostID        string `json:"hostId"`
	NewExecutions int    `json:"newExecutions"`
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func positive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func cleanList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
