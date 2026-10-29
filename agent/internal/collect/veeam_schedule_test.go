package collect

import (
	"testing"
	"time"

	"backup-monitor-agent/internal/veeam"
)

func TestDeriveVeeamScheduleDaily(t *testing.T) {
	base := time.Date(2026, 10, 24, 6, 0, 12, 0, time.Local)
	sessions := []veeam.Session{
		{Start: base, HasStart: true},
		{Start: base.Add(24 * time.Hour), HasStart: true},
		{Start: base.Add(48 * time.Hour).Add(2 * time.Second), HasStart: true},
	}
	expr, human := deriveVeeamSchedule(sessions)
	if expr != "0 6 * * *" {
		t.Errorf("cron = %q", expr)
	}
	if human != "每天 06:00" {
		t.Errorf("human = %q", human)
	}
}

func TestDeriveVeeamScheduleRejectsIrregular(t *testing.T) {
	base := time.Date(2026, 10, 24, 6, 0, 0, 0, time.Local)
	sessions := []veeam.Session{
		{Start: base, HasStart: true},
		{Start: base.Add(25 * time.Hour), HasStart: true},
	}
	if expr, _ := deriveVeeamSchedule(sessions); expr != "" {
		t.Errorf("cron = %q, want empty", expr)
	}
}

func TestDeriveVeeamScheduleNeedsTwoDays(t *testing.T) {
	base := time.Date(2026, 10, 24, 6, 0, 0, 0, time.Local)
	sessions := []veeam.Session{
		{Start: base, HasStart: true},
		{Start: base.Add(2 * time.Minute), HasStart: true},
	}
	if expr, _ := deriveVeeamSchedule(sessions); expr != "" {
		t.Errorf("cron = %q, want empty", expr)
	}
}

func TestDeriveVeeamScheduleRejectsWeeklySpacing(t *testing.T) {
	base := time.Date(2026, 10, 5, 6, 0, 0, 0, time.Local)
	sessions := []veeam.Session{
		{Start: base, HasStart: true},
		{Start: base.Add(7 * 24 * time.Hour), HasStart: true},
	}
	if expr, _ := deriveVeeamSchedule(sessions); expr != "" {
		t.Errorf("cron = %q, want empty", expr)
	}
}
