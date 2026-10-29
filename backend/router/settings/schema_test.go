package settings_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/router/settings"
)

func openSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "backup-monitor.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	return db
}

func TestEnsureResourceSchemaSeedsBasicConfigSections(t *testing.T) {
	db := openSchemaDB(t)
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure schema second run: %v", err)
	}

	sections := map[string]map[string]any{}
	rows, err := db.Query(`SELECT name,config FROM basic_config`)
	if err != nil {
		t.Fatalf("query basic_config: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, raw string
		if err := rows.Scan(&name, &raw); err != nil {
			t.Fatalf("scan basic_config: %v", err)
		}
		sections[name] = domain.ParseJSONObject(raw)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate basic_config: %v", err)
	}

	for _, name := range []string{
		domain.BaseSectionBrand,
		domain.BaseSectionSecurity,
		domain.BaseSectionThresholds,
		domain.BaseSectionAgent,
		domain.BaseSectionAlarm,
	} {
		if _, ok := sections[name]; !ok {
			t.Fatalf("missing basic_config row %s", name)
		}
	}
	if len(sections) != 5 {
		t.Fatalf("basic_config rows = %d, want 5", len(sections))
	}
	if len(sections[domain.BaseSectionBrand]) != 5 || len(sections[domain.BaseSectionSecurity]) != 7 {
		t.Fatalf("brand/security not seeded with defaults: brand=%v security=%v",
			sections[domain.BaseSectionBrand], sections[domain.BaseSectionSecurity])
	}
	if got := settings.LoadSystemBaseConfig(t.Context(), db); got != domain.DefaultSystemBaseConfig() {
		t.Fatalf("seeded branding = %+v, want defaults %+v", got, domain.DefaultSystemBaseConfig())
	}
	if got := domain.MergeAppSettingsSections(sections); got != domain.DefaultAppSettings() {
		t.Fatalf("seeded settings = %+v, want defaults %+v", got, domain.DefaultAppSettings())
	}
	assertBasicConfigOrder(t, db)
}

func assertBasicConfigOrder(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM basic_config ORDER BY id`)
	if err != nil {
		t.Fatalf("query basic_config order: %v", err)
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan basic_config order: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate basic_config order: %v", err)
	}
	want := []string{
		domain.BaseSectionBrand,
		domain.BaseSectionSecurity,
		domain.BaseSectionThresholds,
		domain.BaseSectionAgent,
		domain.BaseSectionAlarm,
	}
	for index := range want {
		if index >= len(got) || got[index] != want[index] {
			t.Fatalf("basic_config order = %v, want %v", got, want)
		}
	}
}
