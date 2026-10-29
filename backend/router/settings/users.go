package settings

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/security"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
)

func replaceSettingsUserRoles(db *sql.DB, userID int64, roleKeys []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("DELETE FROM user_roles WHERE user_id=?", userID); err != nil {
		return err
	}
	for _, key := range uniqueStrings(roleKeys) {
		var roleID, disabled int64
		if err = tx.QueryRow("SELECT id,disabled FROM roles WHERE key=?", key).Scan(&roleID, &disabled); err != nil {
			return errors.New("选择的角色不存在")
		}
		if disabled != 0 {
			return errors.New("角色已禁用，不能分配")
		}
		if _, err = tx.Exec("INSERT OR IGNORE INTO user_roles(user_id,role_id) VALUES(?,?)", userID, roleID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type settingsUserBody struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
	RoleKeys    []string
	Disabled    bool
}

func (a *Router) handleSettingsUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rows, err := a.db.QueryContext(r.Context(), `
			SELECT id,username,userdesc,email,disabled,created_at,last_login_at
			FROM users
			ORDER BY username ASC`)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		type settingsUserRow struct {
			id, disabled                 int64
			createdAt, lastLoginAt       string
			username, description, email string
		}
		userRows := make([]settingsUserRow, 0)
		for rows.Next() {
			var item settingsUserRow
			if err := rows.Scan(&item.id, &item.username, &item.description, &item.email, &item.disabled, &item.createdAt, &item.lastLoginAt); err != nil {
				rows.Close()
				common.WriteError(w, 500, err.Error())
				return
			}
			userRows = append(userRows, item)
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
		for _, item := range userRows {
			row, err := a.buildSettingsUserItem(r.Context(), item.id, item.username, item.description, item.email, item.disabled, item.createdAt, item.lastLoginAt)
			if err != nil {
				common.WriteError(w, 500, err.Error())
				return
			}
			items = append(items, row)
		}
		if err := rows.Err(); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		common.WriteJSON(w, 200, map[string]any{"items": items, "total": len(items)})
		return
	}
	actor, _ := common.CurrentUser(r.Context())
	if r.Method == http.MethodDelete {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			common.WriteError(w, 400, "invalid id")
			return
		}
		if actor.ID == id {
			common.WriteError(w, 400, "不能删除当前登录用户")
			return
		}
		var targetDisabled int64
		var deletedUsername string
		if err = a.db.QueryRowContext(r.Context(), "SELECT disabled,username FROM users WHERE id=?", id).Scan(&targetDisabled, &deletedUsername); err != nil {
			common.WriteError(w, 404, "用户不存在")
			return
		}
		if targetDisabled == 0 {
			common.WriteError(w, 400, "请先禁用用户再删除")
			return
		}
		if err = a.userWorkflow.Delete(r.Context(), service.UserActor{Username: actor.Username, IP: common.ClientIP(r)}, id); err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "删除用户", deletedUsername, "用户 "+deletedUsername+" 已删除", service.AuditResultSuccess)
		for _, query := range []string{
			"DELETE FROM user_roles WHERE user_id=?",
			"DELETE FROM user_group_members WHERE user_id=?",
		} {
			if _, cleanupErr := a.db.ExecContext(r.Context(), query, id); cleanupErr != nil {
				common.WriteError(w, http.StatusInternalServerError, cleanupErr.Error())
				return
			}
		}
		common.WriteJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	var body settingsUserBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	username := strings.TrimSpace(body.Username)
	email := strings.TrimSpace(body.Email)
	displayName := strings.TrimSpace(body.DisplayName)
	if username == "" {
		common.WriteError(w, 400, "用户名不能为空")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		common.WriteError(w, 400, "请输入有效的邮箱地址")
		return
	}
	if displayName == "" {
		displayName = username
	}
	if len(body.Password) < 6 && (r.Method == http.MethodPost || body.Password != "") {
		common.WriteError(w, 400, "密码至少 6 个字符")
		return
	}
	if len(body.RoleKeys) == 0 {
		body.RoleKeys = []string{"viewer"}
	}
	if r.Method == http.MethodPost {
		var existing int64
		if err := a.db.QueryRowContext(r.Context(), "SELECT COUNT(id) FROM users WHERE LOWER(username)=LOWER(?)", username).Scan(&existing); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		if existing > 0 {
			common.WriteError(w, 409, "用户名已存在")
			return
		}
		hash, err := security.HashPassword(body.Password)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		now := appcommon.NowTimestamp()
		res, err := a.db.ExecContext(r.Context(), "INSERT INTO users (username,userdesc,pass,email,disabled,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", username, displayName, hash, email, boolInt(body.Disabled), now, now)
		if err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		id, _ := res.LastInsertId()
		if err := replaceSettingsUserRoles(a.db, id, body.RoleKeys); err != nil {
			_, _ = a.db.ExecContext(r.Context(), "DELETE FROM users WHERE id=?", id)
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "创建用户", username, "本地用户 "+username+" 已创建", service.AuditResultSuccess)
		item, err := a.settingsUserResponse(r.Context(), id)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		common.WriteJSON(w, 201, item)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, 400, "invalid id")
		return
	}
	if r.Method == http.MethodPut {
		if body.Disabled && actor.ID == id {
			common.WriteError(w, 400, "不能禁用当前登录用户")
			return
		}
		var oldUsername, oldDisplayName string
		if err := a.db.QueryRowContext(r.Context(), "SELECT username,COALESCE(userdesc,'') FROM users WHERE id=?", id).Scan(&oldUsername, &oldDisplayName); err != nil {
			common.WriteError(w, 404, "用户不存在")
			return
		}
		var oldEmail string
		_ = a.db.QueryRowContext(r.Context(), "SELECT COALESCE(email,'') FROM users WHERE id=?", id).Scan(&oldEmail)
		oldRoles := queryRoleKeys(a.db, "SELECT r.key FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=?", id)
		if strings.EqualFold(oldUsername, "admin") && (body.Disabled || !strings.EqualFold(username, oldUsername)) {
			common.WriteError(w, 400, "默认管理员不能禁用、删除或改名")
			return
		}
		err = a.userWorkflow.Update(r.Context(), service.UserActor{Username: actor.Username, IP: common.ClientIP(r)}, id, domain.UserPayload{Username: username, UserDesc: displayName, Password: body.Password, Email: email, Disabled: body.Disabled})
		if err != nil {
			WriteUserWorkflowError(w, err)
			return
		}
		if err = replaceSettingsUserRoles(a.db, id, body.RoleKeys); err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if detail := userChangeDetail(username, oldEmail, oldDisplayName, oldRoles, email, displayName, body.Password, body.RoleKeys); detail != "" {
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新用户", username, detail, service.AuditResultSuccess)
		}
		item, err := a.settingsUserResponse(r.Context(), id)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		common.WriteJSON(w, 200, item)
		return
	}
	common.WriteError(w, 405, "method not allowed")
}

// userChangeDetail 生成“更新用户”审计详情：变更项按 邮箱、显示名称、密码、角色 顺序合并，
// 如“用户 alice 已更新：邮箱、显示名称、密码、角色已修改”；无任何修改时返回空串（不记审计）
func userChangeDetail(username, oldEmail, oldDisplayName string, oldRoles []string, newEmail, newDisplayName, newPassword string, newRoles []string) string {
	changes := make([]string, 0, 4)
	if strings.TrimSpace(newEmail) != strings.TrimSpace(oldEmail) {
		changes = append(changes, "邮箱")
	}
	if strings.TrimSpace(newDisplayName) != strings.TrimSpace(oldDisplayName) {
		changes = append(changes, "显示名称")
	}
	if strings.TrimSpace(newPassword) != "" {
		changes = append(changes, "密码")
	}
	if !equalStringSets(oldRoles, uniqueStrings(newRoles)) {
		changes = append(changes, "角色")
	}
	if len(changes) == 0 {
		return ""
	}
	return "用户 " + username + " 已更新：" + mergeChangeItems(changes)
}

func (a *Router) handleSettingsUserDisabled(w http.ResponseWriter, r *http.Request) {
	actor, _ := common.CurrentUser(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct{ Disabled bool }
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	var username string
	if err := a.db.QueryRowContext(r.Context(), "SELECT username FROM users WHERE id=?", id).Scan(&username); err != nil {
		common.WriteError(w, http.StatusNotFound, "user not found")
		return
	}
	if strings.EqualFold(username, "admin") {
		common.WriteError(w, http.StatusBadRequest, "默认管理员不可禁用")
		return
	}
	if actor.ID == id {
		common.WriteError(w, http.StatusBadRequest, "不能禁用当前登录用户")
		return
	}
	_, err = a.db.ExecContext(r.Context(), "UPDATE users SET disabled=?,updated_at="+appcommon.SQLTimestampNow+" WHERE id=?", boolInt(body.Disabled), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordSettingsStatusEvent(r.Context(), actor.Username, common.ClientIP(r), "用户", username, body.Disabled)
	item, err := a.settingsUserResponse(r.Context(), id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, item)
}

func (a *Router) settingsUserResponse(ctx context.Context, id int64) (map[string]any, error) {
	var username, description, email string
	var disabled int64
	var createdAt, lastLoginAt string
	err := a.db.QueryRowContext(ctx, `
		SELECT username,userdesc,email,disabled,created_at,last_login_at
		FROM users WHERE id=?`, id).Scan(&username, &description, &email, &disabled, &createdAt, &lastLoginAt)
	if err != nil {
		return nil, err
	}
	return a.buildSettingsUserItem(ctx, id, username, description, email, disabled, createdAt, lastLoginAt)
}

func (a *Router) buildSettingsUserItem(ctx context.Context, id int64, username, description, email string, disabled int64, createdAt, lastLoginAt string) (map[string]any, error) {
	directRoles, effectiveRoles, permissions, err := UserAccess(a.db, id, username)
	if err != nil {
		return nil, err
	}
	roleKey := "viewer"
	if strings.EqualFold(username, "admin") {
		roleKey = "admin"
	} else if len(effectiveRoles) > 0 {
		roleKey, _ = effectiveRoles[0]["key"].(string)
	}
	return map[string]any{
		"id": strconv.FormatInt(id, 10), "username": username, "displayName": description,
		"email": email, "role": roleKey, "source": "local", "roles": effectiveRoles,
		"directRoles": directRoles, "effectiveUserRoles": effectiveRoles,
		"disabled": disabled != 0, "permissions": permissions,
		"effectiveUserPermissions": permissions,
		"createdAt":                formatTimestamp(createdAt),
		"lastLoginAt":              formatTimestamp(lastLoginAt),
	}, nil
}

func formatTimestamp(value string) string {
	return strings.TrimSpace(value)
}
