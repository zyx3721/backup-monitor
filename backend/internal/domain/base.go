package domain

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

// SystemBaseConfig 系统基础配置：品牌标识与安全时效。
type SystemBaseConfig struct {
	SiteName                         string  `json:"siteName" example:"Backup Monitor"`
	LoginName                        string  `json:"loginName" example:"Backup Monitor"`
	AppName                          string  `json:"appName" example:"Backup Monitor"`
	AppSubtitle                      string  `json:"appSubtitle" example:"Backup Control Plane"`
	IconData                         string  `json:"iconData" example:"/favicon.svg"`
	ResetCodeTTLMinutes              int     `json:"resetCodeTtlMinutes" example:"10"`
	ResetCaptchaTTLMinutes           int     `json:"resetCaptchaTtlMinutes" example:"1"`
	PasswordResetSendCooldownMinutes float64 `json:"passwordResetSendCooldownMinutes" example:"0.5"`
	PasswordResetRateLimitMinutes    int     `json:"passwordResetRateLimitMinutes" example:"5"`
	WecomStateTTLMinutes             int     `json:"wecomStateTtlMinutes" example:"5"`
	LoginMaxFailures                 int     `json:"loginMaxFailures" example:"5"`
	LoginLockoutMinutes              int     `json:"loginLockoutMinutes" example:"2"`
	Section                          string  `json:"section,omitempty" example:"brand"`
}

// 基础配置的五个区块标识：brand 品牌标识、security 安全时效、thresholds 判定与保留、agent Agent 通道、alarm 安全与告警
const (
	BaseSectionBrand      = "brand"
	BaseSectionSecurity   = "security"
	BaseSectionThresholds = "thresholds"
	BaseSectionAgent      = "agent"
	BaseSectionAlarm      = "alarm"
)

// DefaultSystemBaseConfig 返回系统基础配置默认值。
func DefaultSystemBaseConfig() SystemBaseConfig {
	return SystemBaseConfig{
		SiteName:                         "Backup Monitor",
		LoginName:                        "Backup Monitor",
		AppName:                          "Backup Monitor",
		AppSubtitle:                      "Backup Control Plane",
		IconData:                         "/favicon.svg",
		ResetCodeTTLMinutes:              10,
		ResetCaptchaTTLMinutes:           1,
		PasswordResetSendCooldownMinutes: 0.5,
		PasswordResetRateLimitMinutes:    5,
		WecomStateTTLMinutes:             5,
		LoginMaxFailures:                 5,
		LoginLockoutMinutes:              2,
	}
}

// NormalizeSystemBaseConfig 为空值与非法数值回填默认值。
func NormalizeSystemBaseConfig(config SystemBaseConfig) SystemBaseConfig {
	def := DefaultSystemBaseConfig()
	config.SiteName = baseText(config.SiteName, def.SiteName)
	config.LoginName = baseText(config.LoginName, def.LoginName)
	config.AppName = baseText(config.AppName, def.AppName)
	config.AppSubtitle = baseText(config.AppSubtitle, def.AppSubtitle)
	config.IconData = baseText(config.IconData, def.IconData)
	config.ResetCodeTTLMinutes = baseInt(config.ResetCodeTTLMinutes, def.ResetCodeTTLMinutes)
	config.ResetCaptchaTTLMinutes = baseInt(config.ResetCaptchaTTLMinutes, def.ResetCaptchaTTLMinutes)
	if !(config.PasswordResetSendCooldownMinutes > 0) {
		config.PasswordResetSendCooldownMinutes = def.PasswordResetSendCooldownMinutes
	}
	config.PasswordResetRateLimitMinutes = baseInt(config.PasswordResetRateLimitMinutes, def.PasswordResetRateLimitMinutes)
	config.WecomStateTTLMinutes = baseInt(config.WecomStateTTLMinutes, def.WecomStateTTLMinutes)
	config.LoginMaxFailures = baseInt(config.LoginMaxFailures, def.LoginMaxFailures)
	config.LoginLockoutMinutes = baseInt(config.LoginLockoutMinutes, def.LoginLockoutMinutes)
	return config
}

// ValidateSystemBaseConfig 校验系统基础配置取值范围。
func ValidateSystemBaseConfig(config SystemBaseConfig) error {
	for _, item := range []struct {
		label string
		value string
	}{
		{"网站名称", config.SiteName},
		{"认证页品牌名称", config.LoginName},
		{"控制台品牌名称", config.AppName},
		{"控制台品牌副标题", config.AppSubtitle},
	} {
		if utf8.RuneCountInString(item.value) > 60 {
			return errors.New(item.label + "长度不能超过 60 个字符")
		}
	}
	if !strings.HasPrefix(config.IconData, "/") && !strings.HasPrefix(config.IconData, "data:image/") {
		return errors.New("图标格式无效，须为站点路径或图片 Data URL")
	}
	if config.ResetCodeTTLMinutes < 1 || config.ResetCodeTTLMinutes > 60 {
		return errors.New("找回密码验证码有效期范围为 1-60 分钟")
	}
	if config.ResetCaptchaTTLMinutes < 1 || config.ResetCaptchaTTLMinutes > 10 {
		return errors.New("图形验证码有效期范围为 1-10 分钟")
	}
	if config.PasswordResetSendCooldownMinutes < 0.5 || config.PasswordResetSendCooldownMinutes > 10 || !isHalfMinuteStep(config.PasswordResetSendCooldownMinutes) {
		return errors.New("发送冷却时间范围为 0.5-10 分钟，且步长为 0.5 分钟")
	}
	if config.PasswordResetRateLimitMinutes < 5 || config.PasswordResetRateLimitMinutes > 10 {
		return errors.New("频率限制统计窗口范围为 5-10 分钟")
	}
	if config.WecomStateTTLMinutes < 1 || config.WecomStateTTLMinutes > 60 {
		return errors.New("企业微信扫码有效期范围为 1-60 分钟")
	}
	if config.LoginMaxFailures < 3 || config.LoginMaxFailures > 10 {
		return errors.New("登录失败锁定次数范围为 3-10 次")
	}
	if config.LoginLockoutMinutes < 1 || config.LoginLockoutMinutes > 10 {
		return errors.New("登录锁定等待时长范围为 1-10 分钟")
	}
	return nil
}

func isHalfMinuteStep(value float64) bool {
	return math.Abs(value*2-math.Round(value*2)) < 1e-6
}

func baseText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// Sections 按基础配置区块拆分：brand 品牌标识五字段、security 安全时效七字段
func (c SystemBaseConfig) Sections() map[string]map[string]any {
	return map[string]map[string]any{
		BaseSectionBrand: {
			"siteName":    c.SiteName,
			"loginName":   c.LoginName,
			"appName":     c.AppName,
			"appSubtitle": c.AppSubtitle,
			"iconData":    c.IconData,
		},
		BaseSectionSecurity: {
			"resetCodeTtlMinutes":              c.ResetCodeTTLMinutes,
			"resetCaptchaTtlMinutes":           c.ResetCaptchaTTLMinutes,
			"passwordResetSendCooldownMinutes": c.PasswordResetSendCooldownMinutes,
			"passwordResetRateLimitMinutes":    c.PasswordResetRateLimitMinutes,
			"wecomStateTtlMinutes":             c.WecomStateTTLMinutes,
			"loginMaxFailures":                 c.LoginMaxFailures,
			"loginLockoutMinutes":              c.LoginLockoutMinutes,
		},
	}
}

// MergeSystemBaseConfigSection 按区块白名单用 patch 覆盖 existing 的对应字段，
// 其余字段保留 existing 原值，避免整包提交相互覆盖；section 非法时第二个返回值为 false。
func MergeSystemBaseConfigSection(existing, patch SystemBaseConfig, section string) (SystemBaseConfig, bool) {
	switch section {
	case BaseSectionBrand:
		existing.SiteName = patch.SiteName
		existing.LoginName = patch.LoginName
		existing.AppName = patch.AppName
		existing.AppSubtitle = patch.AppSubtitle
		existing.IconData = patch.IconData
	case BaseSectionSecurity:
		existing.ResetCodeTTLMinutes = patch.ResetCodeTTLMinutes
		existing.ResetCaptchaTTLMinutes = patch.ResetCaptchaTTLMinutes
		existing.PasswordResetSendCooldownMinutes = patch.PasswordResetSendCooldownMinutes
		existing.PasswordResetRateLimitMinutes = patch.PasswordResetRateLimitMinutes
		existing.WecomStateTTLMinutes = patch.WecomStateTTLMinutes
		existing.LoginMaxFailures = patch.LoginMaxFailures
		existing.LoginLockoutMinutes = patch.LoginLockoutMinutes
	default:
		return existing, false
	}
	return existing, true
}

func baseInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
