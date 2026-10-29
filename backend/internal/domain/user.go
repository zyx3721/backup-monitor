package domain

import (
	"errors"
	"strings"
)

type UserPayload struct {
	Username string `json:"username"`
	UserDesc string `json:"userDesc"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}

func ValidateUserUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}
	return nil
}
