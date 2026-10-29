package settings

import (
	"net/http"

	"backup-monitor-backend/router/common"
)

// Permission 角色权限目录条目，Implied 字段用于角色编辑勾选联动与服务端闭包补全
type Permission struct {
	Key                   string   `json:"key"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Category              string   `json:"category"`
	ImpliedReadPermission string   `json:"impliedReadPermission,omitempty"`
	ImpliedPermissions    []string `json:"impliedPermissions,omitempty"`
}

// permissionCatalog 权限目录，Category 与侧边栏导航同名，角色编辑按集合分组展示
var permissionCatalog = []Permission{
	{Key: "backup.dashboard.read", Name: "查看概览", Description: "查看备份监控概览与趋势", Category: "概览"},
	{Key: "backup.hosts.read", Name: "查看主机", Description: "查看备份主机与主机详情", Category: "备份主机"},
	{Key: "backup.hosts.manage", Name: "管理主机", Description: "修改主机角色、删除与导出主机", Category: "备份主机", ImpliedReadPermission: "backup.hosts.read"},
	{Key: "backup.tasks.read", Name: "查看任务", Description: "查看备份任务与执行计划", Category: "备份任务"},
	{Key: "backup.tasks.manage", Name: "管理任务", Description: "修改执行计划、删除与导出任务", Category: "备份任务", ImpliedReadPermission: "backup.tasks.read"},
	{Key: "backup.executions.read", Name: "查看执行记录", Description: "查看备份执行记录与详情", Category: "执行记录"},
	{Key: "backup.executions.manage", Name: "管理执行记录", Description: "删除与导出执行记录", Category: "执行记录", ImpliedReadPermission: "backup.executions.read"},
	{Key: "backup.alerts.read", Name: "查看告警", Description: "查看告警事件", Category: "告警中心"},
	{Key: "backup.alerts.manage", Name: "管理告警", Description: "恢复、删除与导出告警事件", Category: "告警中心", ImpliedReadPermission: "backup.alerts.read"},
	{Key: "backup.notifications.read", Name: "查看通知", Description: "查看通知渠道与订阅配置", Category: "通知管理"},
	{Key: "backup.notifications.manage", Name: "管理通知", Description: "维护并测试通知渠道", Category: "通知管理", ImpliedReadPermission: "backup.notifications.read"},
	{Key: "backup.agents.read", Name: "查看 Agent 配置", Description: "查看采集配置、Agent 令牌与二进制", Category: "Agent 管理"},
	{Key: "backup.agents.manage", Name: "管理 Agent 配置", Description: "维护采集配置、令牌并推送更新", Category: "Agent 管理", ImpliedReadPermission: "backup.agents.read"},

	{Key: "audit.read", Name: "查看审计", Description: "查看操作审计日志", Category: "审计日志"},
	{Key: "audit.manage", Name: "管理审计", Description: "导出、清空操作审计日志", Category: "审计日志", ImpliedReadPermission: "audit.read"},

	{Key: "settings.base.read", Name: "查看基础配置", Description: "查看品牌标识、登录展示、安全时效与备份参数", Category: "系统配置"},
	{Key: "settings.base.manage", Name: "管理基础配置", Description: "维护品牌标识、登录展示、安全时效与备份参数", Category: "系统配置", ImpliedReadPermission: "settings.base.read"},
	{Key: "settings.users.read", Name: "查看用户配置", Description: "查看用户、用户群组和角色", Category: "系统配置"},
	{Key: "settings.users.manage", Name: "管理用户配置", Description: "维护用户、用户群组和角色", Category: "系统配置", ImpliedReadPermission: "settings.users.read"},
	{Key: "settings.auth.read", Name: "查看认证配置", Description: "查看 AD/LDAP 与企业微信认证配置", Category: "系统配置"},
	{Key: "settings.auth.manage", Name: "管理认证配置", Description: "维护并测试 AD/LDAP 与企业微信认证配置", Category: "系统配置", ImpliedReadPermission: "settings.auth.read"},
	{Key: "settings.notifications.read", Name: "查看找密配置", Description: "查看找回密码邮件媒介", Category: "系统配置"},
	{Key: "settings.notifications.manage", Name: "管理找密配置", Description: "维护并测试找回密码邮件媒介", Category: "系统配置", ImpliedReadPermission: "settings.notifications.read"},
}

func (a *Router) handleSettingsPermissions(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, map[string]any{"items": permissionCatalog, "total": len(permissionCatalog)})
}
