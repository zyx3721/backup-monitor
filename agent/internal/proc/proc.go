// Package proc 按关键字探测本机进程，用于判定备份任务是否仍在运行。
package proc

import (
	"strings"
	"sync"
	"time"
)

const cacheTTL = 3 * time.Second

var (
	cacheMu       sync.Mutex
	cacheCommands []string
	cacheAt       time.Time
)

// Keyword 归一化任务配置里的进程关键字，空串表示不启用进程探测。
func Keyword(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

// Running 判断是否存在命令行命中关键字的进程。
func Running(keyword string) bool {
	keyword = Keyword(keyword)
	if keyword == "" {
		return false
	}
	return matchCommands(snapshot(), keyword)
}

func snapshot() []string {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if time.Since(cacheAt) < cacheTTL {
		return cacheCommands
	}
	cacheCommands = commands()
	cacheAt = time.Now()
	return cacheCommands
}

func matchCommands(commands []string, keyword string) bool {
	needle := strings.ToLower(keyword)
	if needle == "" {
		return false
	}
	for _, command := range commands {
		if strings.Contains(strings.ToLower(command), needle) {
			return true
		}
	}
	return false
}
