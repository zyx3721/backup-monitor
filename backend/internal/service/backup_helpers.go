package service

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"backup-monitor-backend/internal/domain"
)

func newUUID() string {
	return uuid.NewString()
}

func marshalJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// preferredExecutionTime 优先取实际开始时间，缺失时回退到创建时间
func preferredExecutionTime(startTime, creationTime string) string {
	if value := strings.TrimSpace(startTime); value != "" {
		return value
	}
	return creationTime
}

// findLegacyLabelTask 匹配标签未解析时遗留的旧任务，避免修复后同主机同计划重复建任务
func findLegacyLabelTask(tx *sql.Tx, hostID, name, taskType, scheduleCron string) (domain.BackupTask, bool) {
	if name == "" || taskType == "" || scheduleCron == "" {
		return domain.BackupTask{}, false
	}
	return queryTask(tx, `SELECT id,host_id,host_name,name,COALESCE(task_key,''),type,COALESCE(tool,''),
		COALESCE(schedule,''),COALESCE(schedule_cron,''),schedule_locked,status,COALESCE(last_execution_time,''),config,'' 
		FROM tasks WHERE host_id=? AND type=? AND COALESCE(schedule_cron,'')=? AND instr(name, ?)=1 LIMIT 1`,
		hostID, taskType, scheduleCron, name+" key=")
}
