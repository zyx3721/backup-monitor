package service_test

import (
	"testing"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
)

func mailChannel(recipients any) domain.NotifyChannel {
	return domain.NotifyChannel{
		Name:      "ops",
		Type:      "mail",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"recipients": recipients},
	}
}

func TestValidateChannelAcceptsRecipientArray(t *testing.T) {
	channel := mailChannel([]string{"ops@example.com", "oncall@example.com"})
	if message := service.ValidateChannel(channel); message != "" {
		t.Fatalf("unexpected validation message: %s", message)
	}
}

func TestValidateChannelRejectsInvalidRecipient(t *testing.T) {
	channel := mailChannel([]string{"ops@example.com", "not-an-email"})
	if message := service.ValidateChannel(channel); message != "收件邮箱格式不正确：not-an-email" {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateChannelRejectsEmptyRecipients(t *testing.T) {
	channel := mailChannel([]string{})
	if message := service.ValidateChannel(channel); message != "请至少填写一个收件邮箱" {
		t.Fatalf("message = %q", message)
	}
}

func TestNormalizeChannelConfigParsesJSONRecipients(t *testing.T) {
	channel := mailChannel("[\"ops@example.com\", \"oncall@example.com\"]")
	service.NormalizeChannelConfig(&channel)
	list, ok := channel.Config["recipients"].([]string)
	if !ok || len(list) != 2 || list[0] != "ops@example.com" || list[1] != "oncall@example.com" {
		t.Fatalf("unexpected recipients: %#v", channel.Config["recipients"])
	}
	if message := service.ValidateChannel(channel); message != "" {
		t.Fatalf("unexpected validation message: %s", message)
	}
}

func TestNormalizeChannelConfigSplitsLegacyRecipients(t *testing.T) {
	channel := mailChannel("ops@example.com, oncall@example.com")
	service.NormalizeChannelConfig(&channel)
	list, ok := channel.Config["recipients"].([]string)
	if !ok || len(list) != 2 {
		t.Fatalf("unexpected recipients: %#v", channel.Config["recipients"])
	}
}

func TestValidateChannelRequiresEventType(t *testing.T) {
	channel := mailChannel([]string{"ops@example.com"})
	channel.EventType = nil
	if message := service.ValidateChannel(channel); message != "请至少选择一个通知事件" {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateChannelRejectsInvalidWebhookHeaders(t *testing.T) {
	channel := domain.NotifyChannel{
		Name:      "hook",
		Type:      "webhook",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"url": "https://example.com/hook", "headers": "{not-json"},
	}
	if message := service.ValidateChannel(channel); message != "请求头不是合法的 JSON" {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateChannelAcceptsWecomAppSecret(t *testing.T) {
	channel := domain.NotifyChannel{
		Name:      "wecom",
		Type:      "wecom",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"corpId": "corp", "appSecret": "secret"},
	}
	if message := service.ValidateChannel(channel); message != "" {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateChannelAcceptsLegacyWecomSecret(t *testing.T) {
	channel := domain.NotifyChannel{
		Name:      "wecom",
		Type:      "wecom",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"corpId": "corp", "secret": "secret"},
	}
	if message := service.ValidateChannel(channel); message != "" {
		t.Fatalf("message = %q", message)
	}
}

func TestNormalizeChannelConfigDefaultsFeishuIdType(t *testing.T) {
	channel := domain.NotifyChannel{Type: "feishu", Config: map[string]any{}}
	service.NormalizeChannelConfig(&channel)
	if channel.Config["receiveIdType"] != "user_id" {
		t.Fatalf("receiveIdType = %v", channel.Config["receiveIdType"])
	}
}

func TestValidateChannelRejectsInvalidFeishuIdType(t *testing.T) {
	channel := domain.NotifyChannel{
		Name:      "feishu",
		Type:      "feishu",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"appKey": "app", "appSecret": "secret", "receiveIdType": "mobile"},
	}
	if message := service.ValidateChannel(channel); message != "请选择有效的 ID 类型" {
		t.Fatalf("message = %q", message)
	}
}

func TestValidateChannelRejectsUnsupportedWebhookMethod(t *testing.T) {
	channel := domain.NotifyChannel{
		Name:      "hook",
		Type:      "webhook",
		Level:     "all",
		EventType: []string{"backup_failed"},
		Config:    map[string]any{"url": "https://example.com/hook", "method": "DELETE"},
	}
	if message := service.ValidateChannel(channel); message != "请选择有效的请求方法" {
		t.Fatalf("message = %q", message)
	}
}
