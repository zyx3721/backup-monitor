//go:build linux

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var systemdDirs = []string{"/etc/systemd/system", "/run/systemd/system", "/usr/lib/systemd/system", "/lib/systemd/system"}

// Install 写入并启动指定名称的 systemd 单元。
func Install(executable, name, displayName string) (string, error) {
	if os.Geteuid() != 0 {
		return "", errors.New("root privileges are required to install the systemd service")
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = DisplayName
	}
	absolute, err := filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	if path, ok := findUnitFile(name); ok {
		return "systemd service " + name + " is already installed (" + path + ")", nil
	}
	unit := fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%q -service -name %q
# 日志由 Agent 自行写入二进制同目录 agent.log，服务模式下不写 journald
StandardOutput=null
StandardError=null
SyslogIdentifier=%s
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`, displayName, absolute, name, name)

	target := filepath.Join(systemdDirs[0], name+".service")
	if err := os.WriteFile(target, []byte(unit), 0o644); err != nil {
		return "", fmt.Errorf("write systemd unit: %w", err)
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		return "", err
	}
	if err := runSystemctl("enable", name); err != nil {
		return "", err
	}
	if err := runSystemctl("restart", name); err != nil {
		return "", err
	}
	return "installed systemd service " + name, nil
}

// Uninstall 停止并删除指定名称的 systemd 单元，查找范围包含系统级目录。
func Uninstall(name string) (string, error) {
	if os.Geteuid() != 0 {
		return "", errors.New("root privileges are required to uninstall the systemd service")
	}
	path, ok := findUnitFile(name)
	if !ok {
		return "", errors.New("systemd service is not installed")
	}
	var failures []error
	if err := runSystemctl("disable", "--now", name); err != nil {
		failures = append(failures, err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		failures = append(failures, fmt.Errorf("remove systemd unit: %w", err))
	}
	if err := runSystemctl("daemon-reload"); err != nil {
		failures = append(failures, err)
	}
	if len(failures) > 0 {
		return "", errors.Join(failures...)
	}
	return "uninstalled systemd service " + name, nil
}

// findUnitFile 在系统与用户级 systemd 目录中查找指定名称的单元文件。
func findUnitFile(name string) (string, bool) {
	fileName := name + ".service"
	for _, dir := range systemdDirs {
		path := filepath.Join(dir, fileName)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

// Restart 通过 systemd 重启服务；重启命令放进独立 transient 单元，避免随本单元一起被停掉
func Restart() error {
	name, ok := findUnit()
	if !ok {
		return ErrNotInstalled
	}
	transientErr := runTransientRestart(name)
	if transientErr == nil {
		return nil
	}
	err := runSystemctl("restart", "--no-block", name)
	if err == nil {
		return nil
	}
	if killedBySignal(err) {
		return nil
	}
	return fmt.Errorf("restart %s failed: %w (systemd-run: %v)", name, err, transientErr)
}

// runTransientRestart 用 systemd-run 在独立单元中执行重启，避免 systemd 停本单元时杀掉重启命令
func runTransientRestart(name string) error {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		return err
	}
	unit := fmt.Sprintf("bm-restart-%d", time.Now().UnixNano())
	command := exec.Command("systemd-run", "--quiet", "--collect", "--no-block",
		"--unit="+unit, "--", "systemctl", "restart", name)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemd-run restart failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// killedBySignal 判断命令是否被信号终止；重启时 systemd 会停掉整个 cgroup，属预期情况
func killedBySignal(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return false
	}
	return status.Signaled() && (status.Signal() == syscall.SIGTERM || status.Signal() == syscall.SIGKILL)
}

// findUnit 返回当前进程所属的 systemd 单元名，第二个返回值表示是否真的匹配到单元。
func findUnit() (string, bool) {
	if name, ok := explicitNameValue(); ok {
		_, exists := findUnitFile(name)
		return name, exists
	}
	if value := normalizeName(os.Getenv("BM_SERVICE_NAME")); value != "" {
		return value, true
	}
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		if name := unitFromCgroup(string(data)); name != "" {
			return name, true
		}
	}
	if executable, err := os.Executable(); err == nil {
		if name := unitForExecutable(executable); name != "" {
			return name, true
		}
	}
	return Name, false
}

func unitForExecutable(executable string) string {
	target, err := filepath.EvalSymlinks(executable)
	if err != nil {
		target = executable
	}
	for _, dir := range systemdDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".service") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			if unitExecStartMatches(string(data), target, executable) {
				return entry.Name()
			}
		}
	}
	return ""
}

func unitExecStartMatches(content, target, executable string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}
		path := firstField(strings.TrimPrefix(line, "ExecStart="))
		if path == "" {
			continue
		}
		if path == executable {
			return true
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil && resolved == target {
			return true
		}
	}
	return false
}

func firstField(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "-"))
	if strings.HasPrefix(value, `"`) {
		if end := strings.Index(value[1:], `"`); end >= 0 {
			return value[1 : 1+end]
		}
	}
	if index := strings.IndexAny(value, " \t"); index >= 0 {
		return value[:index]
	}
	return value
}

func unitFromCgroup(content string) string {
	for _, line := range strings.Split(content, "\n") {
		index := strings.LastIndex(line, "/")
		if index < 0 {
			continue
		}
		name := strings.TrimSpace(line[index+1:])
		if strings.HasSuffix(name, ".service") {
			return name
		}
	}
	return ""
}

// Run 在 systemd 模式下直接执行主循环。
func Run(ctx context.Context, run func(context.Context) error) error {
	return run(ctx)
}

// IsService 判断当前进程是否由 systemd 托管。
func IsService() bool {
	if strings.TrimSpace(os.Getenv("INVOCATION_ID")) != "" {
		return true
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}
	return os.Getppid() == 1
}

func runSystemctl(args ...string) error {
	command := exec.Command("systemctl", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
