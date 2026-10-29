package settings

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

func settingsRoleMap(id int64, key, name, description, raw string, builtin, disabled int64, createdAt, updatedAt string) map[string]any {
	permissions := []string{}
	if err := json.Unmarshal([]byte(raw), &permissions); err != nil || permissions == nil {
		permissions = []string{}
	}
	return map[string]any{
		"id":          strconv.FormatInt(id, 10),
		"key":         key,
		"name":        name,
		"description": description,
		"permissions": normalizePermissions(permissions),
		"builtin":     builtin != 0,
		"disabled":    disabled != 0,
		"createdAt":   formatTimestamp(createdAt),
		"updatedAt":   formatTimestamp(updatedAt),
	}
}

func UserAccess(db *sql.DB, userID int64, username string) (directRoles, effectiveRoles []map[string]any, permissions []string, err error) {
	if strings.EqualFold(username, "admin") {
		role := settingsRoleMap(1, "admin", "admin", "系统全部操作权限", mustJSON(permissionKeys(true)), 1, 0, "", "")
		return []map[string]any{role}, []map[string]any{role}, permissionKeys(true), nil
	}

	directRoles, err = querySettingsRoles(db, `
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       r.disabled,r.created_at,r.updated_at
		FROM user_roles ur
		JOIN roles r ON r.id=ur.role_id
		WHERE ur.user_id=? ORDER BY r.name`, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	effectiveRoles, err = querySettingsRoles(db, `
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       r.disabled,r.created_at,r.updated_at
		FROM user_roles ur
		JOIN roles r ON r.id=ur.role_id
		WHERE ur.user_id=? AND r.disabled=0
		UNION
		SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin,
		       r.disabled,r.created_at,r.updated_at
		FROM user_group_members gm
		JOIN user_groups g ON g.id=gm.group_id AND g.disabled=0
		JOIN user_group_roles gr ON gr.group_id=g.id
		JOIN roles r ON r.id=gr.role_id
		WHERE gm.user_id=? AND r.disabled=0
		ORDER BY name`, userID, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	permissionSet := map[string]struct{}{}
	for _, role := range effectiveRoles {
		for _, permission := range role["permissions"].([]string) {
			permissionSet[permission] = struct{}{}
		}
	}
	permissions = make([]string, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	return directRoles, effectiveRoles, normalizePermissions(permissions), nil
}

func querySettingsRoles(db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]map[string]any, 0)
	for rows.Next() {
		var id, builtin, disabled int64
		var createdAt, updatedAt string
		var key, name, description, raw string
		if err := rows.Scan(&id, &key, &name, &description, &raw, &builtin, &disabled, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, settingsRoleMap(id, key, name, description, raw, builtin, disabled, createdAt, updatedAt))
	}
	return roles, rows.Err()
}

func HasPermission(db *sql.DB, user SessionUser, permission string) bool {
	if strings.EqualFold(user.Username, "admin") {
		return true
	}
	_, _, permissions, err := UserAccess(db, user.ID, user.Username)
	if err != nil {
		return false
	}
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func normalizePermissions(keys []string) []string {
	catalog := map[string]Permission{}
	for _, item := range permissionCatalog {
		catalog[item.Key] = item
	}
	expanded := map[string]bool{}
	queue := make([]string, 0, len(keys))
	for _, key := range uniqueStrings(keys) {
		if _, ok := catalog[key]; ok {
			queue = append(queue, key)
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if expanded[key] {
			continue
		}
		expanded[key] = true
		item := catalog[key]
		if item.ImpliedReadPermission != "" {
			queue = append(queue, item.ImpliedReadPermission)
		}
		queue = append(queue, item.ImpliedPermissions...)
	}
	result := make([]string, 0, len(expanded))
	for key := range expanded {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// isReadPermission 判断是否为只读类权限，只读类权限不含任何写操作能力
func isReadPermission(key string) bool {
	return strings.HasSuffix(key, ".read")
}

// AllPermissionKeys 返回权限目录中的全量权限 Key
func AllPermissionKeys() []string {
	keys := make([]string, 0, len(permissionCatalog))
	for _, item := range permissionCatalog {
		keys = append(keys, item.Key)
	}
	return keys
}

// ReadPermissionKeys 返回全部只读类权限 Key，用于聚合入口的最小权限校验
func ReadPermissionKeys() []string {
	keys := make([]string, 0, len(permissionCatalog))
	for _, item := range permissionCatalog {
		if isReadPermission(item.Key) {
			keys = append(keys, item.Key)
		}
	}
	return keys
}

// builtinRolePermissions 内置角色的默认权限集合，返回值已按隐含规则补全
func builtinRolePermissions(key string) []string {
	switch key {
	case "admin":
		return normalizePermissions(AllPermissionKeys())
	case "operator":
		return normalizePermissions([]string{
			"audit.read", "backup.dashboard.read",
			"backup.hosts.read", "backup.tasks.read", "backup.executions.read", "backup.alerts.read",
			"backup.notifications.read", "backup.agents.read",
			"backup.hosts.manage", "backup.tasks.manage", "backup.executions.manage", "backup.alerts.manage",
			"backup.notifications.manage", "backup.agents.manage",
			"settings.base.read", "settings.users.read", "settings.auth.read", "settings.notifications.read",
		})
	case "viewer":
		return normalizePermissions([]string{
			"audit.read", "backup.dashboard.read",
			"backup.hosts.read", "backup.tasks.read", "backup.executions.read", "backup.alerts.read",
			"backup.notifications.read", "backup.agents.read",
			"settings.base.read", "settings.users.read", "settings.auth.read", "settings.notifications.read",
		})
	}
	return nil
}
