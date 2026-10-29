package repository_test

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/router/settings"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "backup-monitor.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure settings schema: %v", err)
	}
	if err := repository.EnsureBackupSchema(db); err != nil {
		t.Fatalf("ensure backup schema: %v", err)
	}
	return db
}

func TestBasicConfigSeedsSchedulerSeconds(t *testing.T) {
	db := newSchemaDB(t)

	var raw string
	if err := db.QueryRow(`SELECT config FROM basic_config WHERE name=?`, domain.BaseSectionThresholds).Scan(&raw); err != nil {
		t.Fatalf("load thresholds config: %v", err)
	}
	got, ok := domain.ParseJSONObject(raw)["schedulerIntervalSec"].(float64)
	if !ok || got != 30 {
		t.Fatalf("schedulerIntervalSec = %v, want 30", got)
	}
}

func TestSaveBackupSettingsStoresSchedulerSeconds(t *testing.T) {
	db := newSchemaDB(t)

	next := domain.DefaultAppSettings()
	next.SchedulerIntervalSec = 45
	if err := repository.NewBackupRepository(db).SaveBackupSettings(next); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT config FROM basic_config WHERE name=?`, domain.BaseSectionThresholds).Scan(&raw); err != nil {
		t.Fatalf("load thresholds config: %v", err)
	}
	parsed := domain.ParseJSONObject(raw)
	got, ok := parsed["schedulerIntervalSec"].(float64)
	if !ok || got != 45 {
		t.Fatalf("schedulerIntervalSec = %v, want 45", parsed["schedulerIntervalSec"])
	}
}

func TestSaveBackupSettingsWritesOneRowPerSection(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)

	next := domain.DefaultAppSettings()
	next.Agent.ReportIntervalSec = 120
	next.DedupeAlerts = false
	if err := repo.SaveBackupSettings(next); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	for _, name := range []string{domain.BaseSectionThresholds, domain.BaseSectionAgent, domain.BaseSectionAlarm} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM basic_config WHERE name=?`, name).Scan(&count); err != nil {
			t.Fatalf("count %s rows: %v", name, err)
		}
		if count != 1 {
			t.Fatalf("%s rows = %d, want 1", name, count)
		}
	}
	if _, err := db.Exec(`SELECT 1 FROM app_settings`); err == nil {
		t.Fatal("app_settings table should not exist")
	}

	got, err := repo.LoadBackupSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if got != next {
		t.Fatalf("loaded settings = %+v, want %+v", got, next)
	}
}

func TestNotificationsInsertAndList(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)

	record := domain.NotificationRecord{
		Channel: "mail", ChannelName: "运维邮件组", Type: "backup_failed", Level: "critical",
		Recipients: "ops@example.com", Subject: "备份失败", Status: "sent",
		HostName: "db-01", TaskName: "db-backup", Message: "备份执行失败",
	}
	if err := repo.InsertNotification(record); err != nil {
		t.Fatalf("insert notification: %v", err)
	}

	records, err := repo.ListNotifications(10, "")
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	got := records[0]
	if got.ChannelName != "运维邮件组" || got.Status != "sent" || got.CreatedAt == "" || got.ID == "" {
		t.Fatalf("record = %+v, want channel name, sent status, id and created time", got)
	}
}

func TestAgentBinariesSlotsSchema(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)
	rows := []repository.AgentBinaryMeta{
		{Platform: "windows", Arch: "amd64", Variant: "legacy", GoVersion: "go1.20.14", Version: "1.0.36", Sha256: "legacy-sha", Size: 100, UploadedAt: "2026-10-29T00:00:00Z"},
		{Platform: "windows", Arch: "amd64", Variant: "standard", GoVersion: "go1.26.0", Version: "1.0.36", Sha256: "standard-sha", Size: 200, UploadedAt: "2026-10-29T00:00:00Z"},
		{Platform: "windows", Arch: "arm64", Variant: "standard", GoVersion: "go1.26.0", Version: "1.0.36", Sha256: "arm-sha", Size: 300, UploadedAt: "2026-10-29T00:00:00Z"},
	}
	for _, row := range rows {
		if err := repo.SaveAgentBinaryMeta(row); err != nil {
			t.Fatalf("save %s/%s/%s: %v", row.Platform, row.Arch, row.Variant, err)
		}
	}
	legacy, err := repo.GetAgentBinaryMeta("windows", "amd64", "legacy")
	if err != nil {
		t.Fatalf("get legacy slot: %v", err)
	}
	if legacy == nil || legacy.GoVersion != "go1.20.14" {
		t.Fatalf("legacy slot = %+v, want go1.20.14", legacy)
	}
	all, err := repo.ListAgentBinaryMeta()
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("slot count = %d, want 3", len(all))
	}
}

func TestAgentTokenExists(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)
	const token = "0123456789abcdef"

	exists, err := repo.AgentTokenExists(token)
	if err != nil {
		t.Fatalf("check missing token: %v", err)
	}
	if exists {
		t.Fatal("token should not exist before creation")
	}
	if err := repo.CreateAgentToken("测试令牌", "", token); err != nil {
		t.Fatalf("create token: %v", err)
	}
	exists, err = repo.AgentTokenExists(token)
	if err != nil {
		t.Fatalf("check existing token: %v", err)
	}
	if !exists {
		t.Fatal("token should exist after creation")
	}
	if err := repo.CreateAgentToken("重复令牌", "", token); err == nil || !strings.Contains(err.Error(), "agent_tokens.token") {
		t.Fatalf("duplicate token error = %v, want unique constraint on agent_tokens.token", err)
	}
}

func TestHostAgentStartedAtRoundTrip(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)
	const startedAt = "2026-10-30T00:58:28.829824Z"
	id, err := repo.InsertHost(domain.IngestHost{
		Hostname:       "agent-start-test",
		IP:             "10.0.0.8",
		AgentVersion:   "1.0.0",
		AgentStartedAt: startedAt,
	}, "linux")
	if err != nil {
		t.Fatalf("insert host: %v", err)
	}
	host, err := repo.GetHost(id)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if host == nil || host.AgentStartedAt != startedAt {
		t.Fatalf("host agentStartedAt = %+v, want %q", host, startedAt)
	}
	hosts, err := repo.ListHosts()
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 || hosts[0].AgentStartedAt != startedAt {
		t.Fatalf("list hosts = %+v, want agentStartedAt %q", hosts, startedAt)
	}
	pushHosts, err := repo.AllHostsForPush()
	if err != nil {
		t.Fatalf("all hosts for push: %v", err)
	}
	if len(pushHosts) != 1 || pushHosts[0].AgentStartedAt != startedAt {
		t.Fatalf("push hosts = %+v, want agentStartedAt %q", pushHosts, startedAt)
	}
}
