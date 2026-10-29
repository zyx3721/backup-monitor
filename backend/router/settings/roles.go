package settings

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type settingsRoleBody struct {
	Key         string
	Name        string
	Description string
	Permissions []string
}

func (a *Router) handleSettingsRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.QueryContext(r.Context(), "SELECT id,key,name,description,permissions,builtin,disabled,created_at,updated_at FROM roles ORDER BY builtin DESC,name ASC")
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()
		items := make([]map[string]any, 0)
		for rows.Next() {
			var id, builtin, disabled int64
			var created, updated string
			var key, name, description, raw string
			if err := rows.Scan(&id, &key, &name, &description, &raw, &builtin, &disabled, &created, &updated); err != nil {
				common.WriteError(w, 500, err.Error())
				return
			}
			var permissions []string
			_ = json.Unmarshal([]byte(raw), &permissions)
			items = append(items, map[string]any{"id": strconv.FormatInt(id, 10), "key": key, "name": name, "description": description, "permissions": normalizePermissions(permissions), "builtin": builtin != 0, "disabled": disabled != 0, "createdAt": formatTimestamp(created), "updatedAt": formatTimestamp(updated)})
		}
		common.WriteJSON(w, 200, map[string]any{"items": items, "total": len(items)})
		return
	}
	if r.Method == http.MethodDelete {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			common.WriteError(w, 400, "invalid id")
			return
		}
		var builtin int64
		var roleName string
		if err = a.db.QueryRowContext(r.Context(), "SELECT builtin,name FROM roles WHERE id=?", id).Scan(&builtin, &roleName); err != nil {
			common.WriteError(w, 404, "角色不存在")
			return
		}
		if builtin != 0 {
			common.WriteError(w, 400, "内置角色不可删除")
			return
		}
		actor, _ := common.CurrentUser(r.Context())
		tx, txErr := a.db.BeginTx(r.Context(), nil)
		if txErr != nil {
			common.WriteError(w, 500, txErr.Error())
			return
		}
		defer tx.Rollback()
		for _, query := range []string{
			"DELETE FROM user_roles WHERE role_id=?",
			"DELETE FROM user_group_roles WHERE role_id=?",
			"DELETE FROM roles WHERE id=?",
		} {
			if _, txErr = tx.ExecContext(r.Context(), query, id); txErr != nil {
				common.WriteError(w, 500, txErr.Error())
				return
			}
		}
		if txErr = tx.Commit(); txErr != nil {
			common.WriteError(w, 500, txErr.Error())
			return
		}
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "删除用户角色", roleName, "用户角色 "+roleName+" 已删除", service.AuditResultSuccess)
		common.WriteJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	actor, _ := common.CurrentUser(r.Context())
	var body settingsRoleBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	key, name := strings.TrimSpace(body.Key), strings.TrimSpace(body.Name)
	if key == "" {
		common.WriteError(w, 400, "角色标识不能为空")
		return
	}
	if !roleKeyPattern.MatchString(key) {
		common.WriteError(w, 400, "角色标识需为小写字母、数字、点、下划线或连字符")
		return
	}
	if name == "" {
		common.WriteError(w, 400, "角色名称不能为空")
		return
	}
	raw := mustJSON(normalizePermissions(body.Permissions))
	now := appcommon.NowTimestamp()
	if r.Method == http.MethodPost {
		res, err := a.db.ExecContext(r.Context(), "INSERT INTO roles(key,name,description,permissions,builtin,created_at,updated_at) VALUES(?,?,?,?,0,?,?)", key, name, strings.TrimSpace(body.Description), raw, now, now)
		if err != nil {
			if isUniqueViolation(err) {
				common.WriteError(w, 409, "角色标识已存在")
				return
			}
			common.WriteError(w, 500, "保存用户角色失败")
			return
		}
		id, _ := res.LastInsertId()
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "创建用户角色", name, "用户角色 "+name+" 已创建", service.AuditResultSuccess)
		common.WriteJSON(w, 201, map[string]any{"id": strconv.FormatInt(id, 10), "key": key, "name": name, "description": body.Description, "permissions": normalizePermissions(body.Permissions), "builtin": false, "disabled": false})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, 400, "invalid id")
		return
	}
	if r.Method == http.MethodPut {
		var putBuiltin int64
		var oldDescription, oldPermissionsRaw string
		if err = a.db.QueryRowContext(r.Context(), "SELECT builtin,COALESCE(description,''),COALESCE(permissions,'[]') FROM roles WHERE id=?", id).Scan(&putBuiltin, &oldDescription, &oldPermissionsRaw); err != nil {
			common.WriteError(w, 404, "角色不存在")
			return
		}
		if putBuiltin != 0 {
			common.WriteError(w, 400, "内置角色不可修改")
			return
		}
		if _, err = a.db.ExecContext(r.Context(), "UPDATE roles SET key=?,name=?,description=?,permissions=?,updated_at=? WHERE id=?", key, name, strings.TrimSpace(body.Description), raw, now, id); err != nil {
			if isUniqueViolation(err) {
				common.WriteError(w, 409, "角色标识已存在")
				return
			}
			common.WriteError(w, 500, "保存用户角色失败")
			return
		}
		var roleKey, roleName, roleDescription, roleRaw string
		var roleBuiltin, roleDisabled int64
		var roleCreatedAt, roleUpdatedAt string
		if err = a.db.QueryRowContext(r.Context(), "SELECT r.key,r.name,r.description,r.permissions,r.builtin,r.disabled,r.created_at,r.updated_at FROM roles r WHERE r.id=?", id).Scan(&roleKey, &roleName, &roleDescription, &roleRaw, &roleBuiltin, &roleDisabled, &roleCreatedAt, &roleUpdatedAt); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		oldPerms := normalizePermissions(unwrapJSONArray(oldPermissionsRaw))
		newPerms := normalizePermissions(body.Permissions)
		if detail := roleChangeDetail(name, oldDescription, oldPerms, body.Description, newPerms); detail != "" {
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新用户角色", name, detail, service.AuditResultSuccess)
		}
		common.WriteJSON(w, 200, settingsRoleMap(id, roleKey, roleName, roleDescription, roleRaw, roleBuiltin, roleDisabled, roleCreatedAt, roleUpdatedAt))
		return
	}
	common.WriteError(w, 405, "method not allowed")
}

// roleChangeDetail 生成“更新用户角色”审计详情：变更项按 描述、权限 顺序合并，
// 如“用户角色 auditor 已更新：描述、权限已修改”；无任何修改时返回空串（不记审计）
func roleChangeDetail(name, oldDescription string, oldPerms []string, newDescription string, newPerms []string) string {
	changes := make([]string, 0, 2)
	if strings.TrimSpace(newDescription) != strings.TrimSpace(oldDescription) {
		changes = append(changes, "描述")
	}
	if !equalStringSets(oldPerms, newPerms) {
		changes = append(changes, "权限")
	}
	if len(changes) == 0 {
		return ""
	}
	return "用户角色 " + name + " 已更新：" + mergeChangeItems(changes)
}

// unwrapJSONArray 解析权限 JSON 数组为字符串切片，解析失败返回空切片
func unwrapJSONArray(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func (a *Router) handleSettingsRoleDisabled(w http.ResponseWriter, r *http.Request) {
	actor, _ := common.CurrentUser(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var builtin int64
	var roleName string
	if err = a.db.QueryRowContext(r.Context(), "SELECT builtin,name FROM roles WHERE id=?", id).Scan(&builtin, &roleName); err != nil {
		common.WriteError(w, http.StatusNotFound, "role not found")
		return
	}
	if builtin != 0 {
		common.WriteError(w, http.StatusBadRequest, "内置角色不可禁用")
		return
	}
	var body struct{ Disabled bool }
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	_, err = a.db.ExecContext(r.Context(), "UPDATE roles SET disabled=?,updated_at="+appcommon.SQLTimestampNow+" WHERE id=?", boolInt(body.Disabled), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordSettingsStatusEvent(r.Context(), actor.Username, common.ClientIP(r), "用户角色", roleName, body.Disabled)
	var key, name, description, raw string
	var roleDisabled int64
	var createdAt, updatedAt string
	if err := a.db.QueryRowContext(r.Context(), "SELECT key,name,description,permissions,disabled,created_at,updated_at FROM roles WHERE id=?", id).Scan(&key, &name, &description, &raw, &roleDisabled, &createdAt, &updatedAt); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, settingsRoleMap(id, key, name, description, raw, builtin, roleDisabled, createdAt, updatedAt))
}

// roleKeyPattern 角色标识规则：小写字母开头，后接小写字母、数字、点、下划线或连字符
var roleKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,40}$`)

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToUpper(err.Error()), "UNIQUE CONSTRAINT FAILED")
}
