// Package repository 提供备份监控域的 SQLite 持久化实现。
package repository

import (
	"database/sql"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
)

// agentBinariesTableSQL 托管槽位表：平台、架构与通道共同确定一份包。
const agentBinariesTableSQL = `CREATE TABLE IF NOT EXISTS agent_binaries (
			platform TEXT NOT NULL,
			arch TEXT NOT NULL,
			variant TEXT NOT NULL DEFAULT 'standard',
			go_version TEXT NOT NULL DEFAULT '',
			version TEXT NOT NULL,
			sha256 TEXT NOT NULL,
			size INTEGER NOT NULL,
			uploaded_at TEXT NOT NULL,
			PRIMARY KEY (platform, arch, variant)
		)`

// EnsureBackupSchema 创建备份监控所需表、索引与默认数据（幂等）。
func EnsureBackupSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS hosts (
			id TEXT PRIMARY KEY,
			hostname TEXT NOT NULL UNIQUE,
			node_id TEXT,
			ip TEXT NOT NULL DEFAULT '',
			os TEXT DEFAULT '',
			role TEXT DEFAULT '',
			agent_version TEXT DEFAULT '',
			agent_started_at TEXT NOT NULL DEFAULT '',
			platform TEXT DEFAULT '',
			arch TEXT NOT NULL DEFAULT '',
			os_build INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'normal',
			last_report_time TEXT DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_hosts_node ON hosts(node_id) WHERE node_id IS NOT NULL AND node_id <> ''`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL,
			host_name TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			task_key TEXT,
			type TEXT NOT NULL DEFAULT 'shell',
			tool TEXT DEFAULT '',
			schedule TEXT DEFAULT '',
			schedule_cron TEXT DEFAULT '',
			schedule_locked INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'no_data',
			last_execution_time TEXT DEFAULT '',
			config TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			UNIQUE(host_id, name)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_key ON tasks(host_id, task_key) WHERE task_key IS NOT NULL AND task_key <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_host ON tasks(host_id)`,
		`CREATE TABLE IF NOT EXISTS executions (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			session_id TEXT,
			job_id TEXT,
			oib_id TEXT,
			state TEXT NOT NULL,
			trigger TEXT,
			creation_time TEXT NOT NULL,
			start_time TEXT,
			end_time TEXT,
			processed_size TEXT,
			transferred_size TEXT,
			artifact_path TEXT,
			exit_code INTEGER,
			attempt INTEGER,
			max_attempts INTEGER,
			is_last_attempt INTEGER,
			message TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_task_time ON executions(task_id, creation_time)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_session ON executions(task_id, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_time ON executions(creation_time)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			host_id TEXT NOT NULL,
			task_id TEXT,
			host_name TEXT NOT NULL DEFAULT '',
			task_name TEXT,
			type TEXT NOT NULL,
			level TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			resolved_at TEXT,
			message TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_scope ON alerts(type, task_id, status)`,
		`CREATE TABLE IF NOT EXISTS agent_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			hostname TEXT NOT NULL,
			description TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			priority INTEGER NOT NULL DEFAULT 0,
			config TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS agent_tokens (
			id TEXT PRIMARY KEY,
			token TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			hostname TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			last_seen_at TEXT,
			created_at TEXT NOT NULL
		)`,
		agentBinariesTableSQL,
		`CREATE TABLE IF NOT EXISTS notify_channels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			event_type TEXT NOT NULL DEFAULT 'backup_failed',
			level TEXT NOT NULL DEFAULT 'all',
			remain_days INTEGER NOT NULL DEFAULT 0,
			trigger_group TEXT NOT NULL DEFAULT '默认分组',
			remark TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			recover INTEGER NOT NULL DEFAULT 0,
			config TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			channel TEXT NOT NULL,
			channel_id TEXT,
			channel_name TEXT,
			type TEXT NOT NULL,
			level TEXT,
			recipients TEXT,
			subject TEXT,
			status TEXT NOT NULL,
			error TEXT,
			host_name TEXT,
			task_name TEXT,
			message TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return seedBackupDefaults(db)
}

// seedBackupDefaults 写入默认采集配置，已有数据时不覆盖。
func seedBackupDefaults(db *sql.DB) error {
	now := appcommon.NowTimestamp()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_profiles`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		configJSON, err := marshalJSON(domain.DefaultProfileConfig())
		if err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO agent_profiles(id,name,hostname,description,enabled,priority,config,created_at,updated_at)
			VALUES(?,?,?,?,1,0,?,?,?)`, newID(), "默认采集配置", "*", "适用于所有主机的默认配置", configJSON, now, now); err != nil {
			return err
		}
	}
	return nil
}
