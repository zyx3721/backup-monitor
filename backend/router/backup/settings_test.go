package backup

import (
	"strings"
	"testing"

	"backup-monitor-backend/internal/domain"
)

func TestSettingsAuditText(t *testing.T) {
	tests := []struct {
		section    string
		wantTarget string
		wantDetail string
	}{
		{"thresholds", "判定与保留", "判定与保留配置已更新"},
		{"agent", "Agent 通道", "Agent 通道配置已更新"},
		{"alarm", "安全与告警", "安全与告警配置已更新"},
		{"", "备份监控基础参数", "备份监控基础参数已更新"},
		{"unknown", "备份监控基础参数", "备份监控基础参数已更新"},
	}
	for _, test := range tests {
		target, detail := settingsAuditText(test.section)
		if target != test.wantTarget || detail != test.wantDetail {
			t.Errorf("settingsAuditText(%q) = (%q, %q), want (%q, %q)",
				test.section, target, detail, test.wantTarget, test.wantDetail)
		}
	}
}

func TestSettingsChangeItems(t *testing.T) {
	base := domain.DefaultAppSettings()

	thresholds := base
	thresholds.ReportTimeoutMinutes = 5
	thresholds.AuditRetentionDays = 30
	if got, want := settingsChangeItems(base, thresholds, domain.BaseSectionThresholds), "：主机上报超时、审计日志保留已修改"; got != want {
		t.Errorf("thresholds changes = %q, want %q", got, want)
	}
	if got := settingsChangeItems(base, thresholds, domain.BaseSectionAgent); got != "" {
		t.Errorf("thresholds change reported under agent = %q, want empty", got)
	}

	agent := base
	agent.Agent.ReportIntervalSec = 120
	agent.Agent.WSBackoffMaxSec = 60
	if got, want := settingsChangeItems(base, agent, domain.BaseSectionAgent), "：默认上报间隔、WS 重连退避上限已修改"; got != want {
		t.Errorf("agent changes = %q, want %q", got, want)
	}

	alarm := base
	alarm.DedupeAlerts = !base.DedupeAlerts
	alarm.RequireAgentToken = !base.RequireAgentToken
	if got, want := settingsChangeItems(base, alarm, domain.BaseSectionAlarm), "：告警去重、要求 Agent 令牌已修改"; got != want {
		t.Errorf("alarm changes = %q, want %q", got, want)
	}

	if got := settingsChangeItems(base, base, ""); got != "" {
		t.Errorf("unchanged settings = %q, want empty", got)
	}
}

func TestValidateManualToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{name: "16 位", token: "0123456789abcdef"},
		{name: "48 位 hex", token: "0123456789abcdef0123456789abcdef0123456789abcdef"},
		{name: "128 位", token: strings.Repeat("a", 128)},
		{name: "常见符号", token: "Abc-123_456.~!@#$%^"},
		{name: "15 位过短", token: "0123456789abcde", wantErr: true},
		{name: "129 位过长", token: strings.Repeat("b", 129), wantErr: true},
		{name: "包含空格", token: "0123456789 abcdef", wantErr: true},
		{name: "包含中文", token: "0123456789abcdef令牌", wantErr: true},
		{name: "包含换行", token: "0123456789abcdef\n", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateManualToken(test.token)
			if test.wantErr && err == nil {
				t.Fatalf("validateManualToken(%q) = nil, want error", test.token)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateManualToken(%q) = %v, want nil", test.token, err)
			}
		})
	}
}
