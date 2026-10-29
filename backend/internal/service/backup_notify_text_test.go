package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"backup-monitor-backend/internal/domain"
)

func TestTestNotifyTextKeepsSingleNewlines(t *testing.T) {
	text := testNotifyText(domain.NotifyChannel{Type: "dingtalk", Config: map[string]any{}}, "2026-10-30 01:46:49")
	if strings.Contains(text, "\n\n") {
		t.Fatalf("text should not contain blank lines: %q", text)
	}
	if !strings.Contains(text, "\n发送时间：") {
		t.Fatalf("text should keep single newline before time: %q", text)
	}
	if !strings.Contains(text, "\n收到本消息") {
		t.Fatalf("text should keep single newline before received line: %q", text)
	}
}

func TestTestNotifyTextDoublesDingtalkMarkdown(t *testing.T) {
	channel := domain.NotifyChannel{Type: "dingtalk", Config: map[string]any{"messageType": "markdown"}}
	text := testNotifyText(channel, "2026-10-30 01:46:49")
	if !strings.Contains(text, "\n\n发送时间：") {
		t.Fatalf("markdown text should double newlines: %q", text)
	}
}

func TestTestNotifyTextDetectsDingtalkMarkdownBody(t *testing.T) {
	channel := domain.NotifyChannel{
		Type:   "dingtalk",
		Config: map[string]any{"body": `{"agent_id":"","userid_list":"","msg":{"msgtype":"markdown","markdown":{"title":"t","text":"x"}}}`},
	}
	if !dingtalkMarkdown(channel) {
		t.Fatal("body with markdown msgtype should be detected")
	}
	if !strings.Contains(testNotifyText(channel, "now"), "\n\n") {
		t.Fatal("detected markdown text should double newlines")
	}
}

func TestDingtalkMarkdownIgnoresOtherChannels(t *testing.T) {
	if dingtalkMarkdown(domain.NotifyChannel{Type: "wecom", Config: map[string]any{"messageType": "markdown"}}) {
		t.Fatal("wecom should not be treated as dingtalk markdown")
	}
}

func TestNormalizeFeishuContentObject(t *testing.T) {
	payload := normalizeFeishuContent(map[string]any{
		"msg_type": "interactive",
		"content": map[string]any{
			"config": map[string]any{"wide_screen_mode": true},
			"elements": []any{
				map[string]any{"tag": "div", "text": map[string]any{"tag": "lark_md", "content": "line1\nline2"}},
			},
		},
	})
	body, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T", payload)
	}
	content, ok := body["content"].(string)
	if !ok {
		t.Fatalf("content type = %T, want string", body["content"])
	}
	if !json.Valid([]byte(content)) {
		t.Fatalf("content is not valid JSON: %s", content)
	}
	if !strings.Contains(content, `"line1\nline2"`) {
		t.Fatalf("content lost markdown text: %s", content)
	}
}

func TestNormalizeFeishuContentKeepsString(t *testing.T) {
	payload := normalizeFeishuContent(map[string]any{"msg_type": "text", "content": `{"text":"hello"}`})
	body := payload.(map[string]any)
	if body["content"] != `{"text":"hello"}` {
		t.Fatalf("content = %v", body["content"])
	}
}

func TestSplitFeishuReceiveIDs(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{raw: "", want: []string{}},
		{raw: "ou_a", want: []string{"ou_a"}},
		{raw: " ou_a , ou_b ", want: []string{"ou_a", "ou_b"}},
		{raw: "ou_a，ou_b；ou_c", want: []string{"ou_a", "ou_b", "ou_c"}},
		{raw: "ou_a,ou_a\nou_b", want: []string{"ou_a", "ou_b"}},
	}
	for _, test := range tests {
		if got := splitFeishuReceiveIDs(test.raw); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("splitFeishuReceiveIDs(%q) = %v, want %v", test.raw, got, test.want)
		}
	}
}

func TestNotifyColor(t *testing.T) {
	tests := map[string]string{"测试": "green", "已恢复": "green", "未处理": "red", "告警": "red"}
	for status, want := range tests {
		if got := notifyColor(status); got != want {
			t.Fatalf("notifyColor(%q) = %q, want %q", status, got, want)
		}
	}
	if got := renderString("{{颜色}}", renderVars{Status: "已恢复"}); got != "green" {
		t.Fatalf("renderString color = %q, want green", got)
	}
}

func TestRenderStringTaskFallsBackToHost(t *testing.T) {
	if got := renderString("【备份监控】主机异常/离线：{{任务}}", renderVars{Host: "www-site-50"}); got != "【备份监控】主机异常/离线：www-site-50" {
		t.Fatalf("renderString task fallback = %q", got)
	}
}
