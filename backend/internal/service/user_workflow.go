package service

import (
	"context"
	"errors"
	"strings"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/repository"
	"backup-monitor-backend/internal/security"
)

var ErrUsernameExists = errors.New("username already exists")
var ErrCannotRemoveAdmin = errors.New("cannot remove default admin user")

type UserActor struct {
	Username string
	IP       string
}

type UserWorkflow struct {
	repo  repository.UserRepository
	audit *AuditService
}

func NewUserWorkflow(repo repository.UserRepository, audit *AuditService) *UserWorkflow {
	return &UserWorkflow{repo: repo, audit: audit}
}

func (s *UserWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.UserPayload) error {
	username, err := normalizeUser(req.Username)
	if err != nil {
		return err
	}
	exists, err := s.repo.UsernameExists(ctx, username, id)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameExists
	}
	updatePassword := strings.TrimSpace(req.Password) != ""
	query := `UPDATE users SET username = ?, userdesc = ?, email = ?, disabled = ?, updated_at = ` + appcommon.SQLTimestampNow
	args := []interface{}{username, req.UserDesc, strings.TrimSpace(req.Email), boolToInt(req.Disabled)}
	if updatePassword {
		password, err := security.HashPassword(req.Password)
		if err != nil {
			return err
		}
		query += `, pass = ?`
		args = append(args, password)
	}
	query += ` WHERE id = ?`
	args = append(args, id)
	_, err = s.repo.ExecContext(ctx, query, args...)
	return err
}

func (s *UserWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	username, err := s.repo.FindUsername(ctx, id)
	if err != nil {
		return err
	}
	if strings.EqualFold(username, "admin") {
		return ErrCannotRemoveAdmin
	}
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.repo.DeleteUser(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func normalizeUser(username string) (string, error) {
	username = strings.TrimSpace(username)
	return username, domain.ValidateUserUsername(username)
}
