//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func spawnRestart(executable string, args []string) error {
	command := exec.Command(executable, args...)
	command.Env = append(os.Environ(), fmt.Sprintf("BM_REPLACED_PID=%d", os.Getpid()))
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return command.Start()
}
