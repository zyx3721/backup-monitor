package auth

// 企业微信统一认证中心（SSO）模式处理器：校验认证中心回跳的 ticket 后复用直连模式的登录与绑定链路

import (
	"context"
	"errors"
	"net/http"

	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

// handleWecomSSOCallback 接收统一认证中心回跳 ticket：校验后按绑定关系登录并签发会话
func (a *Router) handleWecomSSOCallback(w http.ResponseWriter, r *http.Request) {
	var req wecomSSORequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, _ := common.CurrentUser(r.Context())
	provider, err := a.wecomSSOEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.verifyWecomSSOTicket(r.Context(), provider, req.Ticket)
	if err != nil {
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", loginAuditTarget(operator.Username), "使用企业微信方式登录失败："+err.Error(), service.AuditResultFailure)
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	a.completeWecomLogin(w, r, userid)
}

// handleWecomSSOBind 用统一认证中心 ticket 为当前登录用户绑定企业微信账号
func (a *Router) handleWecomSSOBind(w http.ResponseWriter, r *http.Request) {
	var req wecomSSORequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	provider, err := a.wecomSSOEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.verifyWecomSSOTicket(r.Context(), provider, req.Ticket)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	a.completeWecomBind(w, r, operator, userid)
}

// wecomSSORequest 统一认证中心回调请求体
type wecomSSORequest struct {
	Ticket string `json:"ticket"`
}

// wecomSSOEnabledProvider 读取启用且处于统一认证模式、凭据完整的企业微信配置
func (a *Router) wecomSSOEnabledProvider(ctx context.Context) (*service.WecomProvider, error) {
	provider, err := service.LoadWecomProvider(ctx, a.db)
	if err != nil {
		return nil, errors.New("读取企业微信认证配置失败")
	}
	if provider == nil || !provider.Enabled || provider.AuthMode != service.WecomAuthModeSSO || !provider.HasWecomCredential() {
		return nil, errors.New("企业微信统一认证未启用或配置不完整")
	}
	return provider, nil
}

// verifyWecomSSOTicket 调用认证中心校验 ticket 并换取成员 userid
func (a *Router) verifyWecomSSOTicket(ctx context.Context, provider *service.WecomProvider, ticket string) (string, error) {
	if a.wecomSSOVerify != nil {
		return a.wecomSSOVerify(ctx, provider, ticket)
	}
	return provider.WecomSSOVerifyTicket(ctx, ticket)
}
