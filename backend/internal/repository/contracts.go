package repository

import (
	"context"
	"database/sql"
	"time"
)

// AuthRepository 是认证域的持久化边界，只暴露 SQL 操作，便于后续补充专属查询而不影响 HTTP 层。
type AuthRepository interface {
	Repository
	FindAuthUser(context.Context, string) (AuthUserRecord, error)
	FindAuthUserByWecom(context.Context, string) (AuthUserRecord, error)
	UserDescription(context.Context, int64) (string, error)
	UpdateAuthPassword(context.Context, int64, string) error
	CreateSession(context.Context, SessionRecord) error
	FindSession(context.Context, string) (SessionRecord, error)
	DeleteSession(context.Context, string) error
	DeleteExpiredSessions(context.Context, time.Time) error
}

// Repository 是各域服务共用的持久化契约。
type Repository interface {
	Executor
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}
