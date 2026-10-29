package common

import (
	"context"
	"errors"

	"backup-monitor-backend/internal/domain"
)

type userContextKey string

const userKey userContextKey = "backup_monitor_user"

// WithUser 将已认证用户写入请求上下文，供认证中间件与处理器共享
func WithUser(ctx context.Context, user domain.SessionUser) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func CurrentUser(ctx context.Context) (domain.SessionUser, error) {
	raw := ctx.Value(userKey)
	if raw == nil {
		return domain.SessionUser{}, errors.New("missing user")
	}
	user, ok := raw.(domain.SessionUser)
	if !ok {
		return domain.SessionUser{}, errors.New("invalid user context")
	}
	return user, nil
}
