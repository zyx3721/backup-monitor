//go:build windows

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Install 创建并启动指定名称的 Windows 服务。
func Install(executable, name, displayName string) (string, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return "", privilegedError("install", err)
	}
	defer manager.Disconnect()

	if strings.TrimSpace(displayName) == "" {
		displayName = DisplayName
	}
	existing, openErr := manager.OpenService(name)
	if openErr == nil {
		existing.Close()
		return "Windows service " + name + " is already installed", nil
	}
	if !errors.Is(openErr, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return "", privilegedError("install", openErr)
	}

	service, err := manager.CreateService(name, executable, mgr.Config{
		DisplayName: displayName,
		StartType:   mgr.StartAutomatic,
		Description: "Backup Monitor Agent",
	}, "-service", "-name", name)
	if err != nil {
		return "", privilegedError("install", err)
	}
	defer service.Close()

	if err := service.Start(); err != nil {
		return "", privilegedError("start", err)
	}
	return "installed Windows service " + name, nil
}

// Uninstall 停止并删除指定名称的 Windows 服务。
func Uninstall(name string) (string, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return "", privilegedError("uninstall", err)
	}
	defer manager.Disconnect()

	service, err := manager.OpenService(name)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return "", errors.New("Windows service is not installed")
		}
		return "", privilegedError("uninstall", err)
	}
	defer service.Close()

	if _, err := service.Control(svc.Stop); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return "", privilegedError("stop", err)
	}
	if err := waitForState(service, svc.Stopped, 20*time.Second); err != nil {
		return "", err
	}
	if err := service.Delete(); err != nil {
		return "", privilegedError("delete", err)
	}
	return "uninstalled Windows service " + name, nil
}

// Restart 拉起重启守护子进程；子进程以旧版本二进制身份等待退出、重启、校验并在失败时回滚
func Restart() error {
	name := resolveServiceName()
	if !serviceExists(name) {
		return ErrNotInstalled
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	backup := executable + ".bak"
	if _, err := os.Stat(backup); err != nil {
		return scheduleRestartScript(name)
	}
	statePath := executable + ".update-restart.json"
	payload, err := json.Marshal(RestartState{Service: name, Executable: executable, Backup: backup, PID: os.Getpid()})
	if err != nil {
		return fmt.Errorf("encode restart state: %w", err)
	}
	if err := os.WriteFile(statePath, payload, 0o600); err != nil {
		return scheduleRestartScript(name)
	}
	command := exec.Command(backup, "-restart-helper", statePath)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	if err := command.Start(); err != nil {
		os.Remove(statePath)
		return scheduleRestartScript(name)
	}
	return nil
}

// scheduleRestartScript 是守护子进程不可用时的兜底：延时脚本先停服务，轮询到停止后再启动
func scheduleRestartScript(name string) error {
	scriptPath, err := writeRestartScript(name)
	if err != nil {
		return err
	}
	command := exec.Command("cmd.exe", "/c", scriptPath)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	if err := command.Start(); err != nil {
		os.Remove(scriptPath)
		return fmt.Errorf("schedule service restart: %w", err)
	}
	return nil
}

// restartScriptTemplate 轮询服务状态，停止后立即启动，避免固定延时造成空等或抢跑。
const restartScriptTemplate = `@echo off
sc.exe stop "{{name}}" >nul 2>&1
for /l %%i in (1,1,30) do (
  sc.exe query "{{name}}" | findstr /C:"STOPPED" >nul && goto run
  ping -n 2 127.0.0.1 >nul
)
:run
sc.exe start "{{name}}" >nul 2>&1
del "%~f0" >nul 2>&1
`

// restartScript 生成重启批处理内容。
func restartScript(name string) string {
	return strings.ReplaceAll(restartScriptTemplate, "{{name}}", name)
}

func writeRestartScript(name string) (string, error) {
	file, err := os.CreateTemp("", "bm-restart-*.cmd")
	if err != nil {
		return "", fmt.Errorf("create restart script: %w", err)
	}
	path := file.Name()
	if _, err := file.WriteString(restartScript(name)); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("write restart script: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close restart script: %w", err)
	}
	return path, nil
}

// Run 以 Windows 服务方式运行主循环。
func Run(_ context.Context, run func(context.Context) error) error {
	handler := &serviceHandler{run: run}
	err := svc.Run(runServiceName(), handler)
	if err != nil {
		return fmt.Errorf("run Windows service: %w", err)
	}
	return nil
}

// runServiceName 返回 SCM 分发时使用的服务名。
func runServiceName() string {
	if name, ok := explicitNameValue(); ok {
		return name
	}
	return Name
}

// IsService 判断当前进程是否由 Windows 服务管理器启动。
func IsService() bool {
	running, err := svc.IsWindowsService()
	return err == nil && running
}

type serviceHandler struct {
	run func(context.Context) error
}

func (h *serviceHandler) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)

	go func() {
		done <- h.run(ctx)
	}()

	current := svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	changes <- current

	for {
		select {
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- current
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case err := <-done:
					return false, exitCode(err)
				case <-time.After(30 * time.Second):
					return false, 1
				}
			}
		case err := <-done:
			return false, exitCode(err)
		}
	}
}

func exitCode(err error) uint32 {
	if err != nil {
		return 1
	}
	return 0
}

func waitForState(service *mgr.Service, desired svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := service.Query()
		if err != nil {
			return fmt.Errorf("query Windows service state: %w", err)
		}
		if status.State == desired {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Windows service did not reach state %d within %s", desired, timeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func privilegedError(action string, err error) error {
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("administrator privileges are required to %s the Windows service: %w", action, err)
	}
	return fmt.Errorf("%s Windows service: %w", action, err)
}

// resolveServiceName 返回当前程序实际注册的服务名：显式设置优先，其次环境变量，再按 ImagePath 匹配。
func resolveServiceName() string {
	if name, ok := explicitNameValue(); ok {
		return name
	}
	if custom := normalizeName(os.Getenv("BM_SERVICE_NAME")); custom != "" {
		return custom
	}
	executable, err := os.Executable()
	if err != nil {
		return Name
	}
	manager, err := mgr.Connect()
	if err != nil {
		return Name
	}
	defer manager.Disconnect()
	services, err := manager.ListServices()
	if err != nil {
		return Name
	}
	for _, candidate := range services {
		service, err := manager.OpenService(candidate)
		if err != nil {
			continue
		}
		config, err := service.Config()
		service.Close()
		if err != nil {
			continue
		}
		if matchesExecutable(serviceExecutable(config.BinaryPathName), executable) {
			return candidate
		}
	}
	return Name
}

// serviceExecutable 从服务 ImagePath 中取出可执行文件路径，兼容引号、参数与无引号空格路径。
func serviceExecutable(binaryPath string) string {
	trimmed := strings.TrimSpace(binaryPath)
	if trimmed == "" {
		return ""
	}
	if trimmed[0] == '"' {
		rest := trimmed[1:]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end]
		}
		return strings.Trim(rest, `"`)
	}
	if index := strings.Index(strings.ToLower(trimmed), ".exe"); index >= 0 {
		return trimmed[:index+4]
	}
	if space := strings.Index(trimmed, " "); space > 0 {
		return trimmed[:space]
	}
	return trimmed
}

// matchesExecutable 按 Windows 路径规则比较两个可执行文件路径是否相同。
func matchesExecutable(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

// serviceExists 判断指定名称的 Windows 服务是否已注册。
func serviceExists(name string) bool {
	manager, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer manager.Disconnect()
	service, err := manager.OpenService(name)
	if err != nil {
		return false
	}
	service.Close()
	return true
}
