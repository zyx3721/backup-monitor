package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"backup-monitor-backend/internal/domain"
)

var notifyTypeLabels = map[string]string{
	"mail":     "邮件",
	"webhook":  "WebHook",
	"wecom":    "企业微信",
	"dingtalk": "钉钉",
	"feishu":   "飞书",
}

var notifyEventLabels = map[string]string{
	"backup_failed":  "备份失败",
	"not_run":        "未按时执行",
	"report_timeout": "主机异常/离线",
}

var alertTypeLabels = map[string]string{
	"backup_failed":  "备份失败",
	"not_run":        "未按时执行",
	"report_timeout": "主机异常/离线",
}

func labelOrRaw(labels map[string]string, key string) string {
	if value, ok := labels[key]; ok {
		return value
	}
	return key
}

var notifyConfigFields = map[string][]struct{ Key, Label string }{
	"mail": {
		{"recipients", "邮件列表"},
	},
	"webhook": {
		{"method", "请求方法"},
		{"url", "请求地址"},
		{"headers", "请求头"},
		{"body", "请求体"},
	},
	"wecom": {
		{"corpId", "企业 ID"},
		{"appSecret", "凭证密钥"},
		{"body", "请求体"},
	},
	"dingtalk": {
		{"appKey", "应用 key"},
		{"appSecret", "应用密钥"},
		{"body", "请求体"},
	},
	"feishu": {
		{"appKey", "应用 key"},
		{"appSecret", "应用密钥"},
		{"receiveIdType", "ID 类型"},
		{"body", "请求体"},
	},
}

func joinChangeItems(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "：" + strings.Join(items, "、") + "已修改"
}

// notifyChannelChangeItems 生成通知渠道编辑的变更项后缀，字段名与编辑窗口一致
func notifyChannelChangeItems(before, after domain.NotifyChannel) string {
	changes := make([]string, 0, 10)
	if before.Name != after.Name {
		changes = append(changes, "通知名称")
	}
	if !equalStringList(before.EventType, after.EventType) {
		changes = append(changes, "通知事件")
	}
	if before.Level != after.Level {
		changes = append(changes, "告警级别")
	}
	if before.Recover != after.Recover {
		changes = append(changes, "恢复通知")
	}
	configMatched := false
	for _, field := range notifyConfigFields[before.Type] {
		if !equalJSONValue(before.Config[field.Key], after.Config[field.Key]) {
			changes = append(changes, field.Label)
			configMatched = true
		}
	}
	if !configMatched && !equalJSONValue(before.Config, after.Config) {
		changes = append(changes, "渠道配置")
	}
	if before.Remark != after.Remark {
		changes = append(changes, "备注")
	}
	if before.Enabled != after.Enabled {
		changes = append(changes, "启用")
	}
	return joinChangeItems(changes)
}

// agentTokenChangeItems 生成 Agent 令牌编辑的变更项后缀，字段名与令牌界面一致
func agentTokenChangeItems(before, after domain.AgentToken) string {
	changes := make([]string, 0, 3)
	if before.Name != after.Name {
		changes = append(changes, "令牌名称")
	}
	if before.Hostname != after.Hostname {
		changes = append(changes, "绑定主机")
	}
	if before.Enabled != after.Enabled {
		changes = append(changes, "启用")
	}
	return joinChangeItems(changes)
}

// agentProfileChangeItems 生成采集配置编辑的变更项后缀，顺序与编辑窗口字段一致
func agentProfileChangeItems(before, after domain.AgentProfile) string {
	changes := make([]string, 0, 22)
	if before.Name != after.Name {
		changes = append(changes, "配置名称")
	}
	if before.Hostname != after.Hostname {
		changes = append(changes, "匹配主机")
	}
	if before.Priority != after.Priority {
		changes = append(changes, "优先级")
	}
	if before.Config.ReportIntervalSec != after.Config.ReportIntervalSec {
		changes = append(changes, "上报间隔（秒）")
	}
	if before.Enabled != after.Enabled {
		changes = append(changes, "启用")
	}
	if before.Description != after.Description {
		changes = append(changes, "描述")
	}
	if before.Config.CollectCrontab != after.Config.CollectCrontab {
		changes = append(changes, "采集 Crontab")
	}
	if before.Config.CollectVeeam != after.Config.CollectVeeam {
		changes = append(changes, "采集 Veeam")
	}
	if before.Config.ArtifactCheck != after.Config.ArtifactCheck {
		changes = append(changes, "产物检查")
	}
	if before.Config.LabelPrefix != after.Config.LabelPrefix {
		changes = append(changes, "任务标签前缀")
	}
	if before.Config.MaxArtifactAgeHours != after.Config.MaxArtifactAgeHours {
		changes = append(changes, "产物最大年龄")
	}
	if before.Config.ArtifactMaxDepth != after.Config.ArtifactMaxDepth {
		changes = append(changes, "产物扫描深度")
	}
	if before.Config.MaxSessionsPerJob != after.Config.MaxSessionsPerJob {
		changes = append(changes, "单作业会话上限")
	}
	if before.Config.MaxSessionInfoCalls != after.Config.MaxSessionInfoCalls {
		changes = append(changes, "会话详情调用")
	}
	if before.Config.MaxJobInfoCalls != after.Config.MaxJobInfoCalls {
		changes = append(changes, "作业详情调用")
	}
	if before.Config.HeartbeatIntervalSec != after.Config.HeartbeatIntervalSec {
		changes = append(changes, "心跳间隔")
	}
	if before.Config.ManualTriggerMinutes != after.Config.ManualTriggerMinutes {
		changes = append(changes, "手动执行判定")
	}
	if before.Config.ArtifactPreToleranceMinutes != after.Config.ArtifactPreToleranceMinutes {
		changes = append(changes, "产物时间容差")
	}
	if before.Config.AutoUpdateMinutes != after.Config.AutoUpdateMinutes {
		changes = append(changes, "自动更新检查")
	}
	if !equalStringList(before.Config.IncludeKeywords, after.Config.IncludeKeywords) {
		changes = append(changes, "关键字白名单")
	}
	if !equalStringList(before.Config.IncludePaths, after.Config.IncludePaths) {
		changes = append(changes, "路径白名单")
	}
	if !equalStringList(before.Config.ExcludeKeywords, after.Config.ExcludeKeywords) {
		changes = append(changes, "排除关键字")
	}
	return joinChangeItems(changes)
}

// alertAuditDetail 生成告警审计详情，action 为手动恢复或删除
func alertAuditDetail(alert *domain.Alert, action string) string {
	if alert == nil {
		return "告警" + action
	}
	return fmt.Sprintf("告警内容「%s」（主机：%s，类型：%s）%s",
		alert.Message, alert.HostName, labelOrRaw(alertTypeLabels, alert.Type), action)
}

// notifyChannelAuditDetail 生成通知渠道创建与删除的审计详情，action 传“已创建”或“已删除”
func notifyChannelAuditDetail(channel *domain.NotifyChannel, action string) string {
	if channel == nil {
		return "通知渠道" + action
	}
	types := make([]string, 0, len(channel.EventType))
	for _, item := range channel.EventType {
		types = append(types, labelOrRaw(notifyEventLabels, item))
	}
	return fmt.Sprintf("通知渠道 %s （方式：%s，类型：%s）%s",
		channel.Name, labelOrRaw(notifyTypeLabels, channel.Type), strings.Join(types, "、"), action)
}

// binarySizeText 按 1024 进制格式化二进制大小，与界面显示保持一致
func binarySizeText(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	value := float64(size)
	index := 0
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	if index == 0 {
		return fmt.Sprintf("%.0f %s", value, units[index])
	}
	return fmt.Sprintf("%.1f %s", value, units[index])
}

// equalStringList 忽略顺序与首尾空白比较字符串集合，默认关键字带尾空格时也能稳定判等
func equalStringList(left, right []string) bool {
	leftCopy := trimStringList(left)
	rightCopy := trimStringList(right)
	if len(leftCopy) != len(rightCopy) {
		return false
	}
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func trimStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

// configStringList 读取渠道配置中的字符串列表，兼容 JSON 解码出的 []any
func configStringList(raw any) []string {
	switch value := raw.(type) {
	case []string:
		return value
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	return nil
}

// uniqueRecipients 按不区分大小写去重收件人，保留首次出现的写法
func uniqueRecipients(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

// equalJSONValue 按 JSON 语义比较两个配置值，忽略 []any 与 []string 等具体类型差异
func equalJSONValue(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return bytes.Equal(leftJSON, rightJSON)
}

// agentBinaryToolchain 生成审计里的 Go 版本片段，未知时返回空串
func agentBinaryToolchain(goVersion string) string {
	if strings.TrimSpace(goVersion) == "" {
		return ""
	}
	return "Go：" + goVersion + "，"
}

// agentUploadAuditDetail 生成二进制上传的审计详情
func agentUploadAuditDetail(name, platform, arch, variant, goVersion string, size int64, sha256 string) string {
	return fmt.Sprintf("Agent 二进制文件 %s （平台：%s，架构：%s，通道：%s，%s大小：%s，sha256：%s）已上传",
		name, platformLabel(platform), arch, variant, agentBinaryToolchain(goVersion), binarySizeText(size), sha256)
}

// agentPushAuditDetail 生成二进制推送的审计详情，成功、失败或跳过为空时省略对应片段
func agentPushAuditDetail(platform string, versions []string, successIPs, failedIPs, skippedIPs []string) string {
	head := fmt.Sprintf("推送 Agent 更新（平台：%s", platformLabel(platform))
	if len(versions) > 0 {
		head += "，版本：" + strings.Join(versions, "、")
	}
	head += "）"
	parts := make([]string, 0, 3)
	if len(successIPs) > 0 {
		parts = append(parts, "推送成功："+strings.Join(successIPs, "、"))
	}
	if len(failedIPs) > 0 {
		parts = append(parts, "推送失败："+strings.Join(failedIPs, "、"))
	}
	if len(skippedIPs) > 0 {
		parts = append(parts, "已跳过："+strings.Join(skippedIPs, "、")+"（无匹配安装包）")
	}
	if len(parts) == 0 {
		return head
	}
	return head + "，" + strings.Join(parts, "，")
}

// formatAuditTime 按界面口径（本地时区、24 小时制）格式化审计里的时间
func formatAuditTime(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return value
	}
	return parsed.In(time.Local).Format("2006/01/02 15:04:05")
}

// notifyTestAuditDetail 生成通知渠道测试的审计详情，邮件成功时补收件人
func notifyTestAuditDetail(channel *domain.NotifyChannel, name string, ok bool, message string) string {
	if !ok {
		return "通知渠道 " + name + " 测试失败：" + message
	}
	detail := "通知渠道 " + name + " 测试成功"
	if channel != nil && channel.Type == "mail" {
		if recipients := uniqueRecipients(configStringList(channel.Config["recipients"])); len(recipients) > 0 {
			detail += "，已发送至 " + strings.Join(recipients, "、")
		}
	}
	return detail
}
