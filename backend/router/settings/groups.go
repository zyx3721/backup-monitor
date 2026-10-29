package settings

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func replaceSettingsGroupRelations(db *sql.DB, groupID int64, memberIDs, roleKeys []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	validatedMembers := make([]int64, 0, len(memberIDs))
	seenMembers := map[int64]struct{}{}
	for _, rawID := range memberIDs {
		id, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
		if err != nil {
			return errors.New("成员用户不存在")
		}
		if _, exists := seenMembers[id]; exists {
			continue
		}
		var exists int
		if err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE id=?", id).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			return errors.New("成员用户不存在")
		}
		seenMembers[id] = struct{}{}
		validatedMembers = append(validatedMembers, id)
	}

	type roleRelation struct{ id int64 }
	validatedRoles := make([]roleRelation, 0, len(roleKeys))
	for _, key := range uniqueStrings(roleKeys) {
		var roleID, disabled int64
		if err = tx.QueryRow("SELECT id,disabled FROM roles WHERE key=?", key).Scan(&roleID, &disabled); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("选择的角色不存在")
			}
			return err
		}
		if disabled != 0 {
			return errors.New("角色已禁用，不能分配")
		}
		validatedRoles = append(validatedRoles, roleRelation{id: roleID})
	}

	if _, err = tx.Exec("DELETE FROM user_group_members WHERE group_id=?", groupID); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM user_group_roles WHERE group_id=?", groupID); err != nil {
		return err
	}
	for _, id := range validatedMembers {
		if _, err = tx.Exec("INSERT INTO user_group_members(group_id,user_id) VALUES(?,?)", groupID, id); err != nil {
			return err
		}
	}
	for _, role := range validatedRoles {
		if _, err = tx.Exec("INSERT INTO user_group_roles(group_id,role_id) VALUES(?,?)", groupID, role.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func settingsGroupRelations(db *sql.DB, groupID int64) ([]map[string]any, []map[string]any, error) {
	members := []map[string]any{}
	rows, err := db.Query("SELECT u.id,u.username,u.userdesc FROM user_group_members m JOIN users u ON u.id=m.user_id WHERE m.group_id=? ORDER BY u.username", groupID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id int64
		var username, desc string
		if err := rows.Scan(&id, &username, &desc); err != nil {
			rows.Close()
			return nil, nil, err
		}
		members = append(members, map[string]any{"id": strconv.FormatInt(id, 10), "username": username, "displayName": desc})
	}
	rows.Close()
	roles := []map[string]any{}
	rrows, err := db.Query("SELECT r.id,r.key,r.name,r.description,r.permissions,r.builtin FROM user_group_roles g JOIN roles r ON r.id=g.role_id WHERE g.group_id=? ORDER BY r.name", groupID)
	if err != nil {
		return nil, nil, err
	}
	for rrows.Next() {
		var id int64
		var key, name, desc, raw string
		var builtin int64
		if err := rrows.Scan(&id, &key, &name, &desc, &raw, &builtin); err != nil {
			rrows.Close()
			return nil, nil, err
		}
		var permissions []string
		_ = json.Unmarshal([]byte(raw), &permissions)
		roles = append(roles, map[string]any{"id": strconv.FormatInt(id, 10), "key": key, "name": name, "description": desc, "permissions": permissions, "builtin": builtin != 0})
	}
	rrows.Close()
	return members, roles, nil
}

type settingsGroupBody struct {
	Name        string
	Description string
	Disabled    bool
	MemberIDs   []string
	RoleKeys    []string
}

func (a *Router) handleSettingsGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.QueryContext(r.Context(), "SELECT id,name,description,disabled FROM user_groups ORDER BY name ASC")
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		type settingsGroupRow struct {
			id, disabled      int64
			name, description string
		}
		groupRows := make([]settingsGroupRow, 0)
		for rows.Next() {
			var item settingsGroupRow
			if err := rows.Scan(&item.id, &item.name, &item.description, &item.disabled); err != nil {
				rows.Close()
				common.WriteError(w, 500, err.Error())
				return
			}
			groupRows = append(groupRows, item)
		}
		if err := rows.Close(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		if err := rows.Err(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		items := []map[string]any{}
		for _, item := range groupRows {
			members, roles, relationErr := settingsGroupRelations(a.db, item.id)
			if relationErr != nil {
				common.WriteError(w, 500, relationErr.Error())
				return
			}
			items = append(items, map[string]any{"id": strconv.FormatInt(item.id, 10), "name": item.name, "description": item.description, "disabled": item.disabled != 0, "members": members, "roles": roles})
		}
		common.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
		return
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		actor, _ := common.CurrentUser(r.Context())
		var body settingsGroupBody
		if !common.DecodeJSON(w, r, &body) {
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			common.WriteError(w, 400, "用户群组名称不能为空")
			return
		}
		now := appcommon.NowTimestamp()
		var id int64
		var err error
		var previousDisabled int64
		var oldDescription string
		var oldMembers []int64
		var oldRoles []string
		if r.Method == http.MethodPost {
			res, e := a.db.ExecContext(r.Context(), "INSERT INTO user_groups(name,description,disabled,created_at,updated_at) VALUES(?,?,?,?,?)", name, strings.TrimSpace(body.Description), boolInt(body.Disabled), now, now)
			err = e
			if err == nil {
				id, _ = res.LastInsertId()
			}
		} else {
			id, err = strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
			if err == nil {
				_ = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(disabled,0),COALESCE(description,'') FROM user_groups WHERE id=?", id).Scan(&previousDisabled, &oldDescription)
				oldMembers = queryInt64Set(a.db, "SELECT user_id FROM user_group_members WHERE group_id=?", id)
				oldRoles = queryRoleKeys(a.db, "SELECT r.key FROM user_group_roles gr JOIN roles r ON r.id=gr.role_id WHERE gr.group_id=?", id)
				_, err = a.db.ExecContext(r.Context(), "UPDATE user_groups SET name=?,description=?,disabled=?,updated_at=? WHERE id=?", name, strings.TrimSpace(body.Description), boolInt(body.Disabled), now, id)
			}
		}
		if err != nil {
			if isUniqueViolation(err) {
				common.WriteError(w, 409, fmt.Sprintf("用户群组 %s 已存在", name))
				return
			}
			common.WriteError(w, 500, "保存用户群组失败")
			return
		}
		if err := replaceSettingsGroupRelations(a.db, id, body.MemberIDs, body.RoleKeys); err != nil {
			if isUniqueViolation(err) {
				common.WriteError(w, 409, "用户群组已存在")
				return
			}
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if r.Method == http.MethodPost {
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "创建用户群组", name, "用户群组 "+name+" 已创建", service.AuditResultSuccess)
		} else {
			if previousDisabled != boolInt(body.Disabled) {
				a.recordSettingsStatusEvent(r.Context(), actor.Username, common.ClientIP(r), "用户群组", name, body.Disabled)
			}
			detail := groupChangeDetail(name, oldDescription, oldMembers, oldRoles, body)
			if detail != "" {
				a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新用户群组", name, detail, service.AuditResultSuccess)
			}
		}
		members, roles, relationErr := settingsGroupRelations(a.db, id)
		if relationErr != nil {
			common.WriteError(w, http.StatusInternalServerError, relationErr.Error())
			return
		}
		common.WriteJSON(w, 200, map[string]any{"id": strconv.FormatInt(id, 10), "name": name, "description": body.Description, "disabled": body.Disabled, "members": members, "roles": roles})
		return
	}
	if r.Method == http.MethodDelete {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			common.WriteError(w, 400, "invalid id")
			return
		}
		var groupName string
		if err := a.db.QueryRowContext(r.Context(), "SELECT COALESCE(name,'') FROM user_groups WHERE id=?", id).Scan(&groupName); err != nil {
			common.WriteError(w, 404, "用户群组不存在")
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
			"DELETE FROM user_group_members WHERE group_id=?",
			"DELETE FROM user_group_roles WHERE group_id=?",
			"DELETE FROM user_groups WHERE id=?",
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
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "删除用户群组", groupName, "用户群组 "+groupName+" 已删除", service.AuditResultSuccess)
		common.WriteJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	common.WriteError(w, 405, "method not allowed")
}

// groupChangeDetail 生成“更新用户群组”审计详情：变更项按 描述、成员、角色 顺序合并，
// 如“用户群组 运维组 已更新：描述、成员、角色已修改”；除启用状态外无其他变更时返回空串
func groupChangeDetail(name, oldDescription string, oldMembers []int64, oldRoles []string, body settingsGroupBody) string {
	changes := make([]string, 0, 3)
	if strings.TrimSpace(body.Description) != strings.TrimSpace(oldDescription) {
		changes = append(changes, "描述")
	}
	if !equalInt64Sets(oldMembers, parseGroupMemberIDs(body.MemberIDs)) {
		changes = append(changes, "成员")
	}
	if !equalStringSets(oldRoles, uniqueStrings(body.RoleKeys)) {
		changes = append(changes, "角色")
	}
	if len(changes) == 0 {
		return ""
	}
	return "用户群组 " + name + " 已更新：" + mergeChangeItems(changes)
}

// parseGroupMemberIDs 群组成员 ID 列表转升序去重的整型集合
func parseGroupMemberIDs(rawIDs []string) []int64 {
	seen := map[int64]bool{}
	values := make([]int64, 0, len(rawIDs))
	for _, raw := range rawIDs {
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}
