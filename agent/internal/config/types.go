// Package config 定义 Agent 本地配置结构、角色取值与共享默认值。
package config

import "strings"

const (
	RoleDatabase       = "database"
	RoleWeb            = "web"
	RoleApplication    = "application"
	RoleFile           = "file"
	RoleVirtualization = "virtualization"
)

const (
	DefaultIntervalSec          = 600
	MinIntervalSec              = 10
	DefaultHTTPTimeoutSec       = 20
	DefaultHeartbeatIntervalSec = 30
	DefaultWSBackoffMinSec      = 2
	DefaultWSBackoffMaxSec      = 30
	DefaultManualTriggerMinutes = 30
	DefaultArtifactMaxAgeHours  = 30
	DefaultArtifactMaxDepth     = 3
	DefaultMaxSessionsPerJob    = 5
	DefaultMaxSessionInfoCalls  = 10
	DefaultMaxJobInfoCalls      = 20
	DefaultPreToleranceMinutes  = 10
	DefaultLogMaxMb             = 10
	DefaultLogBackups           = 2
)

// Version 是当前 Agent 版本号，可由构建参数覆盖。
var Version = "1.0.0"

// Local 是 Agent 本地启动配置，优先级为命令行参数、环境变量、JSON 文件、默认值。
type Local struct {
	ServerURL   string `json:"serverUrl"`
	Token       string `json:"token"`
	Hostname    string `json:"hostname"`
	IntervalSec int    `json:"intervalSec"`
	Role        string `json:"role"`
	LogFile     string `json:"logFile"`
	LogMaxMb    int    `json:"logMaxMb"`
	LogBackups  int    `json:"logBackups"`
}

// DefaultLocal 返回本地配置的默认值。
func DefaultLocal() Local {
	return Local{
		IntervalSec: DefaultIntervalSec,
		Role:        RoleDatabase,
		LogMaxMb:    DefaultLogMaxMb,
		LogBackups:  DefaultLogBackups,
	}
}

// ValidRole 判断角色是否属于合法取值。
func ValidRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleDatabase, RoleWeb, RoleApplication, RoleFile, RoleVirtualization:
		return true
	default:
		return false
	}
}

// NormalizeRole 返回规范化后的角色，非法值返回空串。
func NormalizeRole(role string) string {
	value := strings.ToLower(strings.TrimSpace(role))
	if ValidRole(value) {
		return value
	}
	return ""
}

// Roles 返回全部合法角色。
func Roles() []string {
	return []string{RoleDatabase, RoleWeb, RoleApplication, RoleFile, RoleVirtualization}
}
