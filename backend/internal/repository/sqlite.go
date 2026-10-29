package repository

import (
	"context"
	"database/sql"
)

// DomainRepository 是各领域仓储接口共享的 SQLite 实现，
// 不包含 HTTP 或业务策略代码。
type DomainRepository struct{ store *Store }

func NewDomainRepository(store *Store) *DomainRepository { return &DomainRepository{store: store} }
func (r *DomainRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return r.store.Query(query, args...)
}
func (r *DomainRepository) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.store.QueryContext(ctx, query, args...)
}
func (r *DomainRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	return r.store.QueryRow(query, args...)
}
func (r *DomainRepository) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.store.QueryRowContext(ctx, query, args...)
}
func (r *DomainRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	return r.store.Exec(query, args...)
}
func (r *DomainRepository) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.store.ExecContext(ctx, query, args...)
}
func (r *DomainRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.store.BeginTx(ctx, opts)
}

func NewAuthRepository(s *Store) AuthRepository { return NewDomainRepository(s) }
func NewUserRepository(s *Store) UserRepository { return NewDomainRepository(s) }
