package system

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backup-monitor-backend/config"
	"backup-monitor-backend/internal/service"
)

type Router struct {
	db    *sql.DB
	audit *service.AuditService
	cfg   config.Config
}

func New(db *sql.DB, audit *service.AuditService, cfg config.Config) *Router {
	return &Router{db: db, audit: audit, cfg: cfg}
}

// Register 挂载系统域路由：审计日志按查看权限，清空按管理权限
func (a *Router) Register(r chi.Router, requirePermission func(string) func(http.Handler) http.Handler) {
	r.With(requirePermission("audit.read")).Get("/api/history", a.handleHistory)
	r.Post("/api/history/export", a.handleHistoryExport)
	r.With(requirePermission("audit.manage")).Post("/api/history/clear", a.handleHistoryClear)
}
