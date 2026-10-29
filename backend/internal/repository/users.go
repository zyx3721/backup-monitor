package repository

import (
	"context"
	"database/sql"
)

// UserRepository 用户域的持久化契约
type UserRepository interface {
	Repository
	UsernameExists(context.Context, string, int64) (bool, error)
	FindUsername(context.Context, int64) (string, error)
	FindAdminID(context.Context) (int64, error)
	DeleteUser(context.Context, *sql.Tx, int64) error
}

func (r *DomainRepository) UsernameExists(ctx context.Context, username string, excludeID int64) (bool, error) {
	var count int64
	err := r.QueryRowContext(ctx, `SELECT COUNT(id) FROM users WHERE LOWER(username) = LOWER(?) AND id <> ?`, username, excludeID).Scan(&count)
	return count > 0, err
}

func (r *DomainRepository) FindUsername(ctx context.Context, id int64) (string, error) {
	var username string
	return username, r.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, id).Scan(&username)
}

func (r *DomainRepository) FindAdminID(ctx context.Context) (int64, error) {
	var id int64
	err := r.QueryRowContext(ctx, `SELECT id FROM users WHERE LOWER(username) = 'admin' LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 1, nil
	}
	return id, err
}

func (r *DomainRepository) DeleteUser(ctx context.Context, tx *sql.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}
