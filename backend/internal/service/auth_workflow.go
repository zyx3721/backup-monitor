package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/security"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrUserNotProvisioned = errors.New("user not provisioned")

type LDAPAuthenticator func(context.Context, string, string) error
type authClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Source   string `json:"source"`
	jwt.RegisteredClaims
}
type AuthWorkflow struct {
	repo       repository.AuthRepository
	secret     string
	ldap       LDAPAuthenticator
	sessionTTL time.Duration
}

func NewAuthWorkflow(repo repository.AuthRepository, secret string, ldap LDAPAuthenticator, sessionTTL time.Duration) *AuthWorkflow {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	return &AuthWorkflow{repo: repo, secret: secret, ldap: ldap, sessionTTL: sessionTTL}
}
func (s *AuthWorkflow) Login(ctx context.Context, req domain.AuthLoginRequest) (domain.AuthLoginResponse, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.Mode == "" {
		req.Mode = "local"
	}
	if req.Username == "" {
		return domain.AuthLoginResponse{}, errors.New("username is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return domain.AuthLoginResponse{}, errors.New("password is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return domain.AuthLoginResponse{}, errors.New("password is required")
	}
	isLDAP := req.Mode == "ldap"
	if isLDAP {
		if s.ldap == nil {
			return domain.AuthLoginResponse{}, errors.New("LDAP is unavailable")
		}
		if err := s.ldap(ctx, req.Username, req.Password); err != nil {
			return domain.AuthLoginResponse{}, err
		}
	}
	record, e := s.repo.FindAuthUser(ctx, req.Username)
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			if isLDAP {
				return domain.AuthLoginResponse{}, ErrUserNotProvisioned
			}
			return domain.AuthLoginResponse{}, ErrInvalidCredentials
		}
		return domain.AuthLoginResponse{}, e
	}
	if !isLDAP {
		if !security.VerifyPassword(record.Password, req.Password) {
			return domain.AuthLoginResponse{}, ErrInvalidCredentials
		}
	}
	if record.Disabled != 0 {
		return domain.AuthLoginResponse{}, ErrUserNotProvisioned
	}
	user := domain.SessionUser{ID: record.ID, Username: record.Username, Source: req.Mode}
	return s.sessionFor(ctx, user)
}

// LoginByWecom 依据企微绑定关系定位用户并签发会话令牌，未绑定或被禁用时拒绝登录
func (s *AuthWorkflow) LoginByWecom(ctx context.Context, wecomUserid string) (domain.AuthLoginResponse, error) {
	wecomUserid = strings.TrimSpace(wecomUserid)
	if wecomUserid == "" {
		return domain.AuthLoginResponse{}, errors.New("wecom userid is required")
	}
	record, e := s.repo.FindAuthUserByWecom(ctx, wecomUserid)
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return domain.AuthLoginResponse{}, ErrWecomNotBound
		}
		return domain.AuthLoginResponse{}, e
	}
	if record.Disabled != 0 {
		return domain.AuthLoginResponse{}, ErrUserNotProvisioned
	}
	user := domain.SessionUser{ID: record.ID, Username: record.Username, Source: "wecom"}
	return s.sessionFor(ctx, user)
}

// sessionFor 按配置时长签发 JWT 会话（SESSION_TTL_HOURS，默认 12 小时），
// 同时写入 sessions 会话记录并惰性清理已过期会话
func (s *AuthWorkflow) sessionFor(ctx context.Context, user domain.SessionUser) (domain.AuthLoginResponse, error) {
	now := time.Now()
	jti, e := randomSessionID()
	if e != nil {
		return domain.AuthLoginResponse{}, e
	}
	expiresAt := now.Add(s.sessionTTL)
	claims := authClaims{UserID: user.ID, Username: user.Username, Source: user.Source, RegisteredClaims: jwt.RegisteredClaims{ID: jti, ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now), Subject: user.Username}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, e := token.SignedString([]byte(s.secret))
	if e != nil {
		return domain.AuthLoginResponse{}, e
	}
	if e := s.repo.CreateSession(ctx, repository.SessionRecord{JTI: jti, UserID: user.ID, Username: user.Username, Source: user.Source, CreatedAt: appcommon.FormatTimestamp(now), ExpiresAt: appcommon.FormatTimestamp(expiresAt)}); e != nil {
		return domain.AuthLoginResponse{}, e
	}
	_ = s.repo.DeleteExpiredSessions(ctx, now)
	return domain.AuthLoginResponse{Token: signed, User: user}, nil
}

// randomSessionID 生成 16 字节十六进制随机会话标识，作为会话记录主键与 JWT jti
func randomSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (s *AuthWorkflow) Me(ctx context.Context, user domain.SessionUser) (map[string]interface{}, error) {
	desc, e := s.repo.UserDescription(ctx, user.ID)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	return map[string]interface{}{"id": user.ID, "username": user.Username, "userDesc": desc}, nil
}
