package repository

import (
	"context"
	"time"

	appcommon "backup-monitor-backend/internal/common"
)

// SessionRecord 服务端会话记录：JTI 与签发的 JWT 一一对应。
type SessionRecord struct {
	JTI       string
	UserID    int64
	Username  string
	Source    string
	CreatedAt string
	ExpiresAt string
}

// CreateSession 写入一条会话记录，登录签发令牌时调用。
func (r *DomainRepository) CreateSession(ctx context.Context, session SessionRecord) error {
	_, e := r.ExecContext(ctx, `INSERT INTO sessions(jti,user_id,username,source,created_at,expires_at) VALUES(?,?,?,?,?,?)`,
		session.JTI, session.UserID, session.Username, session.Source, session.CreatedAt, session.ExpiresAt)
	return e
}

// FindSession 按 JTI 查询会话记录，认证中间件校验令牌是否仍有效时调用。
func (r *DomainRepository) FindSession(ctx context.Context, jti string) (SessionRecord, error) {
	var s SessionRecord
	e := r.QueryRowContext(ctx, `SELECT jti,user_id,username,source,created_at,expires_at FROM sessions WHERE jti=? LIMIT 1`, jti).Scan(
		&s.JTI, &s.UserID, &s.Username, &s.Source, &s.CreatedAt, &s.ExpiresAt)
	if e != nil {
		return SessionRecord{}, e
	}
	return s, nil
}

// DeleteSession 按 JTI 删除会话记录，用户注销时调用使令牌立即失效。
func (r *DomainRepository) DeleteSession(ctx context.Context, jti string) error {
	_, e := r.ExecContext(ctx, `DELETE FROM sessions WHERE jti=?`, jti)
	return e
}

// DeleteExpiredSessions 删除指定时间之前过期的会话记录，登录签发时惰性调用。
func (r *DomainRepository) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, e := r.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, appcommon.FormatTimestamp(now))
	return e
}
