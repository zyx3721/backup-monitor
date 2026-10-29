package settings

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

var sidebarPermissionCategories = []string{
	"概览", "备份主机", "备份任务", "执行记录", "告警中心", "通知管理", "Agent 管理", "审计日志", "系统配置",
}

func openPermissionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "permission.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	return db
}

func TestPermissionCatalogInvariants(t *testing.T) {
	if len(permissionCatalog) != 23 {
		t.Fatalf("permission count = %d, want 23 (keep docs/manual.md in sync)", len(permissionCatalog))
	}
	byKey := map[string]Permission{}
	for _, item := range permissionCatalog {
		if _, ok := byKey[item.Key]; ok {
			t.Fatalf("duplicate permission key %s", item.Key)
		}
		if item.Name == "" || item.Description == "" || item.Category == "" {
			t.Fatalf("permission %s missing name/description/category", item.Key)
		}
		byKey[item.Key] = item
	}
	for _, item := range permissionCatalog {
		if item.ImpliedReadPermission == "" {
			if !strings.HasSuffix(item.Key, ".read") {
				t.Fatalf("permission %s is neither read nor implies a read", item.Key)
			}
			continue
		}
		if _, ok := byKey[item.ImpliedReadPermission]; !ok {
			t.Fatalf("permission %s implies unknown key %s", item.Key, item.ImpliedReadPermission)
		}
		if want := strings.TrimSuffix(item.Key, ".manage") + ".read"; item.ImpliedReadPermission != want {
			t.Fatalf("permission %s implies %s, want %s", item.Key, item.ImpliedReadPermission, want)
		}
	}
	if _, ok := byKey["backup.settings.read"]; ok {
		t.Fatal("backup.settings.read should be merged into settings.base.read")
	}
	if _, ok := byKey["backup.settings.manage"]; ok {
		t.Fatal("backup.settings.manage should be merged into settings.base.manage")
	}
}

func TestPermissionCategoriesFollowSidebar(t *testing.T) {
	groups := map[string][]string{}
	for _, item := range permissionCatalog {
		groups[item.Category] = append(groups[item.Category], item.Key)
	}
	categories := make([]string, 0, len(groups))
	for category := range groups {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	want := append([]string(nil), sidebarPermissionCategories...)
	sort.Strings(want)
	if strings.Join(categories, ",") != strings.Join(want, ",") {
		t.Fatalf("permission categories = %v, want %v", categories, want)
	}
	for _, category := range sidebarPermissionCategories {
		if len(groups[category]) == 0 {
			t.Fatalf("permission category %s has no entries", category)
		}
	}
}

func TestPermissionCatalogEndpointGroupsBySidebarOrder(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/settings/permissions", nil)
	(&Router{}).handleSettingsPermissions(recorder, request)

	var payload struct {
		Items []Permission `json:"items"`
		Total int          `json:"total"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode permission payload: %v", err)
	}
	if payload.Total != len(permissionCatalog) || len(payload.Items) != len(permissionCatalog) {
		t.Fatalf("permission payload total = %d items = %d, want %d", payload.Total, len(payload.Items), len(permissionCatalog))
	}
	order := make([]string, 0, len(sidebarPermissionCategories))
	seen := map[string]struct{}{}
	for _, item := range payload.Items {
		if _, ok := seen[item.Category]; ok {
			continue
		}
		seen[item.Category] = struct{}{}
		order = append(order, item.Category)
	}
	if strings.Join(order, ",") != strings.Join(sidebarPermissionCategories, ",") {
		t.Fatalf("permission category order = %v, want %v", order, sidebarPermissionCategories)
	}
}

func TestBuiltinRolePermissions(t *testing.T) {
	operator := builtinRolePermissions("operator")
	for _, permission := range []string{
		"backup.hosts.manage", "backup.tasks.manage", "backup.executions.manage",
		"backup.alerts.manage", "backup.notifications.manage", "backup.agents.manage",
	} {
		if !containsString(operator, permission) {
			t.Fatalf("operator role missing %s", permission)
		}
	}
	for _, permission := range []string{
		"settings.base.manage", "settings.users.manage", "settings.auth.manage",
		"settings.notifications.manage", "audit.manage",
	} {
		if containsString(operator, permission) {
			t.Fatalf("operator role should not hold %s", permission)
		}
	}

	viewer := builtinRolePermissions("viewer")
	if !containsString(viewer, "backup.hosts.read") || !containsString(viewer, "backup.agents.read") {
		t.Fatalf("viewer role missing read permissions: %v", viewer)
	}
	for _, permission := range []string{"backup.hosts.manage", "backup.agents.manage", "backup.tasks.manage"} {
		if containsString(viewer, permission) {
			t.Fatalf("viewer role should not hold %s", permission)
		}
	}

	if admin := builtinRolePermissions("admin"); len(admin) != len(permissionCatalog) {
		t.Fatalf("admin role permissions = %d, want %d", len(admin), len(permissionCatalog))
	}
}

func TestEnsureResourceSchemaKeepsCatalogPermissions(t *testing.T) {
	db := openPermissionTestDB(t)
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if err := EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure schema second run: %v", err)
	}

	for key, want := range map[string][]string{
		"admin":    AllPermissionKeys(),
		"operator": builtinRolePermissions("operator"),
		"viewer":   builtinRolePermissions("viewer"),
	} {
		var raw string
		if err := db.QueryRow(`SELECT permissions FROM roles WHERE key=?`, key).Scan(&raw); err != nil {
			t.Fatalf("query role %s: %v", key, err)
		}
		var permissions []string
		if err := json.Unmarshal([]byte(raw), &permissions); err != nil {
			t.Fatalf("decode role %s permissions: %v", key, err)
		}
		sort.Strings(permissions)
		expected := append([]string(nil), want...)
		sort.Strings(expected)
		if strings.Join(permissions, ",") != strings.Join(expected, ",") {
			t.Fatalf("role %s permissions = %v, want %v", key, permissions, expected)
		}
	}
}
