package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/security"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

// handleSettingsWecomProvider 企业微信认证配置读取与保存：Secret 加密存储且不回显
func (a *Router) handleSettingsWecomProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		provider, err := loadWecomProviderRow(r.Context(), a.db)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		responseConfig := map[string]any{
			"authMode":        provider.authMode,
			"corpid":          provider.corpID,
			"agentid":         provider.agentID,
			"redirectPrefix":  provider.redirectPrefix,
			"hasSecret":       provider.secret != "",
			"ssoBaseUrl":      provider.ssoBaseURL,
			"ssoAppID":        provider.ssoAppID,
			"hasSsoAppSecret": provider.ssoAppSecret != "",
		}
		common.WriteJSON(w, http.StatusOK, map[string]any{
			"id": service.WecomProviderID, "type": service.WecomProviderID,
			"name": provider.name, "enabled": provider.enabled,
			"config": responseConfig, "updatedAt": formatTimestamp(provider.updatedAt),
		})
		return
	}
	var body struct {
		Name        string
		Enabled     bool
		ClearConfig bool
		Config      map[string]any
	}
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		common.WriteError(w, 400, "显示名称不能为空")
		return
	}
	if body.Config == nil {
		body.Config = map[string]any{}
	}
	for key, value := range body.Config {
		if text, ok := value.(string); ok {
			body.Config[key] = strings.TrimSpace(text)
		}
	}
	corpID := stringConfig(body.Config, "corpid")
	agentID := stringConfig(body.Config, "agentid")
	redirectPrefix := strings.TrimRight(stringConfig(body.Config, "redirectPrefix"), "/")
	secret := stringConfig(body.Config, "secret")
	ssoBaseURL := strings.TrimRight(stringConfig(body.Config, "ssoBaseUrl"), "/")
	ssoAppID := stringConfig(body.Config, "ssoAppID")
	ssoAppSecret := stringConfig(body.Config, "ssoAppSecret")

	existing, err := loadWecomProviderRow(r.Context(), a.db)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	authMode := stringConfig(body.Config, "authMode")
	if authMode != service.WecomAuthModeDirect && authMode != service.WecomAuthModeSSO {
		authMode = existing.authMode
	}
	if authMode != service.WecomAuthModeSSO {
		authMode = service.WecomAuthModeDirect
	}
	if body.ClearConfig {
		secret = ""
		ssoAppSecret = ""
	} else {
		if secret == "" {
			secret = existing.secret
		}
		if ssoAppSecret == "" {
			ssoAppSecret = existing.ssoAppSecret
		}
	}
	if secret != "" && !strings.HasPrefix(secret, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(secret)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		secret = encrypted
	}
	if ssoAppSecret != "" && !strings.HasPrefix(ssoAppSecret, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(ssoAppSecret)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		ssoAppSecret = encrypted
	}
	if body.Enabled {
		if authMode == service.WecomAuthModeSSO {
			if ssoBaseURL == "" {
				common.WriteError(w, 400, "认证中心地址不能为空")
				return
			}
			if ssoAppID == "" {
				common.WriteError(w, 400, "应用标识不能为空")
				return
			}
			if ssoAppSecret == "" {
				common.WriteError(w, 400, "应用密钥不能为空")
				return
			}
		} else {
			if corpID == "" {
				common.WriteError(w, 400, "企业 ID（corpid）不能为空")
				return
			}
			if agentID == "" {
				common.WriteError(w, 400, "应用 AgentID 不能为空")
				return
			}
			if secret == "" {
				common.WriteError(w, 400, "应用 Secret 不能为空")
				return
			}
		}
	}
	storedConfig := map[string]any{
		"authMode":       authMode,
		"corpid":         corpID,
		"agentid":        agentID,
		"redirectPrefix": redirectPrefix,
		"ssoBaseUrl":     ssoBaseURL,
		"ssoAppID":       ssoAppID,
	}
	if secret != "" {
		storedConfig["secret"] = secret
	}
	if ssoAppSecret != "" {
		storedConfig["ssoAppSecret"] = ssoAppSecret
	}
	rawConfig, err := json.Marshal(storedConfig)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := a.db.ExecContext(r.Context(),
		"UPDATE auth_config SET enabled=?,title=?,config=?,updated_at=? WHERE name=?",
		boolInt(body.Enabled), name, string(rawConfig), appcommon.NowTimestamp(), service.WecomProviderID,
	); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	changed := boolInt(body.Enabled) != boolInt(existing.enabled) ||
		name != existing.name ||
		secret != existing.secret ||
		ssoAppSecret != existing.ssoAppSecret ||
		!reflect.DeepEqual(storedConfigWithoutSecret(storedConfig), storedConfigWithoutSecret(existing.configMap))
	if changed {
		if operator, operatorErr := common.CurrentUser(r.Context()); operatorErr == nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "更新认证配置", name, name+" 认证配置已更新", service.AuditResultSuccess)
		}
	}
	responseConfig := map[string]any{
		"authMode":        authMode,
		"corpid":          corpID,
		"agentid":         agentID,
		"redirectPrefix":  redirectPrefix,
		"hasSecret":       secret != "",
		"ssoBaseUrl":      ssoBaseURL,
		"ssoAppID":        ssoAppID,
		"hasSsoAppSecret": ssoAppSecret != "",
	}
	common.WriteJSON(w, 200, map[string]any{
		"id": service.WecomProviderID, "type": service.WecomProviderID,
		"name": name, "enabled": body.Enabled,
		"config": responseConfig, "updatedAt": formatTimestamp(appcommon.NowTimestamp()),
	})
}

// wecomProviderRow auth_config 中企业微信配置的现状快照
type wecomProviderRow struct {
	name           string
	enabled        bool
	authMode       string
	corpID         string
	agentID        string
	redirectPrefix string
	secret         string
	ssoBaseURL     string
	ssoAppID       string
	ssoAppSecret   string
	updatedAt      string
	configMap      map[string]any
}

// loadWecomProviderRow 读取企业微信配置行，缺行时返回零值快照
func loadWecomProviderRow(ctx context.Context, db *sql.DB) (wecomProviderRow, error) {
	row := wecomProviderRow{configMap: map[string]any{}, authMode: service.WecomAuthModeDirect}
	var name, rawConfig string
	var enabled int64
	var updatedAt string
	err := db.QueryRowContext(ctx,
		"SELECT title,enabled,COALESCE(config,''),updated_at FROM auth_config WHERE name=?",
		service.WecomProviderID,
	).Scan(&name, &enabled, &rawConfig, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return row, nil
	}
	if err != nil {
		return row, err
	}
	row.name = name
	row.enabled = enabled != 0
	row.updatedAt = updatedAt
	_ = json.Unmarshal([]byte(rawConfig), &row.configMap)
	row.authMode = wecomRowString(row.configMap, "authMode")
	if row.authMode != service.WecomAuthModeSSO {
		row.authMode = service.WecomAuthModeDirect
	}
	row.corpID = wecomRowString(row.configMap, "corpid")
	row.agentID = wecomRowString(row.configMap, "agentid")
	row.redirectPrefix = wecomRowString(row.configMap, "redirectPrefix")
	row.secret = wecomRowString(row.configMap, "secret")
	row.ssoBaseURL = strings.TrimRight(wecomRowString(row.configMap, "ssoBaseUrl"), "/")
	row.ssoAppID = wecomRowString(row.configMap, "ssoAppID")
	row.ssoAppSecret = wecomRowString(row.configMap, "ssoAppSecret")
	return row, nil
}

func wecomRowString(stored map[string]any, key string) string {
	value, _ := stored[key].(string)
	return strings.TrimSpace(value)
}

// storedConfigWithoutSecret 复制配置并剔除加密密钥，用于变更比对与响应
func storedConfigWithoutSecret(source map[string]any) map[string]any {
	clone := map[string]any{}
	for key, value := range source {
		if key == "secret" || key == "ssoAppSecret" {
			continue
		}
		clone[key] = value
	}
	return clone
}
