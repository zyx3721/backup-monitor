package update

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// FailureBackoff 是同一版本自动更新失败后的退避时长，期间不再自动重试
const FailureBackoff = 30 * time.Minute

type failureState struct {
	Version  string    `json:"version"`
	FailedAt time.Time `json:"failedAt"`
}

// RecordFailure 记录目标版本的更新失败时间，供后续自动检查退避
func RecordFailure(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return writeFailure(failureStatePath(executable), version, time.Now())
}

// FailureBlocked 判断目标版本是否仍在失败退避期内
func FailureBlocked(version string, now time.Time) bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	return failureBlocked(failureStatePath(executable), version, now)
}

// ClearFailure 删除失败记录，更新流程重新开始时调用
func ClearFailure() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(failureStatePath(executable))
}

func failureStatePath(executable string) string {
	return executable + ".update-state"
}

func writeFailure(path, version string, at time.Time) error {
	payload, err := json.Marshal(failureState{Version: version, FailedAt: at})
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func failureBlocked(path, version string, now time.Time) bool {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var state failureState
	if err := json.Unmarshal(payload, &state); err != nil {
		return false
	}
	if state.Version != strings.TrimSpace(version) || state.FailedAt.IsZero() {
		return false
	}
	return now.Sub(state.FailedAt) < FailureBackoff
}
