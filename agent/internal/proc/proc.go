// Package proc 按关键字探测本机进程，用于判定备份任务是否仍在运行。
package proc

import (
	"strings"
	"sync"
	"time"
)

const cacheTTL = 3 * time.Second

var (
	cacheMu        sync.Mutex
	cacheCommands  []string
	cacheAvailable bool
	cacheAt        time.Time
)

// Keyword 按顺序返回首个非空的进程关键字，空串表示所有候选均为空。
func Keyword(values ...string) string {
	for _, value := range values {
		if keyword := strings.Trim(strings.TrimSpace(value), `"'`); keyword != "" {
			return keyword
		}
	}
	return ""
}

// Running 判断是否存在命令行命中关键字的进程，并返回进程列表是否可用。
func Running(keyword string) (bool, bool) {
	keyword = Keyword(keyword)
	if keyword == "" {
		return false, false
	}
	commands, available := snapshot()
	return matchSnapshot(commands, available, keyword)
}

func matchSnapshot(commands []string, available bool, keyword string) (bool, bool) {
	keyword = Keyword(keyword)
	if keyword == "" || !available {
		return false, false
	}
	return matchCommands(commands, keyword), true
}

func snapshot() ([]string, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if time.Since(cacheAt) < cacheTTL {
		return cacheCommands, cacheAvailable
	}
	cacheCommands, cacheAvailable = commands()
	cacheAt = time.Now()
	return cacheCommands, cacheAvailable
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
