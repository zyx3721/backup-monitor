package security

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func IsBcryptHash(stored string) bool {
	s := strings.TrimSpace(stored)
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

// VerifyPassword 校验明文密码与 bcrypt 哈希是否匹配
func VerifyPassword(stored, plain string) bool {
	if strings.TrimSpace(stored) == "" {
		return false
	}
	if !IsBcryptHash(stored) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
}
