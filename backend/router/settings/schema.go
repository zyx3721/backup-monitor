package settings

import (
	"database/sql"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
)

func EnsureResourceSchema(db *sql.DB) error {
	statements := []string{
		"CREATE TABLE IF NOT EXISTS roles (id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', permissions TEXT NOT NULL DEFAULT '[]', builtin INTEGER NOT NULL DEFAULT 0, disabled INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS user_groups (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, description TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS user_group_members (group_id INTEGER NOT NULL, user_id INTEGER NOT NULL, PRIMARY KEY (group_id,user_id))",
		"CREATE TABLE IF NOT EXISTS user_group_roles (group_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (group_id,role_id))",
		"CREATE TABLE IF NOT EXISTS user_roles (user_id INTEGER NOT NULL, role_id INTEGER NOT NULL, PRIMARY KEY (user_id,role_id))",
		"CREATE TABLE IF NOT EXISTS reset_captchas (token_hash TEXT PRIMARY KEY, question TEXT NOT NULL, answer_hash TEXT NOT NULL, expires_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS reset_requests (token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL, code_hash TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '', expires_at TEXT NOT NULL DEFAULT '', used INTEGER NOT NULL DEFAULT 0, last_sent_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS basic_config (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, config TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS auth_config (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, title TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS pwd_recover_config (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, enabled INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}', updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS reset_send_log (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL, sent_at TEXT NOT NULL)",
		"CREATE INDEX IF NOT EXISTS idx_reset_send_log_email ON reset_send_log(email,sent_at)",
		"CREATE TABLE IF NOT EXISTS login_failure_log (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, failed_at TEXT NOT NULL)",
		"CREATE INDEX IF NOT EXISTS idx_login_failure_log_username ON login_failure_log(username, failed_at)",
		"INSERT OR IGNORE INTO auth_config(name,title,enabled,config,updated_at) VALUES ('ldap','AD/LDAP',0,'{}'," + appcommon.SQLTimestampNow + ")",
		"INSERT OR IGNORE INTO auth_config(name,title,enabled,config,updated_at) VALUES ('wecom','企业微信',0,'{}'," + appcommon.SQLTimestampNow + ")",
		"INSERT OR IGNORE INTO pwd_recover_config(name,enabled,config,updated_at) VALUES ('email',0,'{}'," + appcommon.SQLTimestampNow + ")",
		"CREATE TABLE IF NOT EXISTS system_secrets (id INTEGER PRIMARY KEY CHECK (id = 1), jwt_secret TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT '')",
		"CREATE TABLE IF NOT EXISTS sessions (jti TEXT PRIMARY KEY, user_id INTEGER NOT NULL, username TEXT NOT NULL, source TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '', expires_at TEXT NOT NULL DEFAULT '')",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at)",
	}
	statements = append(statements, builtinRoleSeedStatements()...)
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return seedBasicConfigSections(db)
}

// basicConfigSectionOrder 基础配置区块的落库顺序，与控制台「基础配置」面板展示顺序一致
var basicConfigSectionOrder = []string{
	domain.BaseSectionBrand,
	domain.BaseSectionSecurity,
	domain.BaseSectionThresholds,
	domain.BaseSectionAgent,
	domain.BaseSectionAlarm,
}

// seedBasicConfigSections 按展示顺序补齐基础配置五个区块，已有行不覆盖，并把行顺序校正为展示顺序
func seedBasicConfigSections(db *sql.DB) error {
	sections := domain.DefaultSystemBaseConfig().Sections()
	for name, section := range domain.DefaultAppSettings().Sections() {
		sections[name] = section
	}
	for _, name := range basicConfigSectionOrder {
		if _, err := db.Exec(`INSERT OR IGNORE INTO basic_config(name,config,updated_at) VALUES(?,?,`+appcommon.SQLTimestampNow+`)`,
			name, mustJSON(sections[name])); err != nil {
			return err
		}
	}
	return nil
}

// builtinRoleSeedStatements 内置角色种子语句，权限集合随权限目录自动刷新
func builtinRoleSeedStatements() []string {
	roles := []struct{ key, name, description string }{
		{key: "admin", name: "admin", description: "备份监控、审计日志、系统配置等所有操作"},
		{key: "operator", name: "operator", description: "日常业务操作与审计查看，不能修改系统配置"},
		{key: "viewer", name: "viewer", description: "只读查看备份监控、审计日志与系统配置"},
	}
	statements := make([]string, 0, len(roles))
	for _, role := range roles {
		statements = append(statements, "INSERT INTO roles(key,name,description,permissions,builtin,created_at,updated_at) VALUES ('"+role.key+"','"+role.name+"','"+role.description+"','"+mustJSON(builtinRolePermissions(role.key))+"',1,"+appcommon.SQLTimestampNow+","+appcommon.SQLTimestampNow+") ON CONFLICT(key) DO UPDATE SET name=excluded.name, description=excluded.description, permissions=excluded.permissions, builtin=1, updated_at=excluded.updated_at")
	}
	return statements
}
