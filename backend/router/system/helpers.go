package system

import (
	"backup-monitor-backend/internal/repository"
)

func (a *Router) fetchRows(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return repository.RowsToMaps(rows)
}
