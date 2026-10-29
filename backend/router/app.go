package router

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"backup-monitor-backend/config"
	"backup-monitor-backend/internal/buildinfo"
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/auth"
	backupr "backup-monitor-backend/router/backup"
	"backup-monitor-backend/router/common"
	"backup-monitor-backend/router/settings"
	"backup-monitor-backend/router/system"

	_ "modernc.org/sqlite"
)

// Config 是根包装配使用的配置别名
type Config = config.Config

type App struct {
	db           *sql.DB
	audit        *service.AuditService
	userWorkflow *service.UserWorkflow
	authWorkflow *service.AuthWorkflow
	backupSvc    *service.BackupService
	backupHub    *service.AgentHub
	cfg          Config
	authR        *auth.Router
	settingsR    *settings.Router
	systemR      *system.Router
	backupR      *backupr.Router
}

// resolveJWTSecret 返回生效的 JWT 签名密钥：显式配置优先；未配置时生成随机密钥
// 并持久化到数据库，使已签发会话的生命周期跟随数据库——删除数据库重建后旧令牌全部失效。
func resolveJWTSecret(db *sql.DB, configured string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		return configured, nil
	}
	var secret string
	err := db.QueryRow("SELECT jwt_secret FROM system_secrets WHERE id=1").Scan(&secret)
	if err == nil && strings.TrimSpace(secret) != "" {
		return secret, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret = hex.EncodeToString(raw)
	now := appcommon.NowTimestamp()
	if _, err := db.Exec("INSERT INTO system_secrets(id,jwt_secret,updated_at) VALUES(1,?,?) ON CONFLICT(id) DO UPDATE SET jwt_secret=excluded.jwt_secret,updated_at=excluded.updated_at", secret, now); err != nil {
		return "", err
	}
	return secret, nil
}

func (a *App) setDB(db *sql.DB) {
	a.db = db
	store := repository.NewStore(db)
	a.audit = service.NewAuditService(db)
	a.userWorkflow = service.NewUserWorkflow(repository.NewUserRepository(store), a.audit)
	a.authWorkflow = service.NewAuthWorkflow(repository.NewAuthRepository(store), a.cfg.JWTSecret, func(ctx context.Context, username, password string) error {
		err := settings.AuthenticateLDAPUser(a.db, a.cfg.JWTSecret, username, password)
		if settings.IsLDAPInvalidCredentialsError(err) {
			return service.ErrInvalidCredentials
		}
		return err
	}, a.cfg.SessionTTL)
	a.backupSvc = service.NewBackupService(repository.NewBackupRepository(db))
	a.backupSvc.SetDataDir(dataDirFor(a.cfg.DBPath))
	a.backupHub = service.NewAgentHub(repository.NewBackupRepository(db), a.backupSvc)
	a.backupSvc.SetHub(a.backupHub)
	a.backupR = backupr.New(a.backupSvc, a.backupHub, a.audit, a.cfg)
}

// dataDirFor 返回数据库同级的运行时数据目录，Agent 二进制存放于其 agents 子目录。
func dataDirFor(dbPath string) string {
	if strings.TrimSpace(dbPath) == "" || strings.EqualFold(dbPath, ":memory:") {
		return "data"
	}
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return "data"
	}
	return dir
}

// RunWithConfig 使用指定配置启动后端服务，供命令行参数覆盖场景复用完整启动流程
func RunWithConfig(cfg config.Config) {
	if err := system.EnsureDatabaseInitialized(cfg); err != nil {
		log.Fatalf("Database initialization failed: %s", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("Open database failed: %s", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := common.SetupSQLite(db); err != nil {
		log.Fatalf("Initialize SQLite settings failed: %s", err)
	}

	if err := system.EnsureRuntimeSchema(db); err != nil {
		log.Fatalf("Initialize database schema failed: %s", err)
	}

	if err := repository.EnsureBackupSchema(db); err != nil {
		log.Fatalf("Initialize backup monitor schema failed: %s", err)
	}

	jwtSecret, err := resolveJWTSecret(db, cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Resolve JWT secret failed: %s", err)
	}
	cfg.JWTSecret = jwtSecret

	app := &App{cfg: cfg}
	app.setDB(db)
	router := app.routes()
	app.backupSvc.StartScheduler(context.Background())

	addr := cfg.ServerAddr
	log.Printf("Backup Monitor Go API %s started, listening on %s", buildinfo.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
