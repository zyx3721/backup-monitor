package system

import (
	"database/sql"

	"backup-monitor-backend/router/settings"
)

// EnsureRuntimeSchema 初始化运行期所需的表结构与默认数据
func EnsureRuntimeSchema(db *sql.DB) error {
	return settings.EnsureResourceSchema(db)
}
