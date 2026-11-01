package backup

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleClearExecutions(t *testing.T) {
	router, svc := newSettingsTestRouter(t)
	db := svc.Repo().DB()
	for _, statement := range []string{
		`INSERT INTO hosts (id,hostname,ip,created_at) VALUES ('h-clear','host-clear','10.0.0.9','2026-01-01T00:00:00Z')`,
		`INSERT INTO tasks (id,host_id,host_name,name,type,config,created_at) VALUES ('t-clear','h-clear','host-clear','task-clear','shell','{}','2026-01-01T00:00:00Z')`,
		`INSERT INTO executions (id,task_id,state,creation_time) VALUES ('e-clear','t-clear','success','2026-01-01T00:00:00Z')`,
		`INSERT INTO executions (id,task_id,state,creation_time) VALUES ('e-keep','t-clear','success','2026-01-02T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed execution: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/backup/executions/clear",
		bytes.NewBufferString(`{"ids":["e-clear"]}`))
	router.handleClearExecutions(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Deleted int `json:"deleted"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", response.Deleted)
	}
	countRows(t, db, `SELECT COUNT(*) FROM executions WHERE id='e-clear'`, 0)
	countRows(t, db, `SELECT COUNT(*) FROM executions WHERE id='e-keep'`, 1)
}

func TestHandleClearAlerts(t *testing.T) {
	router, svc := newSettingsTestRouter(t)
	db := svc.Repo().DB()
	for _, statement := range []string{
		`INSERT INTO hosts (id,hostname,ip,created_at) VALUES ('h-clear','host-clear','10.0.0.9','2026-01-01T00:00:00Z')`,
		`INSERT INTO alerts (id,host_id,host_name,type,level,created_at) VALUES ('a-clear','h-clear','host-clear','backup_failed','critical','2026-01-01T00:00:00Z')`,
		`INSERT INTO alerts (id,host_id,host_name,type,level,created_at) VALUES ('a-keep','h-clear','host-clear','not_run','warning','2026-01-02T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed alert: %v", err)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/backup/alerts/clear",
		bytes.NewBufferString(`{"ids":["a-clear"]}`))
	router.handleClearAlerts(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Deleted int `json:"deleted"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", response.Deleted)
	}
	countRows(t, db, `SELECT COUNT(*) FROM alerts WHERE id='a-clear'`, 0)
	countRows(t, db, `SELECT COUNT(*) FROM alerts WHERE id='a-keep'`, 1)
}

func countRows(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}
