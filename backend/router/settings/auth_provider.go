package settings

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/security"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func (a *Router) handleSettingsAuthProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var enabled int64
		var name, rawConfig string
		err := a.db.QueryRowContext(r.Context(), "SELECT enabled,title,COALESCE(config,'{}') FROM auth_config WHERE name='ldap'").Scan(&enabled, &name, &rawConfig)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			common.WriteError(w, 500, err.Error())
			return
		}
		if strings.TrimSpace(name) == "" {
			name = "AD/LDAP"
		}
		stored := decodeSettingsMap(rawConfig)
		config := cloneMap(stored)
		hasPassword := strings.TrimSpace(stringConfig(stored, "bindPassword")) != ""
		delete(config, "bindPassword")
		config["hasBindPassword"] = hasPassword
		common.WriteJSON(w, 200, map[string]any{"id": "ldap", "type": "ldap", "name": name, "enabled": enabled == 1, "config": config, "updatedAt": a.ldapProviderUpdatedAt(r.Context())})
		return
	}
	if r.Method == http.MethodPost {
		operator, _ := common.CurrentUser(r.Context())
		providerName := a.ldapProviderDisplayName(r.Context())
		cfg, err := loadLDAPSettings(a.db, a.cfg.JWTSecret)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		conn, err := dialAndBindLDAP(cfg)
		if err != nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, ldapUserMessage(err), service.AuditResultFailure)
			common.WriteError(w, http.StatusServiceUnavailable, ldapUserMessage(err))
			return
		}
		matchedUsers, countErr := countLDAPUsers(conn, cfg)
		_ = conn.Close()
		if countErr != nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, ldapUserMessage(countErr), service.AuditResultFailure)
			common.WriteError(w, http.StatusServiceUnavailable, ldapUserMessage(countErr))
			return
		}
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, fmt.Sprintf("AD/LDAP 认证配置测试成功，匹配用户数：%d", matchedUsers), service.AuditResultSuccess)
		common.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "matchedUsers": matchedUsers})
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
	delete(body.Config, "hasBindPassword")
	host := stringConfig(body.Config, "host")
	baseDN := stringConfig(body.Config, "baseDN")
	bindDN := stringConfig(body.Config, "bindDN")
	filter := stringConfig(body.Config, "userFilter")
	var port int
	switch value := body.Config["port"].(type) {
	case float64:
		port = int(value)
	case string:
		port, _ = strconv.Atoi(value)
	}
	var existingEnabled int64
	var existingName, existingRaw string
	if err := a.db.QueryRowContext(r.Context(), "SELECT enabled,title,COALESCE(config,'{}') FROM auth_config WHERE name='ldap'").Scan(&existingEnabled, &existingName, &existingRaw); err != nil && !errors.Is(err, sql.ErrNoRows) {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existing := decodeSettingsMap(existingRaw)
	existingPassword := stringConfig(existing, "bindPassword")
	password := stringConfig(body.Config, "bindPassword")
	if body.ClearConfig {
		password = ""
	} else if password == "" {
		password = existingPassword
	}
	if password != "" && !strings.HasPrefix(password, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(password)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		password = encrypted
	}
	if body.Enabled {
		if host == "" {
			common.WriteError(w, 400, "LDAP 服务器地址不能为空")
			return
		}
		if baseDN == "" {
			common.WriteError(w, 400, "Base DN 不能为空")
			return
		}
		if filter == "" {
			common.WriteError(w, 400, "用户过滤器不能为空")
			return
		}
		if bindDN == "" {
			common.WriteError(w, 400, "绑定 DN 不能为空")
			return
		}
		if password == "" {
			common.WriteError(w, 400, "绑定密码不能为空")
			return
		}
		if settingsBool(body.Config, "useTLS") && settingsBool(body.Config, "startTLS") {
			common.WriteError(w, 400, "LDAPS 与 StartTLS 不能同时启用")
			return
		}
		if port <= 0 {
			common.WriteError(w, 400, "端口不能为空")
			return
		}
	}
	providerConfig := cloneMap(body.Config)
	delete(providerConfig, "bindPassword")
	if password != "" {
		providerConfig["bindPassword"] = password
	}
	if _, err := a.db.ExecContext(r.Context(), "UPDATE auth_config SET enabled=?,title=?,config=?,updated_at=? WHERE name='ldap'", boolInt(body.Enabled), name, mustJSON(providerConfig), appcommon.NowTimestamp()); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	authChanged := boolInt(body.Enabled) != existingEnabled ||
		name != existingName ||
		host != stringConfig(existing, "host") ||
		baseDN != stringConfig(existing, "baseDN") ||
		bindDN != stringConfig(existing, "bindDN") ||
		password != existingPassword ||
		filter != stringConfig(existing, "userFilter") ||
		stringConfig(body.Config, "groupFilter") != stringConfig(existing, "groupFilter") ||
		!reflect.DeepEqual(providerConfig, existing)
	if authChanged {
		if operator, operatorErr := common.CurrentUser(r.Context()); operatorErr == nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "更新认证配置", existingName, existingName+" 认证配置已更新", service.AuditResultSuccess)
		}
	}
	responseConfig := cloneMap(body.Config)
	delete(responseConfig, "bindPassword")
	responseConfig["hasBindPassword"] = password != ""
	common.WriteJSON(w, 200, map[string]any{"id": "ldap", "type": "ldap", "name": name, "enabled": body.Enabled, "config": responseConfig, "updatedAt": a.ldapProviderUpdatedAt(r.Context())})
}

func (a *Router) ldapProviderUpdatedAt(ctx context.Context) string {
	var updatedAt string
	if err := a.db.QueryRowContext(ctx, "SELECT updated_at FROM auth_config WHERE name='ldap'").Scan(&updatedAt); err != nil {
		return ""
	}
	return formatTimestamp(updatedAt)
}

func (a *Router) ldapProviderDisplayName(ctx context.Context) string {
	var name string
	if err := a.db.QueryRowContext(ctx, "SELECT COALESCE(NULLIF(title,''),'AD/LDAP') FROM auth_config WHERE name='ldap'").Scan(&name); err != nil {
		return "AD/LDAP"
	}
	return name
}
