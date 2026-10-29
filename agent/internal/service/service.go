// Package service 安装并管理 systemd 或 Windows 服务。
package service

import (
	"errors"
	"os"
	"strings"
)

const (
	Name        = "backup-agent"
	DisplayName = "Backup Monitor Agent"
)

// ErrNotInstalled 表示当前主机未安装对应的系统服务。
var ErrNotInstalled = errors.New("system service is not installed")

var (
	activeName   = Name
	explicitName bool
)

// ResolveName 返回本次操作使用的服务名：命令行优先，其次 BM_SERVICE_NAME，最后默认名。
func ResolveName(flagValue string) string {
	if value := normalizeName(flagValue); value != "" {
		return value
	}
	if value := normalizeName(os.Getenv("BM_SERVICE_NAME")); value != "" {
		return value
	}
	return Name
}

// UseName 将服务名设为显式值，使重启时优先使用而不是自动探测。
func UseName(name string) {
	value := normalizeName(name)
	if value == "" {
		return
	}
	activeName = value
	explicitName = true
}

func normalizeName(value string) string {
	return strings.TrimSuffix(strings.TrimSpace(value), ".service")
}

// DisplayNameFor 返回服务显示名：默认服务沿用固定名称，多实例追加服务名以便区分。
func DisplayNameFor(name string) string {
	if name == Name {
		return DisplayName
	}
	return DisplayName + " (" + name + ")"
}

// ResolveDisplayName 返回服务显示名：命令行优先，其次 BM_DISPLAY_NAME，最后按服务名派生。
func ResolveDisplayName(flagValue, name string) string {
	if value := strings.TrimSpace(flagValue); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("BM_DISPLAY_NAME")); value != "" {
		return value
	}
	return DisplayNameFor(name)
}

func explicitNameValue() (string, bool) {
	return activeName, explicitName
}
