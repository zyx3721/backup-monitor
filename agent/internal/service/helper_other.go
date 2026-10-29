//go:build !windows

package service

import "errors"

// RunRestartHelper 仅 Windows 提供重启守护子进程
func RunRestartHelper(string) error {
	return errors.New("restart helper is only supported on Windows")
}
