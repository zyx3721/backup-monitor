//go:build windows

package sysinfo

import (
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// osBuild 读取 Windows 构建号，用于服务端判断安装包与系统版本是否匹配
func osBuild() int {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return 0
	}
	defer key.Close()
	for _, name := range []string{"CurrentBuildNumber", "CurrentBuild"} {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		build, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && build > 0 {
			return build
		}
	}
	return 0
}

func osName() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return ""
	}
	defer key.Close()
	product, _, err := key.GetStringValue("ProductName")
	if err != nil {
		return ""
	}
	parts := []string{strings.TrimSpace(product)}
	if servicePack, _, err := key.GetStringValue("CSDVersion"); err == nil {
		if value := strings.TrimSpace(servicePack); value != "" {
			parts = append(parts, value)
		}
	}
	if build, _, err := key.GetStringValue("CurrentBuild"); err == nil {
		if value := strings.TrimSpace(build); value != "" {
			parts = append(parts, "Build "+value)
		}
	}
	return strings.Join(parts, " ")
}
