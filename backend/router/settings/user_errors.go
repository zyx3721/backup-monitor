package settings

import (
	"errors"
	"net/http"

	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

func WriteUserWorkflowError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrUsernameExists) {
		common.WriteError(w, http.StatusConflict, "用户名已存在")
		return
	}
	if errors.Is(err, service.ErrCannotRemoveAdmin) {
		common.WriteError(w, http.StatusBadRequest, "默认管理员不能删除")
		return
	}
	message := err.Error()
	if message == "username is required" {
		common.WriteError(w, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if message == "password is required" {
		common.WriteError(w, http.StatusBadRequest, "密码不能为空")
		return
	}
	common.WriteError(w, http.StatusInternalServerError, "保存用户失败")
}
