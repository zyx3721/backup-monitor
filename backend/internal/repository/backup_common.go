package repository

import (
	"database/sql"
	"encoding/json"

	appcommon "backup-monitor-backend/internal/common"

	"github.com/google/uuid"
)

// BackupRepository 封装备份监控域的持久化访问。
type BackupRepository struct {
	db *sql.DB
}

// NewBackupRepository 创建备份监控仓储。
func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// DB 暴露底层连接，供需要事务或复杂查询的服务使用。
func (r *BackupRepository) DB() *sql.DB {
	return r.db
}

func newID() string {
	return uuid.NewString()
}

func marshalJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// nowISO 返回统一格式的当前时间。
func nowISO() string {
	return appcommon.NowTimestamp()
}

func intPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	out := int(value.Int64)
	return &out
}

func boolPtr(value sql.NullInt64) *bool {
	if !value.Valid {
		return nil
	}
	out := value.Int64 == 1
	return &out
}
