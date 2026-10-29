package repository

import (
	"context"
	"database/sql"
)

// Executor 是仓储与服务所需的最小数据库契约，
// 使二者既可作用于 *sql.DB 也可作用于 *sql.Tx。
type Executor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
