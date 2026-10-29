//go:build windows

package collect

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/discover"
)

func (c *Collector) platformEntries(_ context.Context, profile api.ProfileConfig, now time.Time, veeamFound bool) []taskEntry {
	if !profile.CollectVeeam || veeamFound {
		return nil
	}
	path := findVeeamStatusFile()
	if path == "" {
		return nil
	}
	task := discover.Task{
		Name: "Veeam Backup",
		Key:  "veeam:status",
		Type: "veeam",
		Tool: "Veeam",
		Config: map[string]any{
			"source":     "veeam-status",
			"statusFile": path,
			"command":    "veeam-status",
		},
	}
	execution := c.statusExecution(task, profile, now, time.Time{}, false, false)
	if execution == nil {
		return nil
	}
	return []taskEntry{{
		task:       toIngestTask(task),
		executions: []api.IngestExecution{*execution},
	}}
}

func findVeeamStatusFile() string {
	candidates := make([]string, 0, 4)
	if programData := os.Getenv("ProgramData"); programData != "" {
		candidates = append(candidates,
			filepath.Join(programData, "Veeam", "Backup", "veeam-status.txt"),
			filepath.Join(programData, "Veeam", "Endpoint Backup", "veeam-status.txt"),
		)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "veeam-status.txt"))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(workingDirectory, "veeam-status.txt"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate
		}
	}
	return ""
}
