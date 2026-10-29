// Package middleware 提供根路由使用的全局 HTTP 中间件：跨域、JWT 认证与权限校验。
package middleware

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/router/common"
	"backup-monitor-backend/router/settings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthClaims JWT 载荷结构
type AuthClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Source   string `json:"source"`
	jwt.RegisteredClaims
}

// CORS 返回跨域处理中间件，来源列表含 "*" 时放行任意来源
func CORS(origins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]struct{}{}
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			break
		}
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Auth 返回 Bearer JWT 认证中间件，验签后校验服务端会话记录仍有效，通过后把会话用户写入请求上下文
func Auth(jwtSecret string, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if auth == "" {
				common.WriteError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			tokenParts := strings.SplitN(auth, " ", 2)
			if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
				common.WriteError(w, http.StatusUnauthorized, "invalid Authorization header")
				return
			}

			tokenStr := strings.TrimSpace(tokenParts[1])
			claims := &AuthClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				common.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			if strings.TrimSpace(claims.ID) == "" {
				common.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			var expiresAt string
			if e := db.QueryRowContext(r.Context(), "SELECT expires_at FROM sessions WHERE jti=?", claims.ID).Scan(&expiresAt); e != nil {
				common.WriteError(w, http.StatusUnauthorized, "session expired or revoked")
				return
			}
			expires, parseErr := appcommon.ParseTimestamp(expiresAt)
			if parseErr != nil || expires.Before(time.Now()) {
				common.WriteError(w, http.StatusUnauthorized, "session expired or revoked")
				return
			}

			user := domain.SessionUser{ID: claims.UserID, Username: claims.Username, Source: claims.Source}
			ctx := common.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Permission 返回校验单一权限的中间件，无权限时返回 403
func Permission(db *sql.DB, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := common.CurrentUser(r.Context())
			if err != nil {
				common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
				return
			}
			if settings.HasPermission(db, user, permission) {
				next.ServeHTTP(w, r)
				return
			}
			common.WriteError(w, http.StatusForbidden, "permission denied")
		})
	}
}
