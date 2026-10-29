//go:build linux

package collect

import (
	"context"
	"time"

	"backup-monitor-agent/internal/api"
)

func (c *Collector) platformEntries(_ context.Context, _ api.ProfileConfig, _ time.Time, _ bool) []taskEntry {
	return nil
}
