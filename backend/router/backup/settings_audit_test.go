package backup

import (
	"testing"
	"time"

	"backup-monitor-backend/internal/domain"
)

func TestBinarySizeText(t *testing.T) {
	tests := map[int64]string{
		0:          "0 B",
		512:        "512 B",
		1024:       "1.0 KB",
		1536:       "1.5 KB",
		5_872_026:  "5.6 MB",
		12_582_912: "12.0 MB",
	}
	for size, want := range tests {
		if got := binarySizeText(size); got != want {
			t.Errorf("binarySizeText(%d) = %q, want %q", size, got, want)
		}
	}
}

func TestAlertAuditDetail(t *testing.T) {
	alert := &domain.Alert{Message: "备份任务失败", HostName: "db-01", Type: "backup_failed"}
	if got, want := alertAuditDetail(alert, "已手动恢复"), "告警内容「备份任务失败」（主机：db-01，类型：备份失败）已手动恢复"; got != want {
		t.Errorf("restore detail = %q, want %q", got, want)
	}
	if got, want := alertAuditDetail(alert, "已删除"), "告警内容「备份任务失败」（主机：db-01，类型：备份失败）已删除"; got != want {
		t.Errorf("delete detail = %q, want %q", got, want)
	}
	if got := alertAuditDetail(nil, "已删除"); got != "告警已删除" {
		t.Errorf("nil alert detail = %q", got)
	}
}

func TestNotifyChannelAuditDetail(t *testing.T) {
	channel := &domain.NotifyChannel{
		Name:      "运维告警",
		Type:      "mail",
		EventType: []string{"backup_failed", "not_run"},
	}
	if got, want := notifyChannelAuditDetail(channel, "已创建"), "通知渠道 运维告警 （方式：邮件，类型：备份失败、未按时执行）已创建"; got != want {
		t.Errorf("create detail = %q, want %q", got, want)
	}
	if got, want := notifyChannelAuditDetail(channel, "已删除"), "通知渠道 运维告警 （方式：邮件，类型：备份失败、未按时执行）已删除"; got != want {
		t.Errorf("delete detail = %q, want %q", got, want)
	}
	if got := notifyChannelAuditDetail(nil, "已删除"); got != "通知渠道已删除" {
		t.Errorf("nil channel = %q", got)
	}
}

func TestNotifyChannelChangeItems(t *testing.T) {
	before := domain.NotifyChannel{Name: "运维告警", Type: "mail", Level: "all", EventType: []string{"backup_failed"},
		Config: map[string]any{"recipients": []string{"a@example.com"}}}
	after := before
	if got := notifyChannelChangeItems(before, after); got != "" {
		t.Errorf("unchanged = %q, want empty", got)
	}
	after.Enabled = true
	if got, want := notifyChannelChangeItems(before, after), "：启用已修改"; got != want {
		t.Errorf("enable only = %q, want %q", got, want)
	}
	after.Name = "运维告警2"
	after.Level = "critical"
	after.Recover = true
	after.Remark = "备注"
	after.Config = map[string]any{"recipients": []string{"b@example.com"}}
	want := "：通知名称、告警级别、恢复通知、邮件列表、备注、启用已修改"
	if got := notifyChannelChangeItems(before, after); got != want {
		t.Errorf("full changes = %q, want %q", got, want)
	}
}

func TestAgentTokenChangeItems(t *testing.T) {
	before := domain.AgentToken{Name: "生产令牌", Hostname: "db-01", Enabled: true}
	if got := agentTokenChangeItems(before, before); got != "" {
		t.Errorf("unchanged = %q, want empty", got)
	}
	after := domain.AgentToken{Name: "生产令牌2", Hostname: "db-02", Enabled: false}
	if got, want := agentTokenChangeItems(before, after), "：令牌名称、绑定主机、启用已修改"; got != want {
		t.Errorf("token changes = %q, want %q", got, want)
	}
}

func TestAgentProfileChangeItems(t *testing.T) {
	base := domain.AgentProfile{Name: "默认", Hostname: "*", Config: domain.DefaultProfileConfig()}
	if got := agentProfileChangeItems(base, base); got != "" {
		t.Errorf("unchanged = %q, want empty", got)
	}
	all := domain.AgentProfile{
		Name: "全部改动", Hostname: "db-*", Description: "说明", Enabled: true, Priority: 9,
		Config: domain.AgentProfileConfig{
			ReportIntervalSec: 30, CollectCrontab: false, CollectVeeam: false, ArtifactCheck: false,
			LabelPrefix: "#bm:", MaxArtifactAgeHours: 12, ArtifactMaxDepth: 5,
			MaxSessionsPerJob: 6, MaxSessionInfoCalls: 11, MaxJobInfoCalls: 21,
			HeartbeatIntervalSec: 31, ManualTriggerMinutes: 31, ArtifactPreToleranceMinutes: 11,
			AutoUpdateMinutes: 1, IncludeKeywords: []string{"backup"}, IncludePaths: []string{"/data"},
			ExcludeKeywords: []string{"certbot"},
		},
	}
	want := "：配置名称、匹配主机、优先级、上报间隔（秒）、启用、描述、采集 Crontab、采集 Veeam、产物检查、任务标签前缀、产物最大年龄、产物扫描深度、单作业会话上限、会话详情调用、作业详情调用、心跳间隔、手动执行判定、产物时间容差、自动更新检查、关键字白名单、路径白名单、排除关键字已修改"
	if got := agentProfileChangeItems(base, all); got != want {
		t.Errorf("all changes = %q\nwant %q", got, want)
	}
}

func TestAgentUploadAuditDetail(t *testing.T) {
	got := agentUploadAuditDetail("backup-agent-linux", "linux", "amd64", "standard", "go1.25.4", 5_872_026, "abc123")
	want := "Agent 二进制文件 backup-agent-linux （平台：Linux，架构：amd64，通道：standard，Go：go1.25.4，大小：5.6 MB，sha256：abc123）已上传"
	if got != want {
		t.Errorf("upload detail = %q, want %q", got, want)
	}
	win := agentUploadAuditDetail("backup-agent-windows.exe", "windows", "arm64", "standard", "", 2048, "def")
	if want := "Agent 二进制文件 backup-agent-windows.exe （平台：Windows，架构：arm64，通道：standard，大小：2.0 KB，sha256：def）已上传"; win != want {
		t.Errorf("windows upload detail = %q, want %q", win, want)
	}
}

func TestAgentPushAuditDetail(t *testing.T) {
	const head = "推送 Agent 更新（平台：Linux，版本：1.0.36）"
	both := agentPushAuditDetail("linux", []string{"1.0.36"},
		[]string{"10.0.0.1", "10.0.0.2"}, []string{"10.0.0.3"}, nil)
	if want := head + "，推送成功：10.0.0.1、10.0.0.2，推送失败：10.0.0.3"; both != want {
		t.Errorf("both = %q, want %q", both, want)
	}
	onlySuccess := agentPushAuditDetail("linux", []string{"1.0.36"}, []string{"10.0.0.1"}, nil, nil)
	if want := head + "，推送成功：10.0.0.1"; onlySuccess != want {
		t.Errorf("success only = %q, want %q", onlySuccess, want)
	}
	onlyFailed := agentPushAuditDetail("linux", []string{"1.0.36"}, nil, []string{"10.0.0.9"}, nil)
	if want := head + "，推送失败：10.0.0.9"; onlyFailed != want {
		t.Errorf("failed only = %q, want %q", onlyFailed, want)
	}
	skipped := agentPushAuditDetail("windows", nil, nil, nil, []string{"10.0.0.9"})
	if want := "推送 Agent 更新（平台：Windows），已跳过：10.0.0.9（无匹配安装包）"; skipped != want {
		t.Errorf("skipped = %q, want %q", skipped, want)
	}
	multi := agentPushAuditDetail("windows", []string{"1.0.36", "1.0.35"}, []string{"10.0.0.1"}, nil, nil)
	if want := "推送 Agent 更新（平台：Windows，版本：1.0.36、1.0.35），推送成功：10.0.0.1"; multi != want {
		t.Errorf("multi version = %q, want %q", multi, want)
	}
}

func TestFormatAuditTime(t *testing.T) {
	original := time.Local
	time.Local = time.UTC
	defer func() { time.Local = original }()

	if got, want := formatAuditTime("2026-10-21T02:01:00.000Z"), "2026/10/21 02:01:00"; got != want {
		t.Errorf("formatAuditTime = %q, want %q", got, want)
	}
	if got, want := formatAuditTime("2026-01-02T03:04:05Z"), "2026/01/02 03:04:05"; got != want {
		t.Errorf("formatAuditTime = %q, want %q", got, want)
	}
	if got := formatAuditTime("not-a-time"); got != "not-a-time" {
		t.Errorf("invalid input = %q, want unchanged", got)
	}
	if got := formatAuditTime(""); got != "" {
		t.Errorf("empty input = %q, want empty", got)
	}
}

func TestNotifyTestAuditDetail(t *testing.T) {
	mail := &domain.NotifyChannel{Name: "运维告警", Type: "mail", Config: map[string]any{"recipients": []string{"ops@example.com", "dev@example.com"}}}
	if got, want := notifyTestAuditDetail(mail, "运维告警", true, ""), "通知渠道 运维告警 测试成功，已发送至 ops@example.com、dev@example.com"; got != want {
		t.Errorf("mail success = %q, want %q", got, want)
	}
	duplicated := &domain.NotifyChannel{Name: "重复告警", Type: "mail", Config: map[string]any{"recipients": []any{"hongzelong@sunline.cn", "Hongzelong@Sunline.CN", "ops@example.com"}}}
	if got, want := notifyTestAuditDetail(duplicated, "重复告警", true, ""), "通知渠道 重复告警 测试成功，已发送至 hongzelong@sunline.cn、ops@example.com"; got != want {
		t.Errorf("mail success with duplicates = %q, want %q", got, want)
	}
	webhook := &domain.NotifyChannel{Name: "回调", Type: "webhook"}
	if got, want := notifyTestAuditDetail(webhook, "回调", true, ""), "通知渠道 回调 测试成功"; got != want {
		t.Errorf("webhook success = %q, want %q", got, want)
	}
	if got, want := notifyTestAuditDetail(mail, "运维告警", false, "SMTP 主机未配置"), "通知渠道 运维告警 测试失败：SMTP 主机未配置"; got != want {
		t.Errorf("failure = %q, want %q", got, want)
	}
}

func TestConfigStringList(t *testing.T) {
	if got := configStringList([]any{"a@b.com", " c@d.com ", 1}); len(got) != 2 || got[0] != "a@b.com" || got[1] != "c@d.com" {
		t.Errorf("configStringList = %#v", got)
	}
	if got := configStringList([]string{"x@y.com"}); len(got) != 1 {
		t.Errorf("configStringList([]string) = %#v", got)
	}
	if got := configStringList(nil); got != nil {
		t.Errorf("configStringList(nil) = %#v", got)
	}
}
