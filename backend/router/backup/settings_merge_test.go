package backup

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/service"
	settingsrouter "backup-monitor-backend/router/settings"
)

func newSettingsTestRouter(t *testing.T) (*Router, *service.BackupService) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "backup-monitor.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := settingsrouter.EnsureResourceSchema(db); err != nil {
		t.Fatalf("ensure settings schema: %v", err)
	}
	if err := repository.EnsureBackupSchema(db); err != nil {
		t.Fatalf("ensure backup schema: %v", err)
	}
	svc := service.NewBackupService(repository.NewBackupRepository(db))
	return &Router{svc: svc}, svc
}

func TestHandleSaveSettingsMergesPartialBody(t *testing.T) {
	router, svc := newSettingsTestRouter(t)
	current := svc.Settings()
	current.ScheduleMatchWindowMinutes = 90
	current.Agent.ReportIntervalSec = 120
	current.DedupeAlerts = false
	if err := svc.SaveSettings(current); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	body, err := json.Marshal(map[string]any{"section": "thresholds", "reportTimeoutMinutes": 5})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/backup/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.handleSaveSettings(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	got := svc.Settings()
	if got.ReportTimeoutMinutes != 5 {
		t.Fatalf("report timeout = %d, want 5", got.ReportTimeoutMinutes)
	}
	if got.ScheduleMatchWindowMinutes != 90 || got.Agent.ReportIntervalSec != 120 || got.DedupeAlerts {
		t.Fatalf("other fields were reset: %+v", got)
	}
}
