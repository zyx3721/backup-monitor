package backup

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

func (a *Router) handleSettings(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, a.svc.Settings())
}

func (a *Router) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	section := ""
	if raw, ok := body["section"].(string); ok {
		section = strings.TrimSpace(raw)
	}
	delete(body, "section")
	previous := a.svc.Settings()
	settings := domain.NormalizeAppSettings(body)
	if errText := domain.ValidateAppSettings(body); errText != "" {
		common.WriteError(w, http.StatusBadRequest, errText)
		return
	}
	if err := a.svc.SaveSettings(settings); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if a.hub != nil {
		if !previous.RequireAgentToken && settings.RequireAgentToken {
			a.hub.CloseAll()
		}
		a.hub.PushConfigChanged()
	}
	if settings != previous {
		a.recordSettingsAudit(r, section, previous, settings)
	}
	common.WriteJSON(w, http.StatusOK, settings)
}

func (a *Router) recordSettingsAudit(r *http.Request, section string, previous, current domain.AppSettings) {
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		return
	}
	target, detail := settingsAuditText(section)
	detail += settingsChangeItems(previous, current, section)
	_ = a.audit.RecordEvent(r.Context(), user.Username, common.ClientIP(r), service.AuditEvent{
		Module: service.AuditModuleSettings, Action: "更新基础配置", Target: target, Detail: detail, Result: service.AuditResultSuccess,
	})
}

// settingsAuditText 备份参数按保存区块细分审计目标与详情
func settingsAuditText(section string) (string, string) {
	switch section {
	case domain.BaseSectionThresholds:
		return "判定与保留", "判定与保留配置已更新"
	case domain.BaseSectionAgent:
		return "Agent 通道", "Agent 通道配置已更新"
	case domain.BaseSectionAlarm:
		return "安全与告警", "安全与告警配置已更新"
	}
	return "备份监控基础参数", "备份监控基础参数已更新"
}

// settingsChangeItems 生成备份参数变更项后缀，如“：主机上报超时、告警去重已修改”；无字段变化时返回空串
func settingsChangeItems(before, after domain.AppSettings, section string) string {
	changes := make([]string, 0, 14)
	if section == "" || section == domain.BaseSectionThresholds {
		if before.ReportTimeoutMinutes != after.ReportTimeoutMinutes {
			changes = append(changes, "主机上报超时")
		}
		if before.OfflineTimeoutMinutes != after.OfflineTimeoutMinutes {
			changes = append(changes, "主机离线阈值")
		}
		if before.NotRunGraceMinutes != after.NotRunGraceMinutes {
			changes = append(changes, "未执行宽限期")
		}
		if before.ScheduleMatchWindowMinutes != after.ScheduleMatchWindowMinutes {
			changes = append(changes, "计划匹配窗口")
		}
		if before.SchedulerIntervalSec != after.SchedulerIntervalSec {
			changes = append(changes, "巡检间隔")
		}
		if before.RetentionDays != after.RetentionDays {
			changes = append(changes, "执行记录保留")
		}
		if before.NotifyRetentionDays != after.NotifyRetentionDays {
			changes = append(changes, "发送记录保留")
		}
		if before.AuditRetentionDays != after.AuditRetentionDays {
			changes = append(changes, "审计日志保留")
		}
	}
	if section == "" || section == domain.BaseSectionAgent {
		if before.Agent.ReportIntervalSec != after.Agent.ReportIntervalSec {
			changes = append(changes, "默认上报间隔")
		}
		if before.Agent.HTTPTimeoutSec != after.Agent.HTTPTimeoutSec {
			changes = append(changes, "HTTP 超时")
		}
		if before.Agent.WSBackoffMinSec != after.Agent.WSBackoffMinSec {
			changes = append(changes, "WS 重连退避下限")
		}
		if before.Agent.WSBackoffMaxSec != after.Agent.WSBackoffMaxSec {
			changes = append(changes, "WS 重连退避上限")
		}
	}
	if section == "" || section == domain.BaseSectionAlarm {
		if before.DedupeAlerts != after.DedupeAlerts {
			changes = append(changes, "告警去重")
		}
		if before.RequireAgentToken != after.RequireAgentToken {
			changes = append(changes, "要求 Agent 令牌")
		}
	}
	if len(changes) == 0 {
		return ""
	}
	return "：" + strings.Join(changes, "、") + "已修改"
}

func (a *Router) handleProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := a.svc.Repo().ListAgentProfiles()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, profiles)
}

type profileBody struct {
	Name        string         `json:"name"`
	Hostname    string         `json:"hostname"`
	Description string         `json:"description"`
	Enabled     *bool          `json:"enabled"`
	Priority    *int           `json:"priority"`
	Config      map[string]any `json:"config"`
}

func (a *Router) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var body profileBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		common.WriteError(w, http.StatusBadRequest, "配置名称不能为空")
		return
	}
	if strings.TrimSpace(body.Hostname) == "" {
		common.WriteError(w, http.StatusBadRequest, "主机匹配不能为空")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	priority := 0
	if body.Priority != nil {
		priority = *body.Priority
	}
	profile := domain.AgentProfile{Name: strings.TrimSpace(body.Name), Hostname: strings.TrimSpace(body.Hostname),
		Description: strings.TrimSpace(body.Description), Enabled: enabled, Priority: priority,
		Config: domain.NormalizeProfileConfig(body.Config)}
	if err := a.svc.Repo().CreateAgentProfile(profile); err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			common.WriteError(w, http.StatusConflict, "配置名称已存在")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "创建配置失败")
		return
	}
	if a.hub != nil {
		a.hub.PushConfigChanged()
	}
	a.recordAudit(r, service.AuditModuleAgents, "创建采集配置", profile.Name, "采集配置 "+profile.Name+" 已创建", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, profile)
}

func (a *Router) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := a.svc.Repo().GetAgentProfile(id)
	if err != nil || existing == nil {
		common.WriteError(w, http.StatusNotFound, "配置不存在")
		return
	}
	previous := *existing
	var body profileBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	if strings.TrimSpace(body.Name) != "" {
		existing.Name = strings.TrimSpace(body.Name)
	}
	if strings.TrimSpace(body.Hostname) != "" {
		existing.Hostname = strings.TrimSpace(body.Hostname)
	}
	existing.Description = strings.TrimSpace(body.Description)
	if body.Enabled != nil {
		existing.Enabled = *body.Enabled
	}
	if body.Priority != nil {
		existing.Priority = *body.Priority
	}
	if body.Config != nil {
		existing.Config = domain.NormalizeProfileConfig(body.Config)
	}
	if err := a.svc.Repo().UpdateAgentProfile(id, *existing); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "更新配置失败")
		return
	}
	if a.hub != nil {
		a.hub.PushConfigChanged()
	}
	if changes := agentProfileChangeItems(previous, *existing); changes != "" {
		a.recordAudit(r, service.AuditModuleAgents, "更新采集配置", previous.Name, "采集配置 "+previous.Name+" 已更新"+changes, service.AuditResultSuccess)
	}
	common.WriteJSON(w, http.StatusOK, existing)
}

func (a *Router) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	profile, _ := a.svc.Repo().GetAgentProfile(id)
	ok, err := a.svc.Repo().DeleteAgentProfile(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "配置不存在")
		return
	}
	name := id
	if profile != nil {
		name = profile.Name
	}
	if a.hub != nil {
		a.hub.PushConfigChanged()
	}
	a.recordAudit(r, service.AuditModuleAgents, "删除采集配置", name, "采集配置 "+name+" 已删除", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := a.svc.Repo().ListAgentTokens()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, tokens)
}

func (a *Router) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		common.WriteError(w, http.StatusBadRequest, "令牌名称不能为空")
		return
	}
	manual := strings.TrimSpace(body.Token)
	token := manual
	if manual == "" {
		generated, err := generateToken()
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, "生成令牌失败")
			return
		}
		token = generated
	} else {
		if err := validateManualToken(manual); err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		exists, err := a.svc.Repo().AgentTokenExists(manual)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, "读取令牌失败")
			return
		}
		if exists {
			common.WriteError(w, http.StatusConflict, "令牌已存在，请更换")
			return
		}
	}
	if err := a.svc.Repo().CreateAgentToken(name, strings.TrimSpace(body.Hostname), token); err != nil {
		if isUniqueTokenError(err) {
			common.WriteError(w, http.StatusConflict, "令牌已存在，请更换")
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "创建令牌失败")
		return
	}
	a.recordAudit(r, service.AuditModuleAgents, "创建 Agent 令牌", name, "Agent 令牌 "+name+" 已创建", service.AuditResultSuccess)
	tokens, _ := a.svc.Repo().ListAgentTokens()
	for _, item := range tokens {
		if item.Token == token {
			common.WriteJSON(w, http.StatusOK, item)
			return
		}
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"name": name, "token": token})
}

func (a *Router) handleUpdateToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	tokens, err := a.svc.Repo().ListAgentTokens()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var existing *domain.AgentToken
	for i := range tokens {
		if tokens[i].ID == id {
			existing = &tokens[i]
			break
		}
	}
	if existing == nil {
		common.WriteError(w, http.StatusNotFound, "令牌不存在")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = existing.Name
	}
	hostname := existing.Hostname
	if body.Hostname != "" {
		hostname = strings.TrimSpace(body.Hostname)
	}
	enabled := existing.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if err := a.svc.Repo().UpdateAgentToken(id, name, hostname, enabled); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "更新令牌失败")
		return
	}
	nameChanged := name != existing.Name
	hostChanged := hostname != existing.Hostname
	enabledChanged := enabled != existing.Enabled
	if enabledChanged && !nameChanged && !hostChanged {
		action, state := "启用 Agent 令牌", "已启用"
		if !enabled {
			action, state = "停用 Agent 令牌", "已停用"
		}
		a.recordAudit(r, service.AuditModuleAgents, action, name, "Agent 令牌 "+name+" "+state, service.AuditResultSuccess)
	} else if changes := agentTokenChangeItems(*existing, domain.AgentToken{Name: name, Hostname: hostname, Enabled: enabled}); changes != "" {
		a.recordAudit(r, service.AuditModuleAgents, "更新 Agent 令牌", existing.Name, "Agent 令牌 "+existing.Name+" 已更新"+changes, service.AuditResultSuccess)
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tokens, _ := a.svc.Repo().ListAgentTokens()
	name := id
	for _, item := range tokens {
		if item.ID == id {
			name = item.Name
		}
	}
	ok, err := a.svc.Repo().DeleteAgentToken(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "令牌不存在")
		return
	}
	a.recordAudit(r, service.AuditModuleAgents, "删除 Agent 令牌", name, "Agent 令牌 "+name+" 已删除", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleNotifications(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	records, err := a.svc.Notifications(limit, strings.TrimSpace(r.URL.Query().Get("channel")))
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, records)
}

func (a *Router) handleChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := a.svc.Repo().ListNotifyChannels()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type channelView struct {
		domain.NotifyChannel
		Target string `json:"target"`
	}
	out := make([]channelView, 0, len(channels))
	for _, channel := range channels {
		out = append(out, channelView{NotifyChannel: channel, Target: service.ChannelTarget(channel)})
	}
	common.WriteJSON(w, http.StatusOK, out)
}

func decodeChannel(r *http.Request, current *domain.NotifyChannel) (domain.NotifyChannel, string) {
	var body domain.NotifyChannel
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return body, "请求体格式无效"
	}
	channel := body
	if current != nil {
		if channel.Name == "" {
			channel.Name = current.Name
		}
		if channel.Type == "" {
			channel.Type = current.Type
		}
		if channel.Level == "" {
			channel.Level = current.Level
		}
		if channel.TriggerGroup == "" {
			channel.TriggerGroup = current.TriggerGroup
		}
		if channel.Config == nil {
			channel.Config = current.Config
		}
		if len(channel.EventType) == 0 {
			channel.EventType = current.EventType
		}
	}
	channel.Name = strings.TrimSpace(channel.Name)
	if channel.Level == "" {
		channel.Level = "all"
	}
	if channel.TriggerGroup == "" {
		channel.TriggerGroup = "默认分组"
	}
	channel.Remark = strings.TrimSpace(channel.Remark)
	if channel.Config == nil {
		channel.Config = map[string]any{}
	}
	return channel, ""
}

func (a *Router) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	channel, errText := decodeChannel(r, nil)
	if errText != "" {
		common.WriteError(w, http.StatusBadRequest, errText)
		return
	}
	service.NormalizeChannelConfig(&channel)
	if errText := service.ValidateChannel(channel); errText != "" {
		common.WriteError(w, http.StatusBadRequest, errText)
		return
	}
	id, err := a.svc.Repo().CreateNotifyChannel(channel)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "创建通知失败")
		return
	}
	created, _ := a.svc.Repo().GetNotifyChannel(id)
	a.recordAudit(r, service.AuditModuleNotify, "创建通知渠道", channel.Name, notifyChannelAuditDetail(&channel, "已创建"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"channel": created, "target": service.ChannelTarget(channel)})
}

func (a *Router) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	current, err := a.svc.Repo().GetNotifyChannel(id)
	if err != nil || current == nil {
		common.WriteError(w, http.StatusNotFound, "通知不存在")
		return
	}
	previous := *current
	channel, errText := decodeChannel(r, current)
	if errText != "" {
		common.WriteError(w, http.StatusBadRequest, errText)
		return
	}
	channel.Type = current.Type
	service.NormalizeChannelConfig(&channel)
	if errText := service.ValidateChannel(channel); errText != "" {
		common.WriteError(w, http.StatusBadRequest, errText)
		return
	}
	if err := a.svc.Repo().UpdateNotifyChannel(id, channel); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "更新通知失败")
		return
	}
	changes := notifyChannelChangeItems(previous, channel)
	if changes == "：启用已修改" {
		action, state := "启用通知渠道", "已启用"
		if !channel.Enabled {
			action, state = "停用通知渠道", "已停用"
		}
		a.recordAudit(r, service.AuditModuleNotify, action, previous.Name, "通知渠道 "+previous.Name+" "+state, service.AuditResultSuccess)
	} else if changes != "" {
		a.recordAudit(r, service.AuditModuleNotify, "更新通知渠道", previous.Name, "通知渠道 "+previous.Name+" 已更新"+changes, service.AuditResultSuccess)
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"channel": channel, "target": service.ChannelTarget(channel)})
}

func (a *Router) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	current, _ := a.svc.Repo().GetNotifyChannel(id)
	ok, err := a.svc.Repo().DeleteNotifyChannel(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "通知不存在")
		return
	}
	name := id
	if current != nil {
		name = current.Name
	}
	a.recordAudit(r, service.AuditModuleNotify, "删除通知渠道", name, notifyChannelAuditDetail(current, "已删除"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleTestChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	channel, _ := a.svc.Repo().GetNotifyChannel(id)
	ok, message := a.svc.TestNotifyChannel(id)
	name := id
	if channel != nil {
		name = channel.Name
	}
	detail := notifyTestAuditDetail(channel, name, ok, message)
	a.recordAudit(r, service.AuditModuleNotify, "测试通知渠道", name, detail, map[bool]string{true: service.AuditResultSuccess, false: service.AuditResultFailure}[ok])
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": ok, "message": message})
}

func generateToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// validateManualToken 校验自定义令牌：16-128 位可见 ASCII，不含空格与控制字符
func validateManualToken(token string) error {
	if len(token) < 16 {
		return fmt.Errorf("自定义令牌至少 16 位")
	}
	if len(token) > 128 {
		return fmt.Errorf("自定义令牌最多 128 位")
	}
	for _, r := range token {
		if r < 0x21 || r > 0x7e {
			return fmt.Errorf("自定义令牌仅支持字母、数字与常见符号，不允许中文、空格或控制字符")
		}
	}
	return nil
}

// isUniqueTokenError 判断是否命中令牌唯一约束
func isUniqueTokenError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "agent_tokens.token")
}
