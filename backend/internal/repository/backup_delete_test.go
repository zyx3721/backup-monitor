package repository_test

import (
	"database/sql"
	"testing"

	"backup-monitor-backend/internal/repository"
)

func TestDeleteHostCascadesTasksExecutionsAndAlerts(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)

	seedDeleteFixture(t, db)
	ok, err := repo.DeleteHost("h-delete")
	if err != nil {
		t.Fatalf("delete host: %v", err)
	}
	if !ok {
		t.Fatal("delete host returned false")
	}

	assertCount(t, db, `SELECT COUNT(*) FROM hosts WHERE id='h-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM tasks WHERE host_id='h-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM executions WHERE id='e-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM alerts WHERE id='a-delete'`, 0)
}

func TestDeleteTaskCascadesExecutionsAndAlerts(t *testing.T) {
	db := newSchemaDB(t)
	repo := repository.NewBackupRepository(db)

	seedDeleteFixture(t, db)
	ok, err := repo.DeleteTask("t-delete")
	if err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if !ok {
		t.Fatal("delete task returned false")
	}

	assertCount(t, db, `SELECT COUNT(*) FROM tasks WHERE id='t-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM executions WHERE id='e-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM alerts WHERE id='a-delete'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM hosts WHERE id='h-delete'`, 1)
}

func seedDeleteFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO hosts (id,hostname,ip,created_at) VALUES ('h-delete','host-delete','10.0.0.1','2026-01-01T00:00:00Z')`,
		`INSERT INTO tasks (id,host_id,host_name,name,type,config,created_at) VALUES ('t-delete','h-delete','host-delete','task-delete','shell','{}','2026-01-01T00:00:00Z')`,
		`INSERT INTO executions (id,task_id,state,creation_time) VALUES ('e-delete','t-delete','success','2026-01-01T00:00:00Z')`,
		`INSERT INTO alerts (id,host_id,task_id,host_name,type,level,created_at) VALUES ('a-delete','h-delete','t-delete','host-delete','backup_failed','critical','2026-01-01T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed delete fixture: %v", err)
		}
	}
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", query, got, want)
	}
}
