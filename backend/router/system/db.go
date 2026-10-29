package system

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"backup-monitor-backend/config"
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/security"
	"backup-monitor-backend/router/common"
)

var databaseBootstrapSQL = []string{
	`CREATE TABLE audit_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, date TEXT NOT NULL DEFAULT '', sql, authuser, ip, module TEXT DEFAULT '', action TEXT DEFAULT '', target TEXT DEFAULT '', detail TEXT DEFAULT '', result TEXT DEFAULT 'success')`,
	`CREATE INDEX IF NOT EXISTS idx_audit_logs_date ON audit_logs(date)`,
	`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL, userdesc TEXT NOT NULL DEFAULT '', pass TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, wecom_userid TEXT UNIQUE, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '', last_login_at TEXT NOT NULL DEFAULT '')`,
	`CREATE TABLE system_secrets (id INTEGER PRIMARY KEY CHECK (id = 1), jwt_secret TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT '')`,
}

func EnsureDatabaseInitialized(cfg config.Config) error {
	dbPath := strings.TrimSpace(cfg.DBPath)
	if dbPath == "" || strings.EqualFold(dbPath, ":memory:") {
		return nil
	}

	if _, err := os.Stat(dbPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}

	if len(databaseBootstrapSQL) == 0 {
		return fmt.Errorf("database bootstrap sql is empty")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := common.SetupSQLite(db); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i, statement := range databaseBootstrapSQL {
		if _, err := tx.Exec(statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute bootstrap sql #%d failed: %w", i+1, err)
		}
	}

	adminPass, err := security.HashPassword("admin123")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("hash default admin password failed: %w", err)
	}
	now := appcommon.NowTimestamp()
	if _, err := tx.Exec(`INSERT INTO users (username, userdesc, pass, email, disabled, created_at, updated_at) VALUES (?, ?, ?, '', 0, ?, ?)`,
		"admin", "admin", adminPass, now, now); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert default admin failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Database initialized: db=%s, statements=%d", dbPath, len(databaseBootstrapSQL)+1)
	return nil
}
