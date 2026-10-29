package auth

import (
	"backup-monitor-backend/router/common"
	"encoding/json"
	"net/http"

	"backup-monitor-backend/internal/security"
	"backup-monitor-backend/internal/service"
)

func (a *Router) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var body changePasswordBody
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(body.NewPassword) < 6 {
		common.WriteError(w, http.StatusBadRequest, "new password is too short")
		return
	}
	if body.NewPassword != body.ConfirmPassword {
		common.WriteError(w, http.StatusBadRequest, "password confirmation does not match")
		return
	}
	var stored string
	if err := a.db.QueryRowContext(r.Context(), "SELECT pass FROM users WHERE id=?", user.ID).Scan(&stored); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !security.VerifyPassword(stored, body.OldPassword) {
		common.WriteError(w, http.StatusBadRequest, "invalid current password")
		return
	}
	hashed, err := security.HashPassword(body.NewPassword)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err = a.db.ExecContext(r.Context(), "UPDATE users SET pass=? WHERE id=?", hashed, user.ID); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAuditEvent(r.Context(), user.Username, common.ClientIP(r), service.AuditModuleAuth, "修改密码", user.Username, "用户 "+user.Username+" 修改密码成功", service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
