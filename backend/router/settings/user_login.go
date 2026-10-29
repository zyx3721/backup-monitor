package settings

import (
	"context"
	"database/sql"

	appcommon "backup-monitor-backend/internal/common"
)

// RecordSettingsUserLogin 记录用户最近登录时间，供用户配置列表展示。
func RecordSettingsUserLogin(ctx context.Context, db *sql.DB, userID int64) error {
	_, err := db.ExecContext(ctx, "UPDATE users SET last_login_at="+appcommon.SQLTimestampNow+" WHERE id=?", userID)
	return err
}
