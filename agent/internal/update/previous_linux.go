//go:build linux

package update

// WaitForPreviousProcess 在 Linux 上无需等待：重启通过 exec 原地替换进程。
func WaitForPreviousProcess() {}
