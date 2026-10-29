package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"

	"backup-monitor-backend/internal/service"
)

func (a *Router) auditService() *service.AuditService {
	if a.audit == nil {
		a.audit = service.NewAuditService(a.db)
	}
	return a.audit
}

// recordAuditEvent 写入一条与 SQL 无关的显式审计事件（登录、连接测试、配置变更等）
func (a *Router) recordAuditEvent(ctx context.Context, username, ip, module, action, target, detail, result string) {
	event := service.AuditEvent{Module: module, Action: action, Target: target, Detail: detail, Result: result}
	_ = a.auditService().RecordEvent(ctx, username, ip, event)
}

// recordSettingsStatusEvent 记录用户/角色/群组的启用、禁用审计事件（系统配置模块）
func (a *Router) recordSettingsStatusEvent(ctx context.Context, operator, ip, entity, name string, disabled bool) {
	status := "已启用"
	action := "启用" + entity
	if disabled {
		status = "已禁用"
		action = "禁用" + entity
	}
	a.recordAuditEvent(ctx, operator, ip, service.AuditModuleSettings, action, name, entity+" "+name+" 状态更新为 "+status, service.AuditResultSuccess)
}

// queryRoleKeys 查询角色标识集合：users 表传用户直接分配角色，群组传群组角色
func queryRoleKeys(db *sql.DB, query string, args ...interface{}) []string {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if rows.Scan(&key) == nil && strings.TrimSpace(key) != "" {
			keys = append(keys, strings.TrimSpace(key))
		}
	}
	sort.Strings(keys)
	return keys
}

// queryInt64Set 查询整型 ID 集合（群组成员等），升序返回
func queryInt64Set(db *sql.DB, query string, args ...interface{}) []int64 {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	values := make([]int64, 0)
	for rows.Next() {
		var value int64
		if rows.Scan(&value) == nil {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

// equalStringSets 判断两个字符串集合去重排序后是否一致
func equalStringSets(left, right []string) bool {
	leftSorted := uniqueStrings(left)
	rightSorted := uniqueStrings(right)
	sort.Strings(leftSorted)
	sort.Strings(rightSorted)
	if len(leftSorted) != len(rightSorted) {
		return false
	}
	for index := range leftSorted {
		if leftSorted[index] != rightSorted[index] {
			return false
		}
	}
	return true
}

// equalInt64Sets 判断两个整型集合（升序归一后）是否一致
func equalInt64Sets(left, right []int64) bool {
	leftSorted := append([]int64(nil), left...)
	rightSorted := append([]int64(nil), right...)
	sort.Slice(leftSorted, func(i, j int) bool { return leftSorted[i] < leftSorted[j] })
	sort.Slice(rightSorted, func(i, j int) bool { return rightSorted[i] < rightSorted[j] })
	if len(leftSorted) != len(rightSorted) {
		return false
	}
	for index := range leftSorted {
		if leftSorted[index] != rightSorted[index] {
			return false
		}
	}
	return true
}

// mergeChangeItems 多个变更项合并为“邮箱、显示名称、密码、角色已修改”样式的详情尾部
func mergeChangeItems(items []string) string {
	return strings.Join(items, "、") + "已修改"
}

// permissionKeys 返回权限集合：admin 为全量权限，普通账号为只读类权限
func permissionKeys(admin bool) []string {
	if admin {
		return AllPermissionKeys()
	}
	return ReadPermissionKeys()
}

func stringConfig(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}

func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func mapBool(config map[string]any, key string) bool { value, _ := config[key].(bool); return value }

func mustJSON(value any) string { raw, _ := json.Marshal(value); return string(raw) }
