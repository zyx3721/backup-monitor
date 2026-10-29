// Package veeam 通过 veeamconfig 命令行采集 Veeam 作业与会话。
package veeam

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	ansiPattern  = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	spacePattern = regexp.MustCompile(`\s{2,}`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F-]{16,}$`)
)

// Job 是 Veeam 备份作业。
type Job struct {
	ID       string
	Name     string
	Type     string
	Status   string
	Schedule string
}

// Session 是 Veeam 作业的一次会话。
type Session struct {
	ID              string
	JobID           string
	JobName         string
	OIBID           string
	State           string
	Creation        time.Time
	Start           time.Time
	End             time.Time
	HasStart        bool
	HasEnd          bool
	ProcessedSize   string
	TransferredSize string
	Attempt         int
	HasAttempt      bool
	MaxAttempts     int
	HasMaxAttempts  bool
	IsLastAttempt   bool
	HasLastAttempt  bool
	Result          string
}

// Client 封装 veeamconfig 调用。
type Client struct {
	Path    string
	Timeout time.Duration
}

// Find 返回可执行的 veeamconfig 路径。
func Find() string {
	candidates := []string{}
	if runtime.GOOS == "windows" {
		if programFiles := strings.TrimSpace(os.Getenv("ProgramFiles")); programFiles != "" {
			candidates = append(candidates,
				filepath.Join(programFiles, "Veeam", "Endpoint Backup", "veeamconfig.exe"),
				filepath.Join(programFiles, "Veeam", "Backup and Replication", "Console", "veeamconfig.exe"),
			)
		}
	} else {
		candidates = append(candidates,
			"/usr/local/sbin/veeamconfig",
			"/usr/sbin/veeamconfig",
			"/opt/veeam/veeamconfig",
			"/opt/veeam/bin/veeamconfig",
		)
	}
	for _, candidate := range candidates {
		if isExecutable(candidate) {
			return candidate
		}
	}
	for _, name := range []string{"veeamconfig", "veeamconfig.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

// New 创建 Veeam 客户端，未安装时返回 Path 为空的客户端。
func New() *Client {
	return &Client{Path: Find(), Timeout: 30 * time.Second}
}

// Available 判断当前客户端是否可执行。
func (c *Client) Available() bool {
	return c != nil && strings.TrimSpace(c.Path) != ""
}

// ListJobs 解析 veeamconfig job list。
func (c *Client) ListJobs(ctx context.Context) ([]Job, error) {
	output, err := c.run(ctx, "job", "list")
	if err != nil {
		return nil, err
	}
	return parseJobs(output), nil
}

// ListSessions 解析 veeamconfig session list，可按作业 ID 过滤。
func (c *Client) ListSessions(ctx context.Context, jobID string) ([]Session, error) {
	output, err := c.run(ctx, "session", "list")
	if err != nil {
		return nil, err
	}
	sessions := parseSessions(output)
	if strings.TrimSpace(jobID) == "" {
		return sessions, nil
	}
	filtered := make([]Session, 0, len(sessions))
	for _, session := range sessions {
		if strings.EqualFold(session.JobID, jobID) {
			filtered = append(filtered, session)
		}
	}
	return filtered, nil
}

// SessionInfo 解析 veeamconfig session info 的键值输出。
func (c *Client) SessionInfo(ctx context.Context, sessionID string) (Session, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, fmt.Errorf("session ID is required")
	}
	output, err := c.run(ctx, "session", "info", "--id", sessionID)
	if err != nil {
		return Session{}, err
	}
	session := parseSessionInfo(output)
	if session.ID == "" {
		session.ID = sessionID
	}
	return session, nil
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	if !c.Available() {
		return "", fmt.Errorf("veeamconfig is not available")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(runCtx, c.Path, args...)
	command.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	var stdout strings.Builder
	var stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("veeamconfig %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}
