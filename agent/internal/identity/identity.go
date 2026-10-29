// Package identity 解析稳定的主机标识，缺失时生成 UUID 并持久化。
package identity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fallbackFileName = ".agent-id"

// Get 返回稳定的节点标识，优先使用系统机器标识，其次使用同目录持久化文件。
func Get() (string, error) {
	if value := normalize(systemID()); value != "" {
		return value, nil
	}

	path, err := fallbackPath()
	if err != nil {
		return newUUID(), err
	}
	if data, readErr := os.ReadFile(path); readErr == nil {
		if value := normalize(string(data)); value != "" {
			return value, nil
		}
	}

	value := newUUID()
	if writeErr := os.WriteFile(path, []byte(value+"\n"), 0o600); writeErr != nil {
		return value, fmt.Errorf("persist agent id to %s: %w", path, writeErr)
	}
	return value, nil
}

func fallbackPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(executable), fallbackFileName), nil
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func newUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", os.Getpid())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(buf[0:4]),
		hex.EncodeToString(buf[4:6]),
		hex.EncodeToString(buf[6:8]),
		hex.EncodeToString(buf[8:10]),
		hex.EncodeToString(buf[10:16]),
	)
}
