//go:build linux

package update

import (
	"os"
	"syscall"
)

// spawnRestart 用新镜像原地替换当前进程，避免重启窗口内出现两个 Agent。
func spawnRestart(executable string, args []string) error {
	argv := append([]string{executable}, args...)
	return syscall.Exec(executable, argv, os.Environ())
}
