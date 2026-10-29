package auth

import (
	"context"
	"database/sql"

	"github.com/go-chi/chi/v5"

	"backup-monitor-backend/config"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
)

type Router struct {
	db             *sql.DB
	audit          *service.AuditService
	authWorkflow   *service.AuthWorkflow
	cfg            config.Config
	wecomExchange  func(ctx context.Context, provider *service.WecomProvider, code string) (string, error)
	wecomSSOVerify func(ctx context.Context, provider *service.WecomProvider, ticket string) (string, error)
}

func New(db *sql.DB, audit *service.AuditService, authWorkflow *service.AuthWorkflow, cfg config.Config) *Router {
	return &Router{db: db, audit: audit, authWorkflow: authWorkflow, cfg: cfg}
}

type SessionUser = domain.SessionUser
type authLoginRequest = domain.AuthLoginRequest
type changePasswordBody = domain.ChangePasswordBody

// RegisterPublic 挂载无需认证的认证入口与找回密码接口
func (a *Router) RegisterPublic(r chi.Router) {
	r.Post("/api/auth/login", a.handleLogin)
	r.Get("/api/auth/providers", a.handlePublicAuthProviders)
	r.Get("/api/auth/wecom/authorize", a.handleWecomAuthorize)
	r.Post("/api/auth/wecom/callback", a.handleWecomCallback)
	r.Post("/api/auth/wecom/sso/callback", a.handleWecomSSOCallback)
	r.Get("/api/auth/password-reset/captcha", a.handlePasswordResetCaptcha)
	r.Post("/api/auth/password-reset/verify", a.handlePasswordResetVerify)
	r.Post("/api/auth/password-reset/send", a.handlePasswordResetSend)
	r.Post("/api/auth/password-reset/confirm", a.handlePasswordResetConfirm)
}

// Register 挂载需认证的会话接口，修改密码与企微绑定对所有登录用户开放
func (a *Router) Register(r chi.Router) {
	r.Get("/api/auth/me", a.handleMe)
	r.Post("/api/auth/logout", a.handleLogout)
	r.Post("/api/auth/change-password", a.handleChangePassword)
	r.Get("/api/auth/wecom/bind-url", a.handleWecomBindURL)
	r.Post("/api/auth/wecom/bind", a.handleWecomBind)
	r.Post("/api/auth/wecom/sso/bind", a.handleWecomSSOBind)
	r.Delete("/api/auth/wecom/bind", a.handleWecomUnbind)
}
