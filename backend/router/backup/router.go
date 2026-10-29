// Package backup 提供备份监控的 HTTP 接口：Agent 公开通道与管理端接口。
package backup

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backup-monitor-backend/config"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

const (
	permDashboardRead  = "backup.dashboard.read"
	permHostsRead      = "backup.hosts.read"
	permHostsManage    = "backup.hosts.manage"
	permTasksRead      = "backup.tasks.read"
	permTasksManage    = "backup.tasks.manage"
	permExecutionsRead = "backup.executions.read"
	permExecutionsMng  = "backup.executions.manage"
	permAlertsRead     = "backup.alerts.read"
	permAlertsManage   = "backup.alerts.manage"
	permNotifyRead     = "backup.notifications.read"
	permNotifyManage   = "backup.notifications.manage"
	permAgentsRead     = "backup.agents.read"
	permAgentsManage   = "backup.agents.manage"
	permBaseConfigRead = "settings.base.read"
	permBaseConfigMng  = "settings.base.manage"
)

// Router 备份监控 HTTP 路由。
type Router struct {
	svc   *service.BackupService
	hub   *service.AgentHub
	audit *service.AuditService
	cfg   config.Config
}

// New 创建备份监控路由。
func New(svc *service.BackupService, hub *service.AgentHub, audit *service.AuditService, cfg config.Config) *Router {
	return &Router{svc: svc, hub: hub, audit: audit, cfg: cfg}
}

// RegisterPublic 挂载 Agent 通道与公开下载（Agent 接口的令牌校验在处理器内完成）。
func (a *Router) RegisterPublic(r chi.Router) {
	r.Post("/api/ingest/report", a.handleIngest)
	r.Get("/api/agent/config", a.handleAgentConfig)
	r.Get("/api/agent/download", a.handleAgentDownload)
	r.Get("/api/agent/binary/latest", a.handleAgentBinaryLatest)
	r.Get("/api/agent/ws", a.hub.Handler())
	r.Get("/download/backup-agent", a.handlePublicAgentDownload)
}

// Register 挂载管理端接口。
func (a *Router) Register(r chi.Router, requirePermission func(string) func(http.Handler) http.Handler) {
	r.With(requirePermission(permDashboardRead)).Get("/api/backup/dashboard", a.handleDashboard)

	r.With(requirePermission(permHostsRead)).Get("/api/backup/hosts", a.handleHosts)
	r.With(requirePermission(permHostsRead)).Get("/api/backup/hosts/refresh-plan", a.handleHostRefreshPlan)
	r.With(requirePermission(permHostsRead)).Get("/api/backup/hosts/{id}", a.handleHost)
	r.With(requirePermission(permHostsManage)).Put("/api/backup/hosts/{id}/role", a.handleHostRole)
	r.With(requirePermission(permHostsManage)).Delete("/api/backup/hosts/{id}", a.handleDeleteHost)

	r.With(requirePermission(permTasksRead)).Get("/api/backup/tasks", a.handleTasks)
	r.With(requirePermission(permTasksRead)).Get("/api/backup/tasks/{id}", a.handleTask)
	r.With(requirePermission(permTasksRead)).Get("/api/backup/tasks/{id}/executions", a.handleTaskExecutions)
	r.With(requirePermission(permTasksManage)).Delete("/api/backup/tasks/{id}", a.handleDeleteTask)
	r.With(requirePermission(permTasksManage)).Put("/api/backup/tasks/{id}/schedule", a.handleTaskSchedule)

	r.With(requirePermission(permExecutionsRead)).Get("/api/backup/executions", a.handleExecutions)
	r.With(requirePermission(permExecutionsRead)).Get("/api/backup/executions/{id}", a.handleExecution)
	r.With(requirePermission(permExecutionsMng)).Delete("/api/backup/executions/{id}", a.handleDeleteExecution)

	r.With(requirePermission(permAlertsRead)).Get("/api/backup/alerts", a.handleAlerts)
	r.With(requirePermission(permAlertsManage)).Post("/api/backup/alerts/{id}/resolve", a.handleResolveAlert)
	r.With(requirePermission(permAlertsManage)).Delete("/api/backup/alerts/{id}", a.handleDeleteAlert)

	r.With(requirePermission(permNotifyRead)).Get("/api/backup/notify/channels", a.handleChannels)
	r.With(requirePermission(permNotifyManage)).Post("/api/backup/notify/channels", a.handleCreateChannel)
	r.With(requirePermission(permNotifyManage)).Put("/api/backup/notify/channels/{id}", a.handleUpdateChannel)
	r.With(requirePermission(permNotifyManage)).Delete("/api/backup/notify/channels/{id}", a.handleDeleteChannel)
	r.With(requirePermission(permNotifyManage)).Post("/api/backup/notify/channels/{id}/test", a.handleTestChannel)
	r.With(requirePermission(permNotifyRead)).Get("/api/backup/notifications", a.handleNotifications)

	r.With(requirePermission(permBaseConfigRead)).Get("/api/backup/settings", a.handleSettings)
	r.With(requirePermission(permBaseConfigMng)).Put("/api/backup/settings", a.handleSaveSettings)

	r.With(requirePermission(permAgentsRead)).Get("/api/backup/profiles", a.handleProfiles)
	r.With(requirePermission(permAgentsManage)).Post("/api/backup/profiles", a.handleCreateProfile)
	r.With(requirePermission(permAgentsManage)).Put("/api/backup/profiles/{id}", a.handleUpdateProfile)
	r.With(requirePermission(permAgentsManage)).Delete("/api/backup/profiles/{id}", a.handleDeleteProfile)

	r.With(requirePermission(permAgentsRead)).Get("/api/backup/tokens", a.handleTokens)
	r.With(requirePermission(permAgentsManage)).Post("/api/backup/tokens", a.handleCreateToken)
	r.With(requirePermission(permAgentsManage)).Put("/api/backup/tokens/{id}", a.handleUpdateToken)
	r.With(requirePermission(permAgentsManage)).Delete("/api/backup/tokens/{id}", a.handleDeleteToken)

	r.With(requirePermission(permAgentsRead)).Get("/api/backup/agent-binaries", a.handleAgentBinaries)
	r.With(requirePermission(permAgentsManage)).Post("/api/backup/agent-binary", a.handleUploadAgentBinary)
	r.With(requirePermission(permAgentsManage)).Put("/api/backup/agent-update/push", a.handlePushUpdate)
}

func (a *Router) recordAudit(r *http.Request, module, action, target, detail, result string) {
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		return
	}
	_ = a.audit.RecordEvent(r.Context(), user.Username, common.ClientIP(r), service.AuditEvent{
		Module: module, Action: action, Target: target, Detail: detail, Result: result,
	})
}

func (a *Router) authorizeAgent(r *http.Request) bool {
	settings := a.svc.Settings()
	token := strings.TrimSpace(r.Header.Get("X-Agent-Token"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token != "" {
		if found, err := a.svc.Repo().FindAgentToken(token); err == nil && found != nil {
			_ = a.svc.Repo().TouchAgentToken(found.ID, timeNow())
			return true
		}
	}
	return !settings.RequireAgentToken
}
