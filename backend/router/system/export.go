package system

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

var exportModuleNames = map[string]string{
	"hosts":      "主机",
	"tasks":      "任务",
	"executions": "执行记录",
	"alerts":     "告警",
	"audit":      "审计日志",
}

// handleHistoryExport 记录一次数据导出审计，供各页面导出后回写
func (a *Router) handleHistoryExport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Module string `json:"module"`
		Count  int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	name, ok := exportModuleNames[body.Module]
	if !ok {
		common.WriteError(w, http.StatusBadRequest, "无效的导出模块")
		return
	}
	if body.Count < 0 {
		body.Count = 0
	}
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		return
	}
	_ = a.audit.RecordEvent(r.Context(), user.Username, common.ClientIP(r), service.AuditEvent{
		Module: body.Module,
		Action: "导出数据",
		Target: fmt.Sprintf("%d 条", body.Count),
		Detail: "所有" + name + "数据已导出",
		Result: service.AuditResultSuccess,
	})
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
