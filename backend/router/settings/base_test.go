package settings

import (
	"testing"

	"backup-monitor-backend/internal/domain"
)

func TestBaseSectionChangeItems(t *testing.T) {
	base := domain.DefaultSystemBaseConfig()

	brand := base
	brand.SiteName = "备份平台"
	brand.IconData = "/logo.svg"
	if got, want := baseSectionChangeItems(base, brand, domain.BaseSectionBrand), "：网站名称、站点图标已修改"; got != want {
		t.Errorf("brand changes = %q, want %q", got, want)
	}

	security := base
	security.LoginMaxFailures = 9
	security.WecomStateTTLMinutes = 8
	if got, want := baseSectionChangeItems(base, security, domain.BaseSectionSecurity), "：企业微信扫码有效期、登录失败锁定次数已修改"; got != want {
		t.Errorf("security changes = %q, want %q", got, want)
	}

	if got := baseSectionChangeItems(base, base, domain.BaseSectionBrand); got != "" {
		t.Errorf("unchanged brand = %q, want empty", got)
	}
	if got := baseSectionChangeItems(base, security, domain.BaseSectionBrand); got != "" {
		t.Errorf("security change reported under brand = %q, want empty", got)
	}
}

func TestBaseSectionChangeItemsWithoutSection(t *testing.T) {
	base := domain.DefaultSystemBaseConfig()
	next := base
	next.SiteName = "备份平台"
	next.LoginLockoutMinutes = 7
	if got, want := baseSectionChangeItems(base, next, ""), "：网站名称、登录锁定等待时长已修改"; got != want {
		t.Errorf("all sections changes = %q, want %q", got, want)
	}
}
