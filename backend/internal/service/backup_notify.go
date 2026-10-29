package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
)

// NotifyInput 告警通知入参。
type NotifyInput struct {
	Type     string
	Level    string
	Message  string
	HostName string
	HostIP   string
	TaskName string
	Status   string
}

var (
	channelTypes   = map[string]bool{"mail": true, "webhook": true, "wecom": true, "dingtalk": true, "feishu": true}
	eventTypes     = map[string]bool{"backup_failed": true, "not_run": true, "report_timeout": true}
	notifyLevels   = map[string]bool{"all": true, "info": true, "warning": true, "critical": true}
	eventTypeLabel = map[string]string{"backup_failed": "备份失败", "not_run": "未按时执行", "report_timeout": "主机异常/离线"}
	levelOrder     = map[string]int{"info": 1, "warning": 2, "critical": 3}
	levelLabel     = map[string]string{"info": "提示", "warning": "警告", "critical": "严重"}
	emailPattern   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	receiveIdTypes = map[string]bool{"open_id": true, "user_id": true, "union_id": true, "email": true, "chat_id": true}
	tokenCache     = struct {
		sync.Mutex
		items map[string]cachedToken
	}{items: map[string]cachedToken{}}
)

type cachedToken struct {
	value     string
	expiresAt time.Time
}

type renderVars struct {
	Alert     string
	Host      string
	IP        string
	Task      string
	Content   string
	Level     string
	Time      string
	Status    string
	Color     string
	AlertText string
}

// ValidateChannel 校验渠道配置，返回中文错误。
func ValidateChannel(channel domain.NotifyChannel) string {
	if strings.TrimSpace(channel.Name) == "" {
		return "请填写通知名称"
	}
	if !channelTypes[channel.Type] {
		return "请选择有效的通知方式"
	}
	config := channel.Config
	text := func(key string) string {
		if value, ok := config[key].(string); ok {
			return strings.TrimSpace(value)
		}
		return ""
	}
	switch channel.Type {
	case "mail":
		recipients := stringList(config["recipients"])
		if len(recipients) == 0 {
			return "请至少填写一个收件邮箱"
		}
		for _, recipient := range recipients {
			if !emailPattern.MatchString(recipient) {
				return "收件邮箱格式不正确：" + recipient
			}
		}
	case "webhook":
		if text("url") == "" {
			return "请填写请求地址"
		}
		method := strings.ToUpper(firstNonEmpty(text("method"), "POST"))
		if method != "POST" && method != "GET" {
			return "请选择有效的请求方法"
		}
		if headers := text("headers"); headers != "" && !json.Valid([]byte(headers)) {
			return "请求头不是合法的 JSON"
		}
	case "wecom":
		if text("corpId") == "" {
			return "请填写企业 ID"
		}
		if firstNonEmpty(text("appSecret"), text("secret")) == "" {
			return "请填写凭证密钥"
		}
	case "dingtalk", "feishu":
		if text("appKey") == "" {
			return "请填写应用 key"
		}
		if text("appSecret") == "" {
			return "请填写应用密钥"
		}
		if channel.Type == "feishu" && !receiveIdTypes[firstNonEmpty(text("receiveIdType"), "user_id")] {
			return "请选择有效的 ID 类型"
		}
	}
	if channel.Type != "mail" {
		if body := text("body"); body != "" && !json.Valid([]byte(body)) {
			return "请求体不是合法的 JSON"
		}
	}
	if !notifyLevels[channel.Level] {
		return "请选择有效的告警级别"
	}
	if len(channel.EventType) == 0 {
		return "请至少选择一个通知事件"
	}
	for _, event := range channel.EventType {
		if !eventTypes[event] {
			return "请选择有效的事件类型"
		}
	}
	return ""
}

// NormalizeChannelConfig 归一化渠道配置，邮件收件人统一为字符串数组。
func NormalizeChannelConfig(channel *domain.NotifyChannel) {
	if channel.Config == nil {
		channel.Config = map[string]any{}
	}
	if channel.Type == "mail" {
		channel.Config["recipients"] = stringList(channel.Config["recipients"])
	}
	if channel.Type == "feishu" {
		if idType, _ := channel.Config["receiveIdType"].(string); strings.TrimSpace(idType) == "" {
			channel.Config["receiveIdType"] = "user_id"
		}
	}
}

// NotifyAlert 发送告警通知（遵守全局开关、事件类型与级别门槛）。
func (s *BackupService) NotifyAlert(input NotifyInput) error {
	return s.fanout(input, input.Status, false)
}

// NotifyRecover 发送恢复通知（仅发送给勾选恢复通知的渠道）。
func (s *BackupService) NotifyRecover(input NotifyInput) error {
	return s.fanout(input, "已恢复", true)
}

func (s *BackupService) fanout(input NotifyInput, status string, recoverOnly bool) error {
	channels, err := s.repo.ListNotifyChannels()
	if err != nil {
		return err
	}
	if input.Status == "" {
		input.Status = "未处理"
	}
	title := fmt.Sprintf("【备份监控%s】%s：%s", map[bool]string{true: "·已恢复", false: ""}[recoverOnly],
		eventTypeLabel[input.Type], firstNonEmpty(input.TaskName, input.HostName))
	vars := buildRenderVars(input, status)
	for _, channel := range channels {
		if !channel.Enabled || !containsString(channel.EventType, input.Type) || !levelMatches(channel.Level, input.Level) {
			continue
		}
		if recoverOnly && !channel.Recover {
			continue
		}
		ok, sendMessage := s.dispatch(channel, title, vars.AlertText, vars)
		s.recordNotification(channel, input.Type, input.Level, title, input.Message, input.HostName, input.TaskName, sendMessage, ok)
	}
	return nil
}

// TestNotifyChannel 发送测试通知（不受开关与级别限制）。
func (s *BackupService) TestNotifyChannel(id string) (bool, string) {
	channel, err := s.repo.GetNotifyChannel(id)
	if err != nil || channel == nil {
		return false, "通知不存在"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	text := testNotifyText(*channel, now)
	vars := renderVars{Alert: "测试", Host: "测试主机", IP: "127.0.0.1", Task: "测试任务", Content: text,
		Level: "提示", Time: now, Status: "测试", Color: "green", AlertText: text}
	ok, message := s.dispatch(*channel, "【备份监控系统】通知测试", text, vars)
	s.recordNotification(*channel, "test", "", "【备份监控系统】通知测试", text, "", "", message, ok)
	return ok, message
}

func testNotifyText(channel domain.NotifyChannel, now string) string {
	text := strings.Join([]string{
		"这是来自备份监控系统的测试消息。",
		"发送时间：" + now,
		"收到本消息说明该通知配置工作正常。",
	}, "\n")
	if channel.Type == "dingtalk" && dingtalkMarkdown(channel) {
		text = strings.ReplaceAll(text, "\n", "\n\n")
	}
	return text
}

func dingtalkMarkdown(channel domain.NotifyChannel) bool {
	if channel.Type != "dingtalk" {
		return false
	}
	if messageType, _ := channel.Config["messageType"].(string); strings.EqualFold(strings.TrimSpace(messageType), "markdown") {
		return true
	}
	raw, _ := channel.Config["body"].(string)
	if strings.TrimSpace(raw) == "" {
		return false
	}
	var payload struct {
		Msg struct {
			Msgtype string `json:"msgtype"`
		} `json:"msg"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(payload.Msg.Msgtype), "markdown")
}

// Notifications 返回最近的发送记录，channelID 非空时只返回该渠道。
func (s *BackupService) Notifications(limit int, channelID string) ([]domain.NotificationRecord, error) {
	return s.repo.ListNotifications(limit, channelID)
}

func (s *BackupService) recordNotification(channel domain.NotifyChannel, eventType, level, subject, message,
	hostName, taskName, sendMessage string, ok bool) {
	record := domain.NotificationRecord{
		Channel:     channel.Type,
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		Type:        eventType,
		Level:       level,
		Recipients:  channelTarget(channel),
		Subject:     subject,
		Status:      map[bool]string{true: "sent", false: "failed"}[ok],
		Error:       map[bool]string{true: "", false: sendMessage}[ok],
		HostName:    hostName,
		TaskName:    taskName,
		Message:     message,
		CreatedAt:   appcommon.NowTimestamp(),
	}
	if err := s.repo.InsertNotification(record); err != nil {
		log.Printf("Record notification failed: %v", err)
	}
}

func (s *BackupService) dispatch(channel domain.NotifyChannel, title, text string, vars renderVars) (bool, string) {
	config := channel.Config
	textField := func(key string) string {
		if value, ok := config[key].(string); ok {
			return strings.TrimSpace(value)
		}
		return ""
	}
	switch channel.Type {
	case "mail":
		recipients := stringList(config["recipients"])
		if len(recipients) == 0 {
			return false, "未配置收件人"
		}
		if err := s.sendMail(recipients, title, text); err != nil {
			return false, err.Error()
		}
		return true, "已发送"
	case "webhook":
		method := strings.ToUpper(firstNonEmpty(textField("method"), "POST"))
		url := textField("url")
		headers := map[string]string{"Content-Type": "application/json"}
		if raw := textField("headers"); raw != "" {
			parsed := map[string]any{}
			if json.Unmarshal([]byte(raw), &parsed) == nil {
				for key, value := range parsed {
					headers[key] = fmt.Sprint(value)
				}
			}
		}
		body := map[string]any{"content": title + "\n" + text}
		if raw := textField("body"); raw != "" {
			parsed := map[string]any{}
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
				body = parsed
			}
		}
		payload := renderTemplate(body, vars)
		raw, err := json.Marshal(payload)
		if err != nil {
			return false, err.Error()
		}
		request, err := http.NewRequest(method, url, bytes.NewReader(raw))
		if err != nil {
			return false, err.Error()
		}
		for key, value := range headers {
			request.Header.Set(key, value)
		}
		response, err := httpClient().Do(request)
		if err != nil {
			return false, err.Error()
		}
		defer response.Body.Close()
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return false, fmt.Sprintf("HTTP %d %s", response.StatusCode, string(responseBody))
		}
		return true, "已发送"
	case "wecom":
		token, err := s.wecomToken(textField("corpId"), firstNonEmpty(textField("appSecret"), textField("secret")))
		if err != nil {
			return false, err.Error()
		}
		template := map[string]any{"msgtype": "text", "text": map[string]any{"content": title + "\n" + text}}
		if raw := textField("body"); raw != "" {
			parsed := map[string]any{}
			if json.Unmarshal([]byte(raw), &parsed) == nil {
				template = parsed
			}
		}
		return s.postJSON("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+token, renderTemplate(template, vars), "errcode")
	case "dingtalk":
		token, err := s.dingtalkToken(textField("appKey"), textField("appSecret"))
		if err != nil {
			return false, err.Error()
		}
		template := map[string]any{"agent_id": "", "userid_list": "",
			"msg": map[string]any{"msgtype": "text", "text": map[string]any{"content": title + "\n" + text}}}
		if raw := textField("body"); raw != "" {
			parsed := map[string]any{}
			if json.Unmarshal([]byte(raw), &parsed) == nil {
				template = parsed
			}
		}
		return s.postJSON("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token="+token, renderTemplate(template, vars), "errcode")
	case "feishu":
		token, err := s.feishuToken(textField("appKey"), textField("appSecret"))
		if err != nil {
			return false, err.Error()
		}
		receiveIDType := firstNonEmpty(textField("receiveIdType"), "user_id")
		template := map[string]any{"msg_type": "text", "content": "{\"text\":\"" + escapeJSON(title+"\n"+text) + "\"}"}
		if raw := textField("body"); raw != "" {
			parsed := map[string]any{}
			if json.Unmarshal([]byte(raw), &parsed) == nil {
				template = parsed
			}
		}
		url := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveIDType
		headers := map[string]string{"Authorization": "Bearer " + token}
		payload := normalizeFeishuContent(renderTemplate(template, vars))
		if body, ok := payload.(map[string]any); ok {
			return s.sendFeishuMessages(url, body, headers)
		}
		return s.postJSONWithHeaders(url, payload, "code", headers)
	}
	return false, "不支持的通知方式：" + channel.Type
}

func (s *BackupService) sendFeishuMessages(url string, payload map[string]any, headers map[string]string) (bool, string) {
	raw, _ := payload["receive_id"].(string)
	ids := splitFeishuReceiveIDs(raw)
	if len(ids) <= 1 {
		if len(ids) == 1 {
			payload["receive_id"] = ids[0]
		}
		return s.postJSONWithHeaders(url, payload, "code", headers)
	}
	sent := 0
	failed := make([]string, 0, len(ids))
	for _, id := range ids {
		body := make(map[string]any, len(payload))
		for key, value := range payload {
			body[key] = value
		}
		body["receive_id"] = id
		ok, _ := s.postJSONWithHeaders(url, body, "code", headers)
		if ok {
			sent++
			continue
		}
		failed = append(failed, id)
	}
	if len(failed) == 0 {
		return true, fmt.Sprintf("已发送 %d 个接收人", sent)
	}
	details := failed
	if len(details) > 3 {
		details = append(append([]string{}, failed[:3]...), fmt.Sprintf("等 %d 个", len(failed)))
	}
	return false, fmt.Sprintf("成功 %d 个，失败 %d 个：%s", sent, len(failed), strings.Join(details, "、"))
}

func splitFeishuReceiveIDs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', ' ', '\n', '\r', '\t':
			return true
		}
		return false
	})
	seen := make(map[string]bool, len(fields))
	ids := make([]string, 0, len(fields))
	for _, field := range fields {
		id := strings.TrimSpace(field)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func normalizeFeishuContent(payload any) any {
	body, ok := payload.(map[string]any)
	if !ok {
		return payload
	}
	switch content := body["content"].(type) {
	case map[string]any, []any:
		raw, err := json.Marshal(content)
		if err != nil {
			return payload
		}
		body["content"] = string(raw)
	}
	return body
}

func (s *BackupService) postJSON(url string, payload any, codeField string) (bool, string) {
	return s.postJSONWithHeaders(url, payload, codeField, nil)
}

func (s *BackupService) postJSONWithHeaders(url string, payload any, codeField string, headers map[string]string) (bool, string) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err.Error()
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return false, err.Error()
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := httpClient().Do(request)
	if err != nil {
		return false, err.Error()
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Sprintf("HTTP %d %s", response.StatusCode, string(body))
	}
	parsed := map[string]any{}
	if json.Unmarshal(body, &parsed) == nil {
		if code, ok := parsed[codeField]; ok && fmt.Sprint(code) != "0" {
			return false, string(body)
		}
	}
	return true, "已发送"
}

func (s *BackupService) wecomToken(corpID, secret string) (string, error) {
	key := "wecom:" + corpID
	if token, ok := cachedTokenValue(key); ok {
		return token, nil
	}
	url := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=" + corpID + "&corpsecret=" + secret
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := getJSON(url, &parsed); err != nil {
		return "", err
	}
	if parsed.ErrCode != 0 || parsed.AccessToken == "" {
		return "", fmt.Errorf("获取企业微信 access_token 失败：%s", parsed.ErrMsg)
	}
	storeToken(key, parsed.AccessToken, parsed.ExpiresIn)
	return parsed.AccessToken, nil
}

func (s *BackupService) dingtalkToken(appKey, appSecret string) (string, error) {
	key := "dingtalk:" + appKey
	if token, ok := cachedTokenValue(key); ok {
		return token, nil
	}
	url := "https://oapi.dingtalk.com/gettoken?appkey=" + appKey + "&appsecret=" + appSecret
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := getJSON(url, &parsed); err != nil {
		return "", err
	}
	if parsed.ErrCode != 0 || parsed.AccessToken == "" {
		return "", fmt.Errorf("获取钉钉 access_token 失败：%s", parsed.ErrMsg)
	}
	storeToken(key, parsed.AccessToken, parsed.ExpiresIn)
	return parsed.AccessToken, nil
}

func (s *BackupService) feishuToken(appKey, appSecret string) (string, error) {
	key := "feishu:" + appKey
	if token, ok := cachedTokenValue(key); ok {
		return token, nil
	}
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	payload := map[string]string{"app_id": appKey, "app_secret": appSecret}
	raw, _ := json.Marshal(payload)
	response, err := httpClient().Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	var parsed struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Token  string `json:"tenant_access_token"`
		Expire int    `json:"expire"`
	}
	if json.Unmarshal(body, &parsed) != nil || parsed.Code != 0 || parsed.Token == "" {
		return "", fmt.Errorf("获取飞书 tenant_access_token 失败：%s", parsed.Msg)
	}
	storeToken(key, parsed.Token, parsed.Expire)
	return parsed.Token, nil
}

func cachedTokenValue(key string) (string, bool) {
	tokenCache.Lock()
	defer tokenCache.Unlock()
	item, ok := tokenCache.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		return "", false
	}
	return item.value, true
}

func storeToken(key, value string, expiresIn int) {
	if expiresIn <= 300 {
		expiresIn = 7200
	}
	tokenCache.Lock()
	tokenCache.items[key] = cachedToken{value: value, expiresAt: time.Now().Add(time.Duration(expiresIn-300) * time.Second)}
	tokenCache.Unlock()
}

func getJSON(url string, target any) error {
	response, err := httpClient().Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8192))
	return json.Unmarshal(body, target)
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func buildRenderVars(input NotifyInput, status string) renderVars {
	host := input.HostName
	if input.HostIP != "" {
		host = input.HostName + "（" + input.HostIP + "）"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	alert := eventTypeLabel[input.Type]
	lines := []string{"告警：" + alert, "主机：" + host}
	if input.TaskName != "" {
		lines = append(lines, "任务："+input.TaskName)
	}
	lines = append(lines, "内容："+input.Message, "级别："+firstNonEmpty(levelLabel[input.Level], input.Level),
		"时间："+now, "状态："+status)
	return renderVars{Alert: alert, Host: input.HostName, IP: input.HostIP, Task: input.TaskName, Content: input.Message,
		Level: firstNonEmpty(levelLabel[input.Level], input.Level), Time: now, Status: status, Color: notifyColor(status),
		AlertText: strings.Join(lines, "\n")}
}

func notifyColor(status string) string {
	if status == "测试" || status == "已恢复" {
		return "green"
	}
	return "red"
}

func renderTemplate(value any, vars renderVars) any {
	switch typed := value.(type) {
	case string:
		return renderString(typed, vars)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = renderTemplate(item, vars)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			out[key] = renderTemplate(item, vars)
		}
		return out
	}
	return value
}

func renderString(text string, vars renderVars) string {
	pairs := map[string]string{"告警": vars.Alert, "主机": vars.Host, "IP": vars.IP,
		"任务": firstNonEmpty(vars.Task, vars.Host),
		"内容": vars.Content, "级别": vars.Level, "时间": vars.Time, "状态": vars.Status,
		"颜色": firstNonEmpty(vars.Color, notifyColor(vars.Status))}
	out := text
	for key, value := range pairs {
		if value == "" {
			continue
		}
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
		out = strings.ReplaceAll(out, "${"+key+"}", value)
	}
	return strings.ReplaceAll(out, "告警内容", vars.AlertText)
}

// ChannelTarget 返回渠道目标摘要（列表展示用）。
func ChannelTarget(channel domain.NotifyChannel) string {
	return channelTarget(channel)
}

func channelTarget(channel domain.NotifyChannel) string {
	config := channel.Config
	text := func(key string) string {
		if value, ok := config[key].(string); ok {
			return strings.TrimSpace(value)
		}
		return ""
	}
	switch channel.Type {
	case "mail":
		return strings.Join(stringList(config["recipients"]), ", ")
	case "webhook":
		return text("url")
	case "wecom":
		if corpID := text("corpId"); corpID != "" {
			return "企业ID：" + corpID
		}
	case "dingtalk", "feishu":
		if appKey := text("appKey"); appKey != "" {
			return "应用Key：" + appKey
		}
	}
	return ""
}

func stringList(value any) []string {
	out := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
	case []string:
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
	case string:
		if parsed, ok := parseJSONStringList(typed); ok {
			return parsed
		}
		for _, item := range strings.FieldsFunc(typed, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
	}
	return out
}

func parseJSONStringList(value string) ([]string, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "[") {
		return nil, false
	}
	var list []string
	if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
		return nil, false
	}
	out := []string{}
	for _, item := range list {
		if text := strings.TrimSpace(item); text != "" {
			out = append(out, text)
		}
	}
	return out, true
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func levelMatches(configLevel, alertLevel string) bool {
	if configLevel == "" || configLevel == "all" {
		return true
	}
	return levelOrder[alertLevel] >= levelOrder[configLevel]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeJSON(text string) string {
	raw, _ := json.Marshal(text)
	quoted := string(raw)
	if len(quoted) >= 2 {
		return quoted[1 : len(quoted)-1]
	}
	return quoted
}
