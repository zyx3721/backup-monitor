package auth

import (
	"backup-monitor-backend/router/common"
	"backup-monitor-backend/router/settings"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"strings"

	"backup-monitor-backend/internal/service"
	"github.com/golang-jwt/jwt/v5"
)

func (a *Router) handlePublicAuthProviders(w http.ResponseWriter, r *http.Request) {
	items := make([]map[string]any, 0, 2)
	for _, provider := range []struct {
		id, fallback string
	}{
		{id: "ldap", fallback: "AD/LDAP"},
		{id: service.WecomProviderID, fallback: "企业微信"},
	} {
		var name string
		var enabled int64
		err := a.db.QueryRowContext(r.Context(), "SELECT title,enabled FROM auth_config WHERE name=?", provider.id).Scan(&name, &enabled)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if strings.TrimSpace(name) == "" {
			name = provider.fallback
		}
		items = append(items, map[string]any{"id": provider.id, "type": provider.id, "name": name, "enabled": enabled != 0})
	}
	var passwordResetEnabled int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT enabled FROM pwd_recover_config WHERE name='email'").Scan(&passwordResetEnabled); err != nil && !errors.Is(err, sql.ErrNoRows) {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items), "passwordResetEnabled": passwordResetEnabled != 0})
}

func (a *Router) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := a.ensureLoginAllowed(r.Context(), req.Username); err != nil {
		var locked loginLockedError
		if !errors.As(err, &locked) {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.recordAuditEvent(r.Context(), req.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", loginAuditTarget(req.Username), loginFailureDetail(req.Mode, req.Username, err), service.AuditResultFailure)
		common.WriteError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	response, err := a.authWorkflow.Login(r.Context(), req)
	if err != nil {
		a.recordAuditEvent(r.Context(), req.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", loginAuditTarget(req.Username), loginFailureDetail(req.Mode, req.Username, err), service.AuditResultFailure)
		switch {
		case err.Error() == "username is required" || err.Error() == "password is required":
			common.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			if recordErr := a.recordLoginFailure(r.Context(), req.Username); recordErr != nil {
				common.WriteError(w, http.StatusInternalServerError, recordErr.Error())
				return
			}
			common.WriteError(w, http.StatusUnauthorized, "invalid username or password")
		case errors.Is(err, service.ErrUserNotProvisioned):
			common.WriteError(w, http.StatusUnauthorized, "用户未在平台中启用")
		default:
			common.WriteError(w, http.StatusUnauthorized, err.Error())
		}
		return
	}
	if _, err := a.clearLoginFailures(r.Context(), response.User.Username); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	userResponse, err := a.authUserResponse(r.Context(), response.User)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := settings.RecordSettingsUserLogin(r.Context(), a.db, response.User.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAuditEvent(r.Context(), response.User.Username, common.ClientIP(r), service.AuditModuleAuth, "用户登录", response.User.Username, authLoginDetail(response.User.Source, response.User.Username, "登录"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"token": response.Token, "user": userResponse})
}

// loginAuditTarget 登录失败时目标账号可能为空，统一占位展示
func loginAuditTarget(username string) string {
	if strings.TrimSpace(username) == "" {
		return "-"
	}
	return username
}

// loginFailureDetail 将登录失败原因归为可读的审计详情，按登录方式区分来源前缀
func loginFailureDetail(mode, username string, err error) string {
	source := authSourceLabel(mode)
	reason := ""
	switch {
	case err.Error() == "username is required" || err.Error() == "password is required":
		reason = "用户名或密码不能为空"
	case errors.Is(err, service.ErrInvalidCredentials):
		reason = "用户名或密码错误"
	case errors.Is(err, service.ErrUserNotProvisioned):
		reason = "用户未在平台中启用"
	default:
		reason = err.Error()
	}
	return strings.TrimSpace(source+" "+strings.TrimSpace(username)) + " 登录失败：" + reason
}

// authSourceLabel 登录方式来源标签：本地账号为“本地用户”，企业微信扫码为“使用企业微信认证的系统用户”，其余按 AD/LDAP 展示
func authSourceLabel(source string) string {
	switch {
	case strings.EqualFold(strings.TrimSpace(source), "ldap"):
		return "LDAP 用户"
	case strings.EqualFold(strings.TrimSpace(source), "wecom"):
		return "使用企业微信认证的系统用户"
	default:
		return "本地用户"
	}
}

// wecomBoundFlag 查询用户是否已绑定企业微信账号
func (a *Router) wecomBoundFlag(ctx context.Context, userID int64) (bool, error) {
	var bound int64
	if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE wecom_userid IS NOT NULL AND id=?", userID).Scan(&bound); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return bound > 0, nil
}

func (a *Router) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	response, err := a.authWorkflow.Me(r.Context(), user)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	directRoles, effectiveRoles, permissions, accessErr := settings.UserAccess(a.db, user.ID, user.Username)
	if accessErr != nil {
		common.WriteError(w, http.StatusInternalServerError, accessErr.Error())
		return
	}
	role := "viewer"
	if strings.EqualFold(user.Username, "admin") {
		role = "admin"
	} else if len(effectiveRoles) > 0 {
		role, _ = effectiveRoles[0]["key"].(string)
	}
	response["role"] = role
	response["source"] = user.Source
	if response["source"] == "" {
		response["source"] = "local"
	}
	response["permissions"] = permissions
	response["effectiveUserRoles"] = effectiveRoles
	response["directRoles"] = directRoles
	wecomBound, err := a.wecomBoundFlag(r.Context(), user.ID)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response["wecomBound"] = wecomBound
	common.WriteJSON(w, http.StatusOK, response)
}

func (a *Router) authUserResponse(ctx context.Context, user SessionUser) (map[string]any, error) {
	response := map[string]any{"id": user.ID, "username": user.Username}
	if a.authWorkflow != nil {
		base, err := a.authWorkflow.Me(ctx, user)
		if err != nil {
			return nil, err
		}
		for key, value := range base {
			response[key] = value
		}
	}
	directRoles, effectiveRoles, permissions, err := settings.UserAccess(a.db, user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	role := "viewer"
	if strings.EqualFold(user.Username, "admin") {
		role = "admin"
	} else if len(effectiveRoles) > 0 {
		role, _ = effectiveRoles[0]["key"].(string)
	}
	response["role"] = role
	response["source"] = user.Source
	if response["source"] == "" {
		response["source"] = "local"
	}
	response["permissions"] = permissions
	response["effectiveUserRoles"] = effectiveRoles
	response["directRoles"] = directRoles
	wecomBound, err := a.wecomBoundFlag(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	response["wecomBound"] = wecomBound
	return response, nil
}

// authLoginDetail 认证审计详情：按登录方式区分来源，如“本地用户 admin 登录成功”
func authLoginDetail(source, username, action string) string {
	return strings.TrimSpace(authSourceLabel(source)+" "+strings.TrimSpace(username)) + " " + action + "成功"
}

func (a *Router) handleLogout(w http.ResponseWriter, r *http.Request) {
	if jti := sessionJTI(r, a.cfg.JWTSecret); jti != "" {
		_, _ = a.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE jti=?", jti)
	}
	if user, err := common.CurrentUser(r.Context()); err == nil && strings.TrimSpace(user.Username) != "" {
		a.recordAuditEvent(r.Context(), user.Username, common.ClientIP(r), service.AuditModuleAuth, "用户注销", user.Username, authLoginDetail(user.Source, user.Username, "注销"), service.AuditResultSuccess)
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// sessionJTI 从请求头解析当前令牌的会话标识，令牌无效时返回空串
func sessionJTI(r *http.Request, secret string) string {
	parts := strings.SplitN(strings.TrimSpace(r.Header.Get("Authorization")), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(strings.TrimSpace(parts[1]), claims, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return ""
	}
	return claims.ID
}
