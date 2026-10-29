//go:build windows

package update

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

// WaitForPreviousProcess 等待被替换的旧进程退出，避免非服务模式下两个 Agent 并行采集。
func WaitForPreviousProcess() {
	raw := strings.TrimSpace(os.Getenv("BM_REPLACED_PID"))
	if raw == "" {
		return
	}
	pid, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || pid == 0 {
		return
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_, _ = windows.WaitForSingleObject(handle, 3000)
}
