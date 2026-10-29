package settings

import (
	"backup-monitor-backend/router/common"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
)

type publicBrandingResponse struct {
	SiteName    string `json:"siteName"`
	LoginName   string `json:"loginName"`
	AppName     string `json:"appName"`
	AppSubtitle string `json:"appSubtitle"`
	IconData    string `json:"iconData"`
}

func (a *Router) handleSettingsBase(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		common.WriteJSON(w, http.StatusOK, LoadSystemBaseConfig(r.Context(), a.db))
		return
	}
	var body domain.SystemBaseConfig
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	existing := LoadSystemBaseConfig(r.Context(), a.db)
	merged := body
	if body.Section != "" {
		result, ok := domain.MergeSystemBaseConfigSection(existing, body, body.Section)
		if !ok {
			common.WriteError(w, http.StatusBadRequest, "无效的基础配置区块")
			return
		}
		merged = result
	}
	config := domain.NormalizeSystemBaseConfig(merged)
	if err := domain.ValidateSystemBaseConfig(config); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	changed := config != existing
	if err := writeSystemBaseConfig(r.Context(), a.db, config, body.Section); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if changed {
		if actor, actorErr := common.CurrentUser(r.Context()); actorErr == nil {
			target, detail := baseSectionAuditText(body.Section)
			detail += baseSectionChangeItems(existing, config, body.Section)
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新基础配置", target, detail, service.AuditResultSuccess)
		}
	}
	common.WriteJSON(w, http.StatusOK, config)
}

// baseSectionChangeItems 生成基础配置变更项后缀，如“：网站名称、站点图标已修改”；无字段变化时返回空串
func baseSectionChangeItems(before, after domain.SystemBaseConfig, section string) string {
	changes := make([]string, 0, 12)
	if section == "" || section == domain.BaseSectionBrand {
		if before.SiteName != after.SiteName {
			changes = append(changes, "网站名称")
		}
		if before.LoginName != after.LoginName {
			changes = append(changes, "认证页品牌名称")
		}
		if before.AppName != after.AppName {
			changes = append(changes, "控制台品牌名称")
		}
		if before.AppSubtitle != after.AppSubtitle {
			changes = append(changes, "控制台品牌副标题")
		}
		if before.IconData != after.IconData {
			changes = append(changes, "站点图标")
		}
	}
	if section == "" || section == domain.BaseSectionSecurity {
		if before.ResetCodeTTLMinutes != after.ResetCodeTTLMinutes {
			changes = append(changes, "找回密码验证码有效期")
		}
		if before.ResetCaptchaTTLMinutes != after.ResetCaptchaTTLMinutes {
			changes = append(changes, "图形验证码有效期")
		}
		if before.PasswordResetSendCooldownMinutes != after.PasswordResetSendCooldownMinutes {
			changes = append(changes, "发送冷却时间")
		}
		if before.PasswordResetRateLimitMinutes != after.PasswordResetRateLimitMinutes {
			changes = append(changes, "频率限制统计窗口")
		}
		if before.WecomStateTTLMinutes != after.WecomStateTTLMinutes {
			changes = append(changes, "企业微信扫码有效期")
		}
		if before.LoginMaxFailures != after.LoginMaxFailures {
			changes = append(changes, "登录失败锁定次数")
		}
		if before.LoginLockoutMinutes != after.LoginLockoutMinutes {
			changes = append(changes, "登录锁定等待时长")
		}
	}
	if len(changes) == 0 {
		return ""
	}
	return "：" + mergeChangeItems(changes)
}

// baseSectionAuditText 基础配置审计按保存区块细分目标与详情
func baseSectionAuditText(section string) (string, string) {
	switch section {
	case domain.BaseSectionBrand:
		return "品牌标识", "品牌标识配置已更新"
	case domain.BaseSectionSecurity:
		return "安全时效", "安全时效配置已更新"
	}
	return "基础配置", "基础配置已更新"
}

// handlePublicBase 返回免认证的品牌标识，供登录页、找回密码页与启动屏动态展示。
func (a *Router) handlePublicBase(w http.ResponseWriter, r *http.Request) {
	config := LoadSystemBaseConfig(r.Context(), a.db)
	common.WriteJSON(w, http.StatusOK, publicBrandingResponse{
		SiteName:    config.SiteName,
		LoginName:   config.LoginName,
		AppName:     config.AppName,
		AppSubtitle: config.AppSubtitle,
		IconData:    config.IconData,
	})
}

// LoadSystemBaseConfig 读取品牌标识与安全时效：basic_config 按区块分行存储，此处合并 brand、security 两行；
// 备份监控的 thresholds、agent、alarm 三行由仓储层自行读写，缺行或解析失败时回退默认值
func LoadSystemBaseConfig(ctx context.Context, db *sql.DB) domain.SystemBaseConfig {
	merged := map[string]any{}
	for _, name := range []string{domain.BaseSectionBrand, domain.BaseSectionSecurity} {
		var raw string
		if err := db.QueryRowContext(ctx, "SELECT config FROM basic_config WHERE name=?", name).Scan(&raw); err != nil {
			continue
		}
		section := map[string]any{}
		if err := json.Unmarshal([]byte(raw), &section); err != nil {
			continue
		}
		for key, value := range section {
			merged[key] = value
		}
	}
	if len(merged) == 0 {
		return domain.DefaultSystemBaseConfig()
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return domain.DefaultSystemBaseConfig()
	}
	var config domain.SystemBaseConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return domain.DefaultSystemBaseConfig()
	}
	return domain.NormalizeSystemBaseConfig(config)
}

// writeSystemBaseConfig 把整份基础配置按子模块拆分写回 basic_config 对应行；
// 指定 section 时只更新该行，另一行保留原值
func writeSystemBaseConfig(ctx context.Context, db *sql.DB, config domain.SystemBaseConfig, section string) error {
	sections := config.Sections()
	names := []string{domain.BaseSectionBrand, domain.BaseSectionSecurity}
	if section == domain.BaseSectionBrand || section == domain.BaseSectionSecurity {
		names = []string{section}
	}
	for _, name := range names {
		raw, err := json.Marshal(sections[name])
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "UPDATE basic_config SET config=?,updated_at=? WHERE name=?", string(raw), appcommon.NowTimestamp(), name); err != nil {
			return err
		}
	}
	return nil
}
