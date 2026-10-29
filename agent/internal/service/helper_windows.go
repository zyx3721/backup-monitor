//go:build windows

package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// RestartState 描述一次服务重启所需的上下文
type RestartState struct {
	Service    string `json:"service"`
	Executable string `json:"executable"`
	Backup     string `json:"backup"`
	PID        int    `json:"pid"`
}

// RunRestartHelper 以旧版本二进制身份重启服务，校验启动结果，失败时还原备份
func RunRestartHelper(statePath string) error {
	payload, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("read restart state: %w", err)
	}
	var state RestartState
	if err := json.Unmarshal(payload, &state); err != nil {
		return fmt.Errorf("decode restart state: %w", err)
	}
	defer os.Remove(statePath)
	logger := newHelperLogger(filepath.Join(filepath.Dir(state.Executable), "update.log"))
	logger.Printf("开始重启服务 %s（旧进程 %d）", state.Service, state.PID)

	waitProcessExit(state.PID, 60*time.Second)
	if err := startService(state.Service); err != nil {
		logger.Printf("新版本启动失败：%v，开始回滚", err)
		if rollbackErr := rollbackService(state); rollbackErr != nil {
			logger.Printf("回滚失败：%v", rollbackErr)
			return rollbackErr
		}
		logger.Printf("已回滚到旧版本")
		return nil
	}
	logger.Printf("服务已启动并处于运行状态")
	return nil
}

func startService(name string) error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager: %w", err)
	}
	defer manager.Disconnect()
	handle, err := manager.OpenService(name)
	if err != nil {
		return fmt.Errorf("open service %s: %w", name, err)
	}
	defer handle.Close()
	query := func() (uint32, error) {
		status, err := handle.Query()
		if err != nil {
			return 0, err
		}
		return uint32(status.State), nil
	}
	if err := waitForStatus(query, uint32(svc.Stopped), 60*time.Second); err != nil {
		return err
	}
	if err := handle.Start(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	if err := waitForStatus(query, uint32(svc.Running), 30*time.Second); err != nil {
		return err
	}
	return nil
}

func rollbackService(state RestartState) error {
	stopService(state.Service, 30*time.Second)
	if err := restoreBackup(state.Executable, state.Backup); err != nil {
		return err
	}
	return startService(state.Service)
}

func stopService(name string, timeout time.Duration) {
	manager, err := mgr.Connect()
	if err != nil {
		return
	}
	defer manager.Disconnect()
	handle, err := manager.OpenService(name)
	if err != nil {
		return
	}
	defer handle.Close()
	_, _ = handle.Control(svc.Stop)
	_ = waitForStatus(func() (uint32, error) {
		status, err := handle.Query()
		if err != nil {
			return 0, err
		}
		return uint32(status.State), nil
	}, uint32(svc.Stopped), timeout)
}

func restoreBackup(executable, backup string) error {
	if err := os.Rename(backup, executable); err == nil {
		return nil
	}
	_ = os.Remove(executable)
	if err := os.Rename(backup, executable); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}
	return nil
}

func waitProcessExit(pid int, timeout time.Duration) {
	if pid <= 0 {
		return
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		time.Sleep(time.Second)
		return
	}
	defer windows.CloseHandle(handle)
	_, _ = windows.WaitForSingleObject(handle, uint32(timeout/time.Millisecond))
}

type helperLogger struct {
	mutex sync.Mutex
	file  *os.File
}

func newHelperLogger(path string) *helperLogger {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return &helperLogger{}
	}
	return &helperLogger{file: file}
}

func (l *helperLogger) Printf(format string, args ...any) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.file == nil {
		return
	}
	_, _ = l.file.WriteString(time.Now().Format("2006/01/02 15:04:05") + " " + fmt.Sprintf(format, args...) + "\n")
}
