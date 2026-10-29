package common

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func SetupSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA case_sensitive_like = 0;",
		"PRAGMA encoding = \"UTF-8\";",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}
	return nil
}
