package domain

import "testing"

func TestDefaultAppSettingsValues(t *testing.T) {
	got := DefaultAppSettings()
	want := AppSettings{
		ReportTimeoutMinutes: 3, OfflineTimeoutMinutes: 6, NotRunGraceMinutes: 30,
		ScheduleMatchWindowMinutes: 60, SchedulerIntervalSec: 30,
		DedupeAlerts: true, RetentionDays: 90, AuditRetentionDays: 90, NotifyRetentionDays: 90, RequireAgentToken: false,
		Agent: AgentGlobalConfig{ReportIntervalSec: 60, HTTPTimeoutSec: 0, WSBackoffMinSec: 0, WSBackoffMaxSec: 0},
	}
	if got != want {
		t.Fatalf("defaults = %+v, want %+v", got, want)
	}
}

func TestNormalizeAppSettingsFallsBackToDefaults(t *testing.T) {
	if got := NormalizeAppSettings(map[string]any{}); got != DefaultAppSettings() {
		t.Fatalf("empty input = %+v, want defaults", got)
	}
}

func TestNormalizeAppSettingsClampsToReferenceRanges(t *testing.T) {
	got := NormalizeAppSettings(map[string]any{
		"reportTimeoutMinutes":       99999,
		"offlineTimeoutMinutes":      99999,
		"notRunGraceMinutes":         99999,
		"scheduleMatchWindowMinutes": 99999,
		"schedulerIntervalSec":       999999,
		"retentionDays":              99999,
		"auditRetentionDays":         99999,
		"notifyRetentionDays":        99999,
		"agent": map[string]any{
			"reportIntervalSec": 999999,
			"httpTimeoutSec":    99999,
			"wsBackoffMinSec":   99999,
			"wsBackoffMaxSec":   99999,
		},
	})
	if got.ReportTimeoutMinutes != maxReportTimeoutMinutes ||
		got.OfflineTimeoutMinutes != maxOfflineTimeoutMinutes ||
		got.NotRunGraceMinutes != maxNotRunGraceMinutes ||
		got.ScheduleMatchWindowMinutes != maxScheduleMatchWindowMinutes ||
		got.SchedulerIntervalSec != maxSchedulerIntervalSec ||
		got.RetentionDays != maxRetentionDays ||
		got.AuditRetentionDays != maxAuditRetentionDays ||
		got.NotifyRetentionDays != maxRetentionDays {
		t.Fatalf("settings not clamped: %+v", got)
	}
	if got.Agent.ReportIntervalSec != maxAgentReportIntervalSec ||
		got.Agent.HTTPTimeoutSec != maxAgentHTTPTimeoutSec ||
		got.Agent.WSBackoffMinSec != maxAgentWSBackoffMinSec ||
		got.Agent.WSBackoffMaxSec != maxAgentWSBackoffMaxSec {
		t.Fatalf("agent settings not clamped: %+v", got.Agent)
	}
}

func TestNormalizeAppSettingsKeepsAgentMinimum(t *testing.T) {
	got := NormalizeAppSettings(map[string]any{
		"agent": map[string]any{"reportIntervalSec": 10},
	})
	if got.Agent.ReportIntervalSec != 10 {
		t.Fatalf("report interval = %d, want 10", got.Agent.ReportIntervalSec)
	}
}

func TestNormalizeAppSettingsKeepsOfflineThresholdAboveReport(t *testing.T) {
	got := NormalizeAppSettings(map[string]any{
		"reportTimeoutMinutes":  20,
		"offlineTimeoutMinutes": 1,
	})
	if got.OfflineTimeoutMinutes != 60 {
		t.Fatalf("offline = %d, want 60", got.OfflineTimeoutMinutes)
	}

	got = NormalizeAppSettings(map[string]any{
		"reportTimeoutMinutes":  1440,
		"offlineTimeoutMinutes": 1,
	})
	if got.OfflineTimeoutMinutes != 4320 {
		t.Fatalf("offline = %d, want 4320", got.OfflineTimeoutMinutes)
	}
}

func TestNormalizeAppSettingsAgentBackoffOrder(t *testing.T) {
	got := NormalizeAppSettings(map[string]any{
		"agent": map[string]any{
			"reportIntervalSec": 1,
			"wsBackoffMinSec":   120,
			"wsBackoffMaxSec":   1,
		},
	})
	if got.Agent.ReportIntervalSec != 60 {
		t.Fatalf("report interval = %d, want fallback 60", got.Agent.ReportIntervalSec)
	}
	if got.Agent.WSBackoffMaxSec != 120 {
		t.Fatalf("backoff max = %d, want 120", got.Agent.WSBackoffMaxSec)
	}
}

func TestValidateAppSettingsRejectsOfflineBelowReport(t *testing.T) {
	got := ValidateAppSettings(map[string]any{
		"reportTimeoutMinutes":  3,
		"offlineTimeoutMinutes": 2,
	})
	if got != "主机离线阈值不能小于主机上报超时" {
		t.Fatalf("validation = %q", got)
	}
}

func TestValidateAppSettingsAcceptsDefaults(t *testing.T) {
	raw := map[string]any{
		"reportTimeoutMinutes":  3,
		"offlineTimeoutMinutes": 6,
	}
	if got := ValidateAppSettings(raw); got != "" {
		t.Fatalf("validation = %q, want empty", got)
	}
}

func TestAppSettingsSectionsRoundTrip(t *testing.T) {
	custom := AppSettings{
		ReportTimeoutMinutes: 10, OfflineTimeoutMinutes: 30, NotRunGraceMinutes: 45,
		ScheduleMatchWindowMinutes: 90, SchedulerIntervalSec: 60,
		DedupeAlerts: false, RetentionDays: 120, AuditRetentionDays: 180, NotifyRetentionDays: 60, RequireAgentToken: true,
		Agent: AgentGlobalConfig{ReportIntervalSec: 120, HTTPTimeoutSec: 15, WSBackoffMinSec: 5, WSBackoffMaxSec: 45},
	}
	for _, want := range []AppSettings{DefaultAppSettings(), custom} {
		got := MergeAppSettingsSections(want.Sections())
		if got != want {
			t.Fatalf("round trip = %+v, want %+v", got, want)
		}
	}
}

func TestAppSettingsSectionsCoverExpectedRows(t *testing.T) {
	sections := DefaultAppSettings().Sections()
	for _, name := range []string{BaseSectionThresholds, BaseSectionAgent, BaseSectionAlarm} {
		if _, ok := sections[name]; !ok {
			t.Fatalf("missing section %s", name)
		}
	}
	if len(sections) != 3 {
		t.Fatalf("sections = %d, want 3", len(sections))
	}
}

func TestValidateAppSettingsRejectsWSBackoffInverted(t *testing.T) {
	raw := map[string]any{
		"agent": map[string]any{
			"wsBackoffMinSec": float64(60),
			"wsBackoffMaxSec": float64(10),
		},
	}
	if got := ValidateAppSettings(raw); got == "" {
		t.Fatal("expected validation error for inverted backoff")
	}
	ok := map[string]any{
		"agent": map[string]any{
			"wsBackoffMinSec": float64(2),
			"wsBackoffMaxSec": float64(30),
		},
	}
	if got := ValidateAppSettings(ok); got != "" {
		t.Fatalf("unexpected error: %s", got)
	}
}
