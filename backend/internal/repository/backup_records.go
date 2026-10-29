package repository

import (
	"database/sql"

	"backup-monitor-backend/internal/domain"
)

// ListExecutions 返回执行记录（按创建时间倒序），limit<=0 表示不限制。
func (r *BackupRepository) ListExecutions(limit int) ([]domain.BackupExecution, error) {
	query := `SELECT e.id,e.task_id,COALESCE(t.name,''),COALESCE(t.host_id,''),COALESCE(t.host_name,''),COALESCE(t.type,''),
		COALESCE(e.session_id,''),COALESCE(e.job_id,''),COALESCE(e.oib_id,''),e.state,COALESCE(e.trigger,''),
		e.creation_time,COALESCE(e.start_time,''),COALESCE(e.end_time,''),COALESCE(e.processed_size,''),
		COALESCE(e.transferred_size,''),COALESCE(e.artifact_path,''),e.exit_code,e.attempt,e.max_attempts,e.is_last_attempt,
		COALESCE(e.message,'')
		FROM executions e LEFT JOIN tasks t ON t.id=e.task_id ORDER BY e.creation_time DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BackupExecution{}
	for rows.Next() {
		exec, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, exec)
	}
	return out, rows.Err()
}

// ListExecutionsByTask 返回指定任务的执行记录。
func (r *BackupRepository) ListExecutionsByTask(taskID string) ([]domain.BackupExecution, error) {
	rows, err := r.db.Query(`SELECT e.id,e.task_id,COALESCE(t.name,''),COALESCE(t.host_id,''),COALESCE(t.host_name,''),COALESCE(t.type,''),
		COALESCE(e.session_id,''),COALESCE(e.job_id,''),COALESCE(e.oib_id,''),e.state,COALESCE(e.trigger,''),
		e.creation_time,COALESCE(e.start_time,''),COALESCE(e.end_time,''),COALESCE(e.processed_size,''),
		COALESCE(e.transferred_size,''),COALESCE(e.artifact_path,''),e.exit_code,e.attempt,e.max_attempts,e.is_last_attempt,
		COALESCE(e.message,'')
		FROM executions e LEFT JOIN tasks t ON t.id=e.task_id WHERE e.task_id=? ORDER BY e.creation_time DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BackupExecution{}
	for rows.Next() {
		exec, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, exec)
	}
	return out, rows.Err()
}

// GetExecution 按 ID 读取执行记录。
func (r *BackupRepository) GetExecution(id string) (*domain.BackupExecution, error) {
	row := r.db.QueryRow(`SELECT e.id,e.task_id,COALESCE(t.name,''),COALESCE(t.host_id,''),COALESCE(t.host_name,''),COALESCE(t.type,''),
		COALESCE(e.session_id,''),COALESCE(e.job_id,''),COALESCE(e.oib_id,''),e.state,COALESCE(e.trigger,''),
		e.creation_time,COALESCE(e.start_time,''),COALESCE(e.end_time,''),COALESCE(e.processed_size,''),
		COALESCE(e.transferred_size,''),COALESCE(e.artifact_path,''),e.exit_code,e.attempt,e.max_attempts,e.is_last_attempt,
		COALESCE(e.message,'')
		FROM executions e LEFT JOIN tasks t ON t.id=e.task_id WHERE e.id=?`, id)
	exec, err := scanExecution(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

// DeleteExecution 删除执行记录。
func (r *BackupRepository) DeleteExecution(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM executions WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// LatestExecution 返回任务最近一次执行。
func (r *BackupRepository) LatestExecution(taskID string) (*domain.BackupExecution, error) {
	row := r.db.QueryRow(`SELECT id,task_id,'','','','',COALESCE(session_id,''),COALESCE(job_id,''),COALESCE(oib_id,''),state,
		COALESCE(trigger,''),creation_time,COALESCE(start_time,''),COALESCE(end_time,''),COALESCE(processed_size,''),
		COALESCE(transferred_size,''),COALESCE(artifact_path,''),exit_code,attempt,max_attempts,is_last_attempt,COALESCE(message,'')
		FROM executions WHERE task_id=? ORDER BY creation_time DESC LIMIT 1`, taskID)
	exec, err := scanExecution(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

// ExecutionStates 返回按任务分组的执行状态序列（创建时间倒序），供统计连续失败与成功率。
func (r *BackupRepository) ExecutionStates() (map[string][]domain.BackupExecution, error) {
	rows, err := r.db.Query(`SELECT id,task_id,'','','','',COALESCE(session_id,''),COALESCE(job_id,''),COALESCE(oib_id,''),state,
		COALESCE(trigger,''),creation_time,COALESCE(start_time,''),COALESCE(end_time,''),COALESCE(processed_size,''),
		COALESCE(transferred_size,''),COALESCE(artifact_path,''),exit_code,attempt,max_attempts,is_last_attempt,COALESCE(message,'')
		FROM executions ORDER BY creation_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]domain.BackupExecution{}
	for rows.Next() {
		exec, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		out[exec.TaskID] = append(out[exec.TaskID], exec)
	}
	return out, rows.Err()
}

// ListAlerts 返回全部告警（含任务类型）。
func (r *BackupRepository) ListAlerts() ([]domain.Alert, error) {
	rows, err := r.db.Query(`SELECT a.id,a.host_id,COALESCE(a.task_id,''),a.host_name,COALESCE(a.task_name,''),
		COALESCE(t.type,''),a.type,a.level,a.status,a.created_at,COALESCE(a.resolved_at,''),COALESCE(a.message,'')
		FROM alerts a LEFT JOIN tasks t ON t.id=a.task_id ORDER BY a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Alert{}
	for rows.Next() {
		var a domain.Alert
		if err := rows.Scan(&a.ID, &a.HostID, &a.TaskID, &a.HostName, &a.TaskName, &a.TaskType, &a.Type, &a.Level, &a.Status, &a.CreatedAt, &a.ResolvedAt, &a.Message); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAlert 按 ID 读取告警。
func (r *BackupRepository) GetAlert(id string) (*domain.Alert, error) {
	row := r.db.QueryRow(`SELECT a.id,a.host_id,COALESCE(a.task_id,''),a.host_name,COALESCE(a.task_name,''),
		COALESCE(t.type,''),a.type,a.level,a.status,a.created_at,COALESCE(a.resolved_at,''),COALESCE(a.message,'')
		FROM alerts a LEFT JOIN tasks t ON t.id=a.task_id WHERE a.id=?`, id)
	var a domain.Alert
	if err := row.Scan(&a.ID, &a.HostID, &a.TaskID, &a.HostName, &a.TaskName, &a.TaskType, &a.Type, &a.Level, &a.Status, &a.CreatedAt, &a.ResolvedAt, &a.Message); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// ResolveAlert 手动恢复告警。
func (r *BackupRepository) ResolveAlert(id, resolvedAt string) (bool, error) {
	res, err := r.db.Exec(`UPDATE alerts SET status='resolved', resolved_at=? WHERE id=? AND status='active'`, resolvedAt, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// DeleteAlert 删除单条告警。
func (r *BackupRepository) DeleteAlert(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM alerts WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

type executionScanner interface {
	Scan(dest ...any) error
}

func scanExecution(scanner executionScanner) (domain.BackupExecution, error) {
	var e domain.BackupExecution
	var exitCode, attempt, maxAttempts, isLast sql.NullInt64
	if err := scanner.Scan(&e.ID, &e.TaskID, &e.TaskName, &e.HostID, &e.HostName, &e.TaskType,
		&e.SessionID, &e.JobID, &e.OIBID, &e.State, &e.Trigger, &e.CreationTime, &e.StartTime, &e.EndTime,
		&e.ProcessedSize, &e.TransferredSize, &e.ArtifactPath, &exitCode, &attempt, &maxAttempts, &isLast, &e.Message); err != nil {
		return e, err
	}
	e.ExitCode = intPtr(exitCode)
	e.Attempt = intPtr(attempt)
	e.MaxAttempts = intPtr(maxAttempts)
	e.IsLastAttempt = boolPtr(isLast)
	return e, nil
}

// AlertExistsActive 判断指定范围是否已有同类型未处理告警。
func (r *BackupRepository) AlertExistsActive(alertType, taskID, hostID string) (id, level, message string, err error) {
	var row *sql.Row
	if taskID != "" {
		row = r.db.QueryRow(`SELECT id,level,COALESCE(message,'') FROM alerts WHERE type=? AND task_id=? AND status='active' LIMIT 1`, alertType, taskID)
	} else {
		row = r.db.QueryRow(`SELECT id,level,COALESCE(message,'') FROM alerts WHERE type=? AND host_id=? AND status='active' LIMIT 1`, alertType, hostID)
	}
	err = row.Scan(&id, &level, &message)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	return id, level, message, err
}

// AlertExistsRecent 判断抑制窗口内是否已产生同类型告警。
func (r *BackupRepository) AlertExistsRecent(alertType, taskID, hostID, since string) (bool, error) {
	var count int
	var err error
	if taskID != "" {
		err = r.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE type=? AND task_id=? AND created_at>=?`, alertType, taskID, since).Scan(&count)
	} else {
		err = r.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE type=? AND host_id=? AND created_at>=?`, alertType, hostID, since).Scan(&count)
	}
	return count > 0, err
}
