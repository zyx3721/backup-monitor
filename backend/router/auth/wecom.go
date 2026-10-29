package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"backup-monitor-backend/router/settings"
)

// handleWecomAuthorize 签发企业微信扫码登录跳转地址与内嵌二维码参数（公开接口，登录页使用）
func (a *Router) handleWecomAuthorize(w http.ResponseWriter, r *http.Request) {
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := a.wecomAuthorizePayload(r, provider, service.WecomLoginPurpose, 0)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, payload)
}

// handleWecomBindURL 签发企业微信绑定扫码跳转地址（需登录，state 绑定当前用户）
func (a *Router) handleWecomBindURL(w http.ResponseWriter, r *http.Request) {
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := a.wecomAuthorizePayload(r, provider, service.WecomBindPurpose, operator.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]string{"url": payload.URL})
}

// handleWecomCallback 接收企业微信扫码回调：校验 state 后按绑定关系登录并签发会话
func (a *Router) handleWecomCallback(w http.ResponseWriter, r *http.Request) {
	var req wecomCallbackRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, _ := common.CurrentUser(r.Context())
	if _, err := service.VerifyWecomState(a.cfg.JWTSecret, req.State, service.WecomLoginPurpose); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.exchangeWecomCode(r.Context(), provider, req.Code)
	if err != nil {
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", loginAuditTarget(operator.Username), "使用企业微信方式登录失败："+err.Error(), service.AuditResultFailure)
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	a.completeWecomLogin(w, r, userid)
}

// completeWecomLogin 按绑定关系登录、记录审计并写回会话响应，返回 false 表示已写错误响应
func (a *Router) completeWecomLogin(w http.ResponseWriter, r *http.Request, userid string) bool {
	response, err := a.authWorkflow.LoginByWecom(r.Context(), userid)
	if err != nil {
		a.recordAuditEvent(r.Context(), "-", common.ClientIP(r), service.AuditModuleAuth, "用户登录", userid, "企业微信账号 "+userid+" 使用企业微信认证方式登录失败："+wecomLoginFailureMessage(err), service.AuditResultFailure)
		if errors.Is(err, service.ErrUserNotProvisioned) {
			common.WriteError(w, http.StatusUnauthorized, "用户未在平台中启用")
			return false
		}
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return false
	}
	userResponse, err := a.authUserResponse(r.Context(), response.User)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if err := settings.RecordSettingsUserLogin(r.Context(), a.db, response.User.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	a.recordAuditEvent(r.Context(), response.User.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", response.User.Username, authLoginDetail(response.User.Source, response.User.Username, "登录"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"token": response.Token, "user": userResponse})
	return true
}

// handleWecomBind 完成当前登录用户与企业微信账号的绑定
func (a *Router) handleWecomBind(w http.ResponseWriter, r *http.Request) {
	var req wecomCallbackRequest
	if !common.DecodeJSON(w, r, &req) {
		return
	}
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	state, err := service.VerifyWecomState(a.cfg.JWTSecret, req.State, service.WecomBindPurpose)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if state.UserID != operator.ID {
		common.WriteError(w, http.StatusForbidden, "绑定状态与当前用户不符，请重新发起绑定")
		return
	}
	provider, err := a.wecomEnabledProvider(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	userid, err := a.exchangeWecomCode(r.Context(), provider, req.Code)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	a.completeWecomBind(w, r, operator, userid)
}

// completeWecomBind 将当前登录用户与企微成员建立绑定并记录审计，返回 false 表示已写错误响应
func (a *Router) completeWecomBind(w http.ResponseWriter, r *http.Request, operator domain.SessionUser, userid string) bool {
	var boundUser int64
	bindErr := a.db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE wecom_userid=?", userid).Scan(&boundUser)
	if bindErr == nil && boundUser != operator.ID {
		common.WriteError(w, http.StatusConflict, "该企业微信账号已绑定其他用户")
		return false
	}
	if _, err := a.db.ExecContext(r.Context(),
		"UPDATE users SET wecom_userid=?,updated_at="+appcommon.SQLTimestampNow+" WHERE id=?",
		userid, operator.ID,
	); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "绑定企业微信", operator.Username, "系统用户 "+operator.Username+" 绑定企业微信账号 "+userid+" 成功", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "wecomUserid": userid})
	return true
}

// handleWecomUnbind 解除当前登录用户的企业微信绑定
func (a *Router) handleWecomUnbind(w http.ResponseWriter, r *http.Request) {
	operator, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var wecomUserid string
	_ = a.db.QueryRowContext(r.Context(), "SELECT wecom_userid FROM users WHERE id=?", operator.ID).Scan(&wecomUserid)
	if _, err := a.db.ExecContext(r.Context(), "UPDATE users SET wecom_userid=NULL,updated_at="+appcommon.SQLTimestampNow+" WHERE id=?", operator.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(wecomUserid) == "" {
		wecomUserid = "-"
	}
	a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleAuth, "解绑企业微信", operator.Username, "系统用户 "+operator.Username+" 已解除绑定企业微信账号 "+wecomUserid, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// wecomCallbackRequest 扫码回调与绑定共用的请求体
type wecomCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// wecomEnabledProvider 读取启用且凭据完整的企业微信配置，未就绪时返回错误提示
func (a *Router) wecomEnabledProvider(ctx context.Context) (*service.WecomProvider, error) {
	provider, err := service.LoadWecomProvider(ctx, a.db)
	if err != nil {
		return nil, errors.New("读取企业微信认证配置失败")
	}
	if provider == nil || !provider.Enabled || !provider.HasWecomCredential() {
		return nil, errors.New("企业微信认证未启用或配置不完整")
	}
	return provider, nil
}

// wecomAuthorizeEmbed 内嵌二维码登录参数：iframe 地址与回跳路径（直连为内嵌中转路由，SSO 为登录路由），直连模式附带签名 state
type wecomAuthorizeEmbed struct {
	AuthMode     string `json:"auth_mode"`
	IframeURL    string `json:"iframe_url"`
	State        string `json:"state,omitempty"`
	CallbackPath string `json:"callback_path"`
}

// wecomAuthorizeResponse 登录跳转地址与内嵌二维码参数，embed 为空表示仅支持整页跳转
type wecomAuthorizeResponse struct {
	URL   string               `json:"url"`
	Embed *wecomAuthorizeEmbed `json:"embed,omitempty"`
}

// wecomAuthorizePayload 按认证方式构造登录跳转地址与内嵌二维码参数：SSO 复用认证中心入口地址，直连签发 state 并区分整页与内嵌回跳
func (a *Router) wecomAuthorizePayload(r *http.Request, provider *service.WecomProvider, purpose string, userID int64) (wecomAuthorizeResponse, error) {
	if provider.AuthMode == service.WecomAuthModeSSO {
		target := provider.WecomSSOLoginURL()
		return wecomAuthorizeResponse{URL: target, Embed: &wecomAuthorizeEmbed{AuthMode: service.WecomAuthModeSSO, IframeURL: target, CallbackPath: service.WecomLoginCallbackPath}}, nil
	}
	ttlMinutes := settings.LoadSystemBaseConfig(r.Context(), a.db).WecomStateTTLMinutes
	if ttlMinutes < 1 {
		ttlMinutes = service.WecomDefaultStateTTL
	}
	state, err := service.SignWecomState(a.cfg.JWTSecret, service.WecomState{Purpose: purpose, UserID: userID, Exp: time.Now().Add(time.Duration(ttlMinutes) * time.Minute).Unix()})
	if err != nil {
		return wecomAuthorizeResponse{}, err
	}
	baseURL := service.WecomRequestBaseURL(r)
	return wecomAuthorizeResponse{
		URL:   provider.WecomAuthorizeURL(baseURL, state),
		Embed: &wecomAuthorizeEmbed{AuthMode: service.WecomAuthModeDirect, IframeURL: provider.WecomEmbedAuthorizeURL(baseURL, state), State: state, CallbackPath: service.WecomEmbedCallbackPath},
	}, nil
}

// exchangeWecomCode 授权码换取企业微信成员 userid，测试可替换实现
func (a *Router) exchangeWecomCode(ctx context.Context, provider *service.WecomProvider, code string) (string, error) {
	if a.wecomExchange != nil {
		return a.wecomExchange(ctx, provider, code)
	}
	return provider.WecomExchangeCode(ctx, code)
}

// wecomLoginFailureMessage 将企微登录失败归类为可读提示
func wecomLoginFailureMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrWecomNotBound):
		return service.ErrWecomNotBound.Error()
	case errors.Is(err, service.ErrUserNotProvisioned):
		return "用户未在平台中启用"
	default:
		return err.Error()
	}
}
