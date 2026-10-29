//go:build linux

package proc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func commands() []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	list := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(data) == 0 {
			continue
		}
		command := strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
		if command != "" {
			list = append(list, command)
		}
	}
	return list
}
