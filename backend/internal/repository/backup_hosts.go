package repository

import (
	"database/sql"
	"strings"

	"backup-monitor-backend/internal/domain"
)

// ListHosts 返回全部主机及其任务数。
func (r *BackupRepository) ListHosts() ([]domain.Host, error) {
	rows, err := r.db.Query(`SELECT h.id,h.hostname,COALESCE(h.node_id,''),h.ip,COALESCE(h.os,''),COALESCE(h.role,''),
		COALESCE(h.agent_version,''),COALESCE(h.agent_started_at,''),h.status,COALESCE(h.last_report_time,''),COALESCE(h.platform,''),
		COALESCE(h.arch,''),COALESCE(h.os_build,0),(SELECT COUNT(*) FROM tasks t WHERE t.host_id=h.id)
		FROM hosts h ORDER BY h.hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Host{}
	for rows.Next() {
		var h domain.Host
		if err := rows.Scan(&h.ID, &h.Hostname, &h.NodeID, &h.IP, &h.OS, &h.Role, &h.AgentVersion, &h.AgentStartedAt, &h.Status, &h.LastReportTime, &h.Platform, &h.Arch, &h.OSBuild, &h.TaskCount); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetHost 按 ID 读取主机。
func (r *BackupRepository) GetHost(id string) (*domain.Host, error) {
	row := r.db.QueryRow(`SELECT h.id,h.hostname,COALESCE(h.node_id,''),h.ip,COALESCE(h.os,''),COALESCE(h.role,''),
		COALESCE(h.agent_version,''),COALESCE(h.agent_started_at,''),h.status,COALESCE(h.last_report_time,''),COALESCE(h.platform,''),
		COALESCE(h.arch,''),COALESCE(h.os_build,0),(SELECT COUNT(*) FROM tasks t WHERE t.host_id=h.id) FROM hosts h WHERE h.id=?`, id)
	var h domain.Host
	if err := row.Scan(&h.ID, &h.Hostname, &h.NodeID, &h.IP, &h.OS, &h.Role, &h.AgentVersion, &h.AgentStartedAt, &h.Status, &h.LastReportTime, &h.Platform, &h.Arch, &h.OSBuild, &h.TaskCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}

// FindHostByNode 按稳定机器标识查找主机。
func (r *BackupRepository) FindHostByNode(nodeID string) (id string, hostname string, err error) {
	if strings.TrimSpace(nodeID) == "" {
		return "", "", nil
	}
	err = r.db.QueryRow(`SELECT id,hostname FROM hosts WHERE node_id=?`, nodeID).Scan(&id, &hostname)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return id, hostname, err
}

// FindHostByName 按主机名查找主机。
func (r *BackupRepository) FindHostByName(hostname string) (id string, err error) {
	err = r.db.QueryRow(`SELECT id FROM hosts WHERE hostname=?`, hostname).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// InsertHost 新增主机并返回 ID。
func (r *BackupRepository) InsertHost(host domain.IngestHost, platform string) (string, error) {
	id := newID()
	_, err := r.db.Exec(`INSERT INTO hosts(id,hostname,node_id,ip,os,arch,os_build,role,agent_version,agent_started_at,platform,status,last_report_time,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?, 'normal', ?, ?)`,
		id, host.Hostname, nullable(host.NodeID), host.IP, host.OS, host.Arch, host.OSBuild, host.Role, host.AgentVersion, host.AgentStartedAt, platform, nowISO(), nowISO())
	if err != nil {
		return "", err
	}
	return id, nil
}

// DeleteHost 删除主机及其任务、执行记录与告警。
func (r *BackupRepository) DeleteHost(id string) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM executions WHERE task_id IN (SELECT id FROM tasks WHERE host_id=?)`, id); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`DELETE FROM alerts WHERE host_id=? OR task_id IN (SELECT id FROM tasks WHERE host_id=?)`, id, id); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`DELETE FROM tasks WHERE host_id=?`, id); err != nil {
		return false, err
	}
	res, err := tx.Exec(`DELETE FROM hosts WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return affected > 0, nil
}

// UpdateHostRole 修改主机角色；role 为空表示清空。
func (r *BackupRepository) UpdateHostRole(id, role string) (bool, error) {
	res, err := r.db.Exec(`UPDATE hosts SET role=? WHERE id=?`, role, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// ListTasks 返回全部任务（含主机 IP）。
func (r *BackupRepository) ListTasks() ([]domain.BackupTask, error) {
	rows, err := r.db.Query(`SELECT t.id,t.host_id,t.host_name,t.name,COALESCE(t.task_key,''),t.type,COALESCE(t.tool,''),
		COALESCE(t.schedule,''),COALESCE(t.schedule_cron,''),t.schedule_locked,t.status,COALESCE(t.last_execution_time,''),t.config,
		COALESCE(h.ip,'')
		FROM tasks t LEFT JOIN hosts h ON h.id=t.host_id ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BackupTask{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// ListTasksByHost 返回指定主机的任务。
func (r *BackupRepository) ListTasksByHost(hostID string) ([]domain.BackupTask, error) {
	rows, err := r.db.Query(`SELECT t.id,t.host_id,t.host_name,t.name,COALESCE(t.task_key,''),t.type,COALESCE(t.tool,''),
		COALESCE(t.schedule,''),COALESCE(t.schedule_cron,''),t.schedule_locked,t.status,COALESCE(t.last_execution_time,''),t.config,
		COALESCE(h.ip,'')
		FROM tasks t LEFT JOIN hosts h ON h.id=t.host_id WHERE t.host_id=? ORDER BY t.name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BackupTask{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// GetTask 按 ID 读取任务。
func (r *BackupRepository) GetTask(id string) (*domain.BackupTask, error) {
	row := r.db.QueryRow(`SELECT t.id,t.host_id,t.host_name,t.name,COALESCE(t.task_key,''),t.type,COALESCE(t.tool,''),
		COALESCE(t.schedule,''),COALESCE(t.schedule_cron,''),t.schedule_locked,t.status,COALESCE(t.last_execution_time,''),t.config,
		COALESCE(h.ip,'')
		FROM tasks t LEFT JOIN hosts h ON h.id=t.host_id WHERE t.id=?`, id)
	task, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// DeleteTask 删除任务及其执行记录与告警。
func (r *BackupRepository) DeleteTask(id string) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM executions WHERE task_id=?`, id); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`DELETE FROM alerts WHERE task_id=?`, id); err != nil {
		return false, err
	}
	res, err := tx.Exec(`DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return affected > 0, nil
}

// UpdateTaskSchedule 保存任务的 cron 计划。
func (r *BackupRepository) UpdateTaskSchedule(id, cron string) error {
	_, err := r.db.Exec(`UPDATE tasks SET schedule_cron=?, schedule='', schedule_locked=0 WHERE id=?`, cron, id)
	return err
}

// taskScanner 抽象 *sql.Row 与 *sql.Rows 的扫描接口。
type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (domain.BackupTask, error) {
	var task domain.BackupTask
	var scheduleLocked int
	var configRaw string
	if err := scanner.Scan(&task.ID, &task.HostID, &task.HostName, &task.Name, &task.TaskKey, &task.Type, &task.Tool,
		&task.Schedule, &task.ScheduleCron, &scheduleLocked, &task.Status, &task.LastExecutionTime, &configRaw, &task.HostIP); err != nil {
		return task, err
	}
	task.ScheduleLocked = scheduleLocked == 1
	task.Config = domain.ParseJSONObject(configRaw)
	if retention, ok := task.Config["retention"].(string); ok {
		task.Retention = retention
	}
	return task, nil
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// AllHostsForPush 返回推送自更新所需的轻量主机列表。
func (r *BackupRepository) AllHostsForPush() ([]domain.Host, error) {
	rows, err := r.db.Query(`SELECT id,hostname,ip,COALESCE(agent_version,''),COALESCE(agent_started_at,''),COALESCE(platform,''),COALESCE(os,''),COALESCE(arch,''),COALESCE(os_build,0) FROM hosts ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Host{}
	for rows.Next() {
		var h domain.Host
		if err := rows.Scan(&h.ID, &h.Hostname, &h.IP, &h.AgentVersion, &h.AgentStartedAt, &h.Platform, &h.OS, &h.Arch, &h.OSBuild); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// CountHostCascade 统计删除主机时将一并清理的任务、执行记录与告警数量
func (r *BackupRepository) CountHostCascade(id string) (tasks, executions, alerts int, err error) {
	if err = r.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE host_id=?`, id).Scan(&tasks); err != nil {
		return 0, 0, 0, err
	}
	if err = r.db.QueryRow(`SELECT COUNT(*) FROM executions WHERE task_id IN (SELECT id FROM tasks WHERE host_id=?)`, id).Scan(&executions); err != nil {
		return 0, 0, 0, err
	}
	if err = r.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE host_id=? OR task_id IN (SELECT id FROM tasks WHERE host_id=?)`, id, id).Scan(&alerts); err != nil {
		return 0, 0, 0, err
	}
	return tasks, executions, alerts, nil
}

// CountTaskCascade 统计删除任务时将一并清理的执行记录与告警数量
func (r *BackupRepository) CountTaskCascade(id string) (executions, alerts int, err error) {
	if err = r.db.QueryRow(`SELECT COUNT(*) FROM executions WHERE task_id=?`, id).Scan(&executions); err != nil {
		return 0, 0, err
	}
	if err = r.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE task_id=?`, id).Scan(&alerts); err != nil {
		return 0, 0, err
	}
	return executions, alerts, nil
}
