package service

import (
	"context"
	"database/sql"
	"strings"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/repository"
)

// AuditService 负责执行业务写入并记录结构化审计日志：除保留原始 SQL 外，
// 同时归类出模块、操作、目标与结果，供审计日志页面展示与筛选。
type AuditService struct {
	db repository.Executor
}

func NewAuditService(db repository.Executor) *AuditService {
	return &AuditService{db: db}
}

// auditExecWriter 兼容 *sql.DB 与 *sql.Tx 的最小写入接口
type auditExecWriter interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// AuditEvent 审计事件结构：显式事件记录模块、操作、目标、详情与结果，供审计日志页面展示与筛选
type AuditEvent struct {
	Module string
	Action string
	Target string
	Detail string
	Result string
}

const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"

	AuditModuleAuth       = "auth"
	AuditModuleHosts      = "hosts"
	AuditModuleTasks      = "tasks"
	AuditModuleExecutions = "executions"
	AuditModuleAlerts     = "alerts"
	AuditModuleAudit      = "audit"
	AuditModuleNotify     = "notify"
	AuditModuleAgents     = "agents"
	AuditModuleSettings   = "settings"
)

// RecordEvent 记录一条显式审计事件（如登录、连接测试、配置变更）
func (s *AuditService) RecordEvent(ctx context.Context, username, ip string, event AuditEvent) error {
	return s.recordEvent(ctx, s.db, username, ip, event)
}

func (s *AuditService) recordEvent(ctx context.Context, writer auditExecWriter, username, ip string, event AuditEvent) error {
	if strings.TrimSpace(event.Result) == "" {
		event.Result = AuditResultSuccess
	}
	if strings.TrimSpace(event.Target) == "" {
		event.Target = "-"
	}
	return s.insertHistory(ctx, writer, username, ip, event)
}

func (s *AuditService) insertHistory(ctx context.Context, writer auditExecWriter, username, ip string, event AuditEvent) error {
	if strings.TrimSpace(username) == "" {
		username = "unknown"
	}
	_, err := writer.ExecContext(ctx,
		`INSERT INTO audit_logs (date, sql, authuser, ip, module, action, target, detail, result) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		appcommon.NowTimestamp(), "", username, ip, event.Module, event.Action, event.Target, event.Detail, event.Result,
	)
	if err != nil {
		return err
	}
	return nil
}
