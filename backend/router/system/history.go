package system

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/router/common"
	"net/http"
	"strconv"
	"strings"

	"backup-monitor-backend/internal/service"
)

type auditHistoryItem struct {
	ID        int64  `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Username  string `json:"username"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	IP        string `json:"ip"`
	Result    string `json:"result"`
	Detail    string `json:"detail"`
}

// handleHistory 返回结构化审计日志，未指定 limit 时返回全部保留记录
func (a *Router) handleHistory(w http.ResponseWriter, r *http.Request) {
	limit := -1
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	var total int64
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	query := `SELECT id, date, authuser, ip, sql, module, action, target, detail, result FROM audit_logs ORDER BY id DESC`
	args := []interface{}{}
	if limit >= 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := a.fetchRows(query, args...)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]auditHistoryItem, 0, len(rows))
	for _, row := range rows {
		username := strings.TrimSpace(appcommon.AsString(row["authuser"]))
		sqlText := appcommon.AsString(row["sql"])
		module := strings.TrimSpace(appcommon.AsString(row["module"]))
		action := strings.TrimSpace(appcommon.AsString(row["action"]))
		detail := strings.TrimSpace(appcommon.AsString(row["detail"]))
		if detail == "" {
			detail = sqlText
		}
		result := strings.TrimSpace(appcommon.AsString(row["result"]))
		if result == "" {
			result = service.AuditResultSuccess
		}
		target := strings.TrimSpace(appcommon.AsString(row["target"]))
		items = append(items, auditHistoryItem{
			ID:        appcommon.AsInt64(row["id"]),
			Timestamp: appcommon.TimestampUnix(appcommon.AsString(row["date"])),
			Username:  username,
			Module:    module,
			Action:    action,
			Target:    target,
			IP:        strings.TrimSpace(appcommon.AsString(row["ip"])),
			Result:    result,
			Detail:    detail,
		})
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
		"limit": limit,
	})
}
