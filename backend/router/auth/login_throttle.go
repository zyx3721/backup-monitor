package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/router/settings"
)

// loginFailureRetentionHours 登录失败记录保留时长，超期记录在写入时顺带清理
const loginFailureRetentionHours = 24

// loginLockedError 登录锁定错误，Minutes 为按最近一次密码失败时间推算的剩余等待分钟数
type loginLockedError struct {
	Minutes int64
}

func (e loginLockedError) Error() string {
	return fmt.Sprintf("密码连续错误次数过多，请于 %d 分钟后再试", e.Minutes)
}

// ensureLoginAllowed 判定账号密码失败次数达到阈值且仍在锁定时长内时返回锁定错误，锁定到期后清空该账号全部历史失败记录重新计数。
// admin 不受限；企微扫码登录无密码环节，不经过此判定
func (a *Router) ensureLoginAllowed(ctx context.Context, username string) error {
	if strings.EqualFold(strings.TrimSpace(username), "admin") {
		return nil
	}
	config := settings.LoadSystemBaseConfig(ctx, a.db)
	var failures int64
	var lastFailedAt string
	err := a.db.QueryRowContext(ctx,
		"SELECT COUNT(*), COALESCE(MAX(failed_at),'') FROM login_failure_log WHERE username=?",
		strings.ToLower(strings.TrimSpace(username)),
	).Scan(&failures, &lastFailedAt)
	if err != nil {
		return err
	}
	if failures < int64(config.LoginMaxFailures) || lastFailedAt == "" {
		return nil
	}
	lastFailed, parseErr := appcommon.ParseTimestamp(lastFailedAt)
	if parseErr != nil {
		return nil
	}
	remaining := lastFailed.Add(time.Duration(config.LoginLockoutMinutes) * time.Minute).Sub(time.Now())
	if remaining <= 0 {
		_, err = a.db.ExecContext(ctx,
			"DELETE FROM login_failure_log WHERE username=?",
			strings.ToLower(strings.TrimSpace(username)))
		return err
	}
	minutes := int64((remaining + time.Minute - 1) / time.Minute)
	return loginLockedError{Minutes: minutes}
}

// recordLoginFailure 记录一次密码校验失败（用户名统一小写存储），并顺带清理超过
// 保留时长的旧记录，防止失败表无限增长
func (a *Router) recordLoginFailure(ctx context.Context, username string) error {
	name := strings.ToLower(strings.TrimSpace(username))
	if name == "" {
		return nil
	}
	now := time.Now()
	if _, err := a.db.ExecContext(ctx,
		"INSERT INTO login_failure_log(username,failed_at) VALUES(?,?)", name, appcommon.FormatTimestamp(now)); err != nil {
		return err
	}
	cutoff := appcommon.FormatTimestamp(now.Add(-time.Duration(loginFailureRetentionHours) * time.Hour))
	_, err := a.db.ExecContext(ctx,
		"DELETE FROM login_failure_log WHERE failed_at < ?", cutoff)
	return err
}

// clearLoginFailures 登录成功或找回密码重置成功后清除该账号的失败计数，返回实际清除的记录数
func (a *Router) clearLoginFailures(ctx context.Context, username string) (int64, error) {
	result, err := a.db.ExecContext(ctx,
		"DELETE FROM login_failure_log WHERE username=?", strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
