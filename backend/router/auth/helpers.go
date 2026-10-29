package auth

import (
	"context"

	"backup-monitor-backend/internal/service"
)

func (a *Router) auditService() *service.AuditService {
	if a.audit == nil {
		a.audit = service.NewAuditService(a.db)
	}
	return a.audit
}

// recordAuditEvent 写入一条与 SQL 无关的显式审计事件（登录、连接测试、配置变更等）
func (a *Router) recordAuditEvent(ctx context.Context, username, ip, module, action, target, detail, result string) {
	event := service.AuditEvent{Module: module, Action: action, Target: target, Detail: detail, Result: result}
	_ = a.auditService().RecordEvent(ctx, username, ip, event)
}
