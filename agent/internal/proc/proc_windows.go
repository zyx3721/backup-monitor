//go:build windows

package proc

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const (
	wmicTimeout       = 30 * time.Second
	powerShellTimeout = 60 * time.Second
	powerShellQuery   = "Get-WmiObject Win32_Process | ForEach-Object { $_.CommandLine }"
)

// commands 依次用 wmic 与 PowerShell 枚举本机进程完整命令行。
func commands() ([]string, bool) {
	return commandsWith(runCmd)
}

func commandsWith(run func(time.Duration, string, ...string) ([]byte, error)) ([]string, bool) {
	output, err := run(wmicTimeout, "wmic", "process", "get", "CommandLine")
	if err != nil || strings.TrimSpace(string(output)) == "" {
		output, err = run(powerShellTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", powerShellQuery)
	}
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return nil, false
	}
	return commandLines(output), true
}

func runCmd(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

func commandLines(output []byte) []string {
	text := strings.TrimPrefix(string(output), "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := make([]string, 0, 16)
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
