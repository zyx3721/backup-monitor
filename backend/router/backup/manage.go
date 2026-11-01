package backup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
)

func (a *Router) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := a.svc.BuildDashboard()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, dashboard)
}

func (a *Router) handleHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := a.svc.Repo().ListHosts()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, service.FilterHostsByIDs(hosts, r.URL.Query().Get("ids")))
}

func (a *Router) handleHostRefreshPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := a.svc.HostRefreshPlan()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, plan)
}

func (a *Router) handleHost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host, err := a.svc.Repo().GetHost(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if host == nil {
		common.WriteError(w, http.StatusNotFound, "主机不存在")
		return
	}
	tasks, err := a.svc.Repo().ListTasksByHost(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, domain.HostDetail{Host: *host, Tasks: a.svc.DecorateTasks(tasks)})
}

func hostRoleText(role string) string {
	switch role {
	case "database":
		return "数据库"
	case "web":
		return "Web"
	case "application":
		return "应用"
	case "file":
		return "文件"
	case "virtualization":
		return "虚拟化"
	case "":
		return "未设置"
	default:
		return role
	}
}

func (a *Router) handleHostRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role *string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	role := ""
	if body.Role != nil {
		role = strings.TrimSpace(*body.Role)
	}
	if role != "" && !domain.ValidHostRoles[role] {
		common.WriteError(w, http.StatusBadRequest, "无效的主机角色")
		return
	}
	id := chi.URLParam(r, "id")
	host, _ := a.svc.Repo().GetHost(id)
	ok, err := a.svc.Repo().UpdateHostRole(id, role)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "主机不存在")
		return
	}
	target := id
	if host != nil {
		target = host.Hostname
	}
	a.recordAudit(r, service.AuditModuleHosts, "修改主机角色", target, "主机 "+target+" 角色已更新："+hostRoleText(role), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "role": role})
}

func (a *Router) handleDeleteHost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host, _ := a.svc.Repo().GetHost(id)
	tasks, executions, alerts, countErr := a.svc.Repo().CountHostCascade(id)
	if countErr != nil {
		common.WriteError(w, http.StatusInternalServerError, countErr.Error())
		return
	}
	ok, err := a.svc.Repo().DeleteHost(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "主机不存在")
		return
	}
	if a.hub != nil {
		a.hub.CloseHost(id)
	}
	name := id
	if host != nil {
		name = host.Hostname
	}
	detail := fmt.Sprintf("主机 %s 已删除，同时该主机下的 %d 条任务、%d 条执行记录、%d 条告警均已删除",
		name, tasks, executions, alerts)
	a.recordAudit(r, service.AuditModuleHosts, "删除主机", name, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.svc.Repo().ListTasks()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, a.svc.DecorateTasks(tasks))
}

func (a *Router) handleTask(w http.ResponseWriter, r *http.Request) {
	task, err := a.svc.Repo().GetTask(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		common.WriteError(w, http.StatusNotFound, "任务不存在")
		return
	}
	common.WriteJSON(w, http.StatusOK, a.svc.DecorateTasks([]domain.BackupTask{*task})[0])
}

func (a *Router) handleTaskExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if task, _ := a.svc.Repo().GetTask(id); task == nil {
		common.WriteError(w, http.StatusNotFound, "任务不存在")
		return
	}
	executions, err := a.svc.Repo().ListExecutionsByTask(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, executions)
}

func (a *Router) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, _ := a.svc.Repo().GetTask(id)
	executions, alerts, countErr := a.svc.Repo().CountTaskCascade(id)
	if countErr != nil {
		common.WriteError(w, http.StatusInternalServerError, countErr.Error())
		return
	}
	ok, err := a.svc.Repo().DeleteTask(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "任务不存在")
		return
	}
	name := id
	hostName := ""
	if task != nil {
		name = task.Name
		hostName = task.HostName
	}
	detail := fmt.Sprintf("主机 %s 的任务 %s 已删除，同时该任务下的 %d 条执行记录、%d 条告警均已删除",
		hostName, name, executions, alerts)
	a.recordAudit(r, service.AuditModuleTasks, "删除任务", name, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleTaskSchedule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ScheduleCron string `json:"scheduleCron"`
		ServerOnly   bool   `json:"serverOnly"`
		Unlock       bool   `json:"unlock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteError(w, http.StatusBadRequest, "请求体格式无效")
		return
	}
	id := chi.URLParam(r, "id")
	task, err := a.svc.Repo().GetTask(id)
	if err != nil || task == nil {
		common.WriteError(w, http.StatusNotFound, "任务不存在")
		return
	}
	if body.Unlock {
		if err := a.svc.Repo().UnlockTaskSchedule(id); err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.svc.RecomputeTaskStatus(id, time.Now())
		a.recordAudit(r, service.AuditModuleTasks, "解除执行计划锁定", task.Name,
			fmt.Sprintf("主机 %s 的任务 %s 执行计划已解除锁定，恢复由 Agent 上报覆盖", task.HostName, task.Name), service.AuditResultSuccess)
		common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "unlocked": true})
		return
	}
	cron := strings.TrimSpace(body.ScheduleCron)
	if !service.IsValidCron(cron) {
		common.WriteError(w, http.StatusBadRequest, "cron 表达式无效，应为 5 段式（分 时 日 月 周），如 0 23 * * 0")
		return
	}
	delivered, ok, message := false, true, "已保存为服务端判定计划，未下发 Agent"
	if !body.ServerOnly {
		if strings.TrimSpace(task.TaskKey) == "" {
			common.WriteError(w, http.StatusBadRequest, "该任务缺少 key 标签，无法下发修改执行计划")
			return
		}
		if cron == strings.TrimSpace(task.ScheduleCron) {
			common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "delivery": map[string]any{"delivered": true, "ok": true, "message": "计划未变化"}})
			return
		}
		delivered, ok, message = a.hub.SendSetCron(task.HostID, task.TaskKey, cron)
		if !delivered || !ok {
			common.WriteError(w, http.StatusBadRequest, "执行计划修改失败："+message)
			return
		}
	}
	if err := a.svc.Repo().UpdateTaskSchedule(id, cron, body.ServerOnly); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.svc.RecomputeTaskStatus(id, time.Now())
	detail := fmt.Sprintf("主机 %s 的任务 %s 执行计划已更新", task.HostName, task.Name)
	if body.ServerOnly {
		detail = fmt.Sprintf("主机 %s 的任务 %s 执行计划已更新（仅服务端判定，未下发 Agent）", task.HostName, task.Name)
	}
	a.recordAudit(r, service.AuditModuleTasks, "更新执行计划", task.Name, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "serverOnly": body.ServerOnly,
		"delivery": map[string]any{"delivered": delivered, "ok": ok, "message": message}})
}

func (a *Router) handleExecutions(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	executions, err := a.svc.Repo().ListExecutions(limit)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, executions)
}

func (a *Router) handleExecution(w http.ResponseWriter, r *http.Request) {
	execution, err := a.svc.Repo().GetExecution(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if execution == nil {
		common.WriteError(w, http.StatusNotFound, "执行记录不存在")
		return
	}
	common.WriteJSON(w, http.StatusOK, execution)
}

func (a *Router) handleDeleteExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	execution, _ := a.svc.Repo().GetExecution(id)
	ok, err := a.svc.Repo().DeleteExecution(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "执行记录不存在")
		return
	}
	target := id
	detail := "执行记录已删除"
	if execution != nil {
		target = firstNonEmpty(execution.TaskName, execution.ID)
		detail = fmt.Sprintf("主机 %s 的任务 %s （执行时间：%s）执行记录已删除",
			execution.HostName, execution.TaskName, formatAuditTime(execution.CreationTime))
	}
	a.recordAudit(r, service.AuditModuleExecutions, "删除执行记录", target, detail, service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleClearExecutions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	if len(body.IDs) == 0 {
		common.WriteError(w, http.StatusBadRequest, "没有可清空的执行记录")
		return
	}
	deleted, err := a.svc.Repo().ClearExecutions(body.IDs)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAudit(r, service.AuditModuleExecutions, "清空执行记录", "执行记录",
		fmt.Sprintf("已清空 %d 条执行记录", deleted), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted})
}

func (a *Router) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := a.svc.Repo().ListAlerts()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, alerts)
}

func (a *Router) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	alert, _ := a.svc.Repo().GetAlert(id)
	ok, err := a.svc.Repo().ResolveAlert(id, timeNow())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "告警不存在或已恢复")
		return
	}
	name := id
	if alert != nil {
		name = firstNonEmpty(alert.TaskName, alert.HostName)
	}
	a.recordAudit(r, service.AuditModuleAlerts, "恢复告警", name, alertAuditDetail(alert, "已手动恢复"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	alert, _ := a.svc.Repo().GetAlert(id)
	ok, err := a.svc.Repo().DeleteAlert(id)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		common.WriteError(w, http.StatusNotFound, "告警不存在")
		return
	}
	name := id
	if alert != nil {
		name = firstNonEmpty(alert.TaskName, alert.HostName)
	}
	a.recordAudit(r, service.AuditModuleAlerts, "删除告警", name, alertAuditDetail(alert, "已删除"), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *Router) handleClearAlerts(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	if len(body.IDs) == 0 {
		common.WriteError(w, http.StatusBadRequest, "没有可清空的告警记录")
		return
	}
	deleted, err := a.svc.Repo().ClearAlerts(body.IDs)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.recordAudit(r, service.AuditModuleAlerts, "清空告警记录", "告警记录",
		fmt.Sprintf("已清空 %d 条告警记录", deleted), service.AuditResultSuccess)
	common.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": deleted})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
