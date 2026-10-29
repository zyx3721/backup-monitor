package settings

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"encoding/json"
	"net"
	"net/http"
	"reflect"
	"strings"

	"backup-monitor-backend/internal/security"
)

func (a *Router) handleSettingsEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		const name = "邮件通知"
		var raw string
		var enabled int64
		var updatedAt string
		if err := a.db.QueryRowContext(r.Context(), "SELECT enabled,config,updated_at FROM pwd_recover_config WHERE name='email'").Scan(&enabled, &raw, &updatedAt); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		config := decodeSettingsMap(raw)
		sanitizeEmailConfig(config)
		common.WriteJSON(w, 200, map[string]any{"id": "email", "name": name, "enabled": enabled == 1, "passwordResetEnabled": enabled == 1, "config": config, "updatedAt": formatTimestamp(updatedAt)})
		return
	}
	if r.Method == http.MethodPost {
		operator, _ := common.CurrentUser(r.Context())
		var body struct{ To string }
		if !common.DecodeJSON(w, r, &body) {
			return
		}
		var raw string
		if err := a.db.QueryRowContext(r.Context(), "SELECT config FROM pwd_recover_config WHERE name='email'").Scan(&raw); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		config := decodeSettingsMap(raw)
		to := strings.TrimSpace(body.To)
		if err := SendSettingsTestEmail(config, to); err != nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试找密配置", "找回密码邮件", err.Error(), service.AuditResultFailure)
			common.WriteError(w, http.StatusBadGateway, err.Error())
			return
		}
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试找密配置", "找回密码邮件", "找回密码邮件配置测试成功，测试收件人："+to, service.AuditResultSuccess)
		common.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "to": to})
		return
	}
	var body settingsEmailBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	if body.Config == nil {
		body.Config = map[string]any{}
	}
	var existingRaw string
	var existingEnabled int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT enabled,config FROM pwd_recover_config WHERE name='email'").Scan(&existingEnabled, &existingRaw); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existingConfig := decodeSettingsMap(existingRaw)
	if body.ClearConfig {
		body.Config = map[string]any{}
	} else if stringConfig(body.Config, "password") == "" {
		if existingPassword := stringConfig(existingConfig, "password"); existingPassword != "" {
			body.Config["password"] = existingPassword
		}
	}
	delete(body.Config, "hasPassword")
	if password := stringConfig(body.Config, "password"); password != "" && !strings.HasPrefix(password, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(password)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		body.Config["password"] = encrypted
	}
	host, port := stringConfig(body.Config, "smtpHost"), stringConfig(body.Config, "smtpPort")
	if body.PasswordResetEnabled {
		for _, field := range []struct{ key, label string }{
			{"smtpHost", "SMTP 主机"},
			{"smtpPort", "SMTP 端口"},
			{"username", "用户名"},
		} {
			if stringConfig(body.Config, field.key) == "" {
				common.WriteError(w, http.StatusBadRequest, field.label+"不能为空")
				return
			}
		}
		if stringConfig(body.Config, "password") == "" && stringConfig(existingConfig, "password") == "" {
			common.WriteError(w, http.StatusBadRequest, "密码不能为空")
			return
		}
		if stringConfig(body.Config, "from") == "" {
			common.WriteError(w, http.StatusBadRequest, "发件人不能为空")
			return
		}
	}
	if host != "" || port != "" {
		if port == "" {
			port = "25"
		}
		if _, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port)); err != nil {
			common.WriteError(w, 400, "SMTP 主机地址无效")
			return
		}
	}
	if _, err := a.db.ExecContext(r.Context(), "UPDATE pwd_recover_config SET enabled=?,config=?,updated_at=? WHERE name='email'", boolInt(body.PasswordResetEnabled), mustJSON(body.Config), appcommon.NowTimestamp()); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	changed := boolInt(body.PasswordResetEnabled) != existingEnabled ||
		!reflect.DeepEqual(body.Config, existingConfig)
	if changed {
		if actor, actorErr := common.CurrentUser(r.Context()); actorErr == nil {
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新找密配置", "找回密码邮件", "找回密码邮件配置已更新", service.AuditResultSuccess)
		}
	}
	responseConfig := cloneMap(body.Config)
	sanitizeEmailConfig(responseConfig)
	common.WriteJSON(w, 200, map[string]any{"id": "email", "name": "邮件通知", "enabled": mapBool(body.Config, "enabled"), "passwordResetEnabled": body.PasswordResetEnabled, "config": responseConfig, "updatedAt": formatTimestamp(appcommon.NowTimestamp())})
}

func decodeSettingsMap(raw string) map[string]any {
	config := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil || config == nil {
		return map[string]any{}
	}
	return config
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func sanitizeEmailConfig(config map[string]any) {
	if stringConfig(config, "password") != "" {
		delete(config, "password")
		config["hasPassword"] = true
	}
}

type settingsEmailBody struct {
	PasswordResetEnabled bool
	ClearConfig          bool
	Config               map[string]any
}
