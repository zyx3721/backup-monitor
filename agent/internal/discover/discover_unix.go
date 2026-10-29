//go:build linux

package discover

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"backup-monitor-agent/internal/cron"
)

// List 读取当前用户 crontab、/etc/crontab 与 /etc/cron.d 下的备份任务。
func List(ctx context.Context, filter Filter) ([]Task, error) {
	var tasks []Task
	var failures []error

	userName := currentUser()
	if text, exists, err := readUserCrontab(ctx); err != nil {
		failures = append(failures, err)
	} else if exists {
		tasks = append(tasks, ParseCrontab(text, userName, "crontab", false, filter)...)
	}

	if data, err := os.ReadFile("/etc/crontab"); err == nil {
		tasks = append(tasks, ParseCrontab(string(data), "root", "/etc/crontab", true, filter)...)
	} else if !os.IsNotExist(err) {
		failures = append(failures, fmt.Errorf("read /etc/crontab: %w", err))
	}

	paths, _ := filepath.Glob("/etc/cron.d/*")
	sort.Strings(paths)
	for _, path := range paths {
		if strings.Contains(filepath.Base(path), ".") {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("stat %s: %w", path, err))
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("read %s: %w", path, err))
			continue
		}
		tasks = append(tasks, ParseCrontab(string(data), "root", path, true, filter)...)
	}

	return dedupeTasks(tasks), errors.Join(failures...)
}

// ApplyCronByKey 按任务 key 修改 crontab 中的执行计划并返回中文结果。
func ApplyCronByKey(ctx context.Context, key, expression string) (string, error) {
	normalized := strings.Join(strings.Fields(expression), " ")
	if !cron.Validate(normalized) {
		return "计划表达式无效", fmt.Errorf("invalid cron expression %q", expression)
	}

	userName := currentUser()
	var failures []error
	if text, exists, err := readUserCrontab(ctx); err != nil {
		failures = append(failures, err)
	} else if exists {
		rewritten, count := replaceCronEntries(text, key, normalized, userName, false, defaultLabelPrefix)
		if count > 0 {
			if err := writeUserCrontab(ctx, rewritten); err != nil {
				return "更新当前用户 crontab 失败", err
			}
			return "已更新当前用户 crontab", nil
		}
	}

	if data, err := os.ReadFile("/etc/crontab"); err == nil {
		rewritten, count := replaceCronEntries(string(data), key, normalized, "root", true, defaultLabelPrefix)
		if count > 0 {
			if err := writeFileAtomic("/etc/crontab", []byte(rewritten)); err != nil {
				return "更新 /etc/crontab 失败", err
			}
			return "已更新 /etc/crontab", nil
		}
	} else if !os.IsNotExist(err) {
		failures = append(failures, fmt.Errorf("read /etc/crontab: %w", err))
	}

	paths, _ := filepath.Glob("/etc/cron.d/*")
	sort.Strings(paths)
	updated := make([]string, 0)
	for _, path := range paths {
		if strings.Contains(filepath.Base(path), ".") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("read %s: %w", path, err))
			continue
		}
		rewritten, count := replaceCronEntries(string(data), key, normalized, "root", true, defaultLabelPrefix)
		if count == 0 {
			continue
		}
		if err := writeFileAtomic(path, []byte(rewritten)); err != nil {
			return "更新 " + path + " 失败", err
		}
		updated = append(updated, path)
	}
	if len(updated) > 0 {
		return "已更新 " + strings.Join(updated, "、"), nil
	}
	if len(failures) > 0 {
		return "更新计划任务失败", errors.Join(failures...)
	}
	return "未找到匹配的计划任务", fmt.Errorf("no cron entry found for key %q", key)
}

func replaceCronEntries(text, key, newCron, defaultUser string, hasUser bool, prefix string) (string, int) {
	entries := parseCrontabEntries(text, defaultUser, hasUser, prefix)
	lines := strings.Split(text, "\n")
	count := 0
	for _, entry := range entries {
		if entry.key() != key {
			continue
		}
		if entry.index-1 < 0 || entry.index-1 >= len(lines) {
			continue
		}
		rewritten, ok := rewriteCronLine(lines[entry.index-1], newCron, prefix)
		if !ok {
			continue
		}
		lines[entry.index-1] = rewritten
		count++
	}
	return strings.Join(lines, "\n"), count
}

func readUserCrontab(ctx context.Context) (string, bool, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	command := exec.CommandContext(commandCtx, "crontab", "-l")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err == nil {
		return stdout.String(), true, nil
	}
	output := strings.ToLower(strings.TrimSpace(stderr.String() + " " + stdout.String()))
	if strings.Contains(output, "no crontab") {
		return "", false, nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return "", false, fmt.Errorf("crontab command is not available")
	}
	return "", false, fmt.Errorf("read current user crontab: %w: %s", err, strings.TrimSpace(output))
}

func writeUserCrontab(ctx context.Context, text string) error {
	commandCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	command := exec.CommandContext(commandCtx, "crontab", "-")
	command.Stdin = strings.NewReader(text)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("write current user crontab: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".backup-agent-cron-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary file for %s: %w", path, err)
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return fmt.Errorf("set mode for %s: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func currentUser() string {
	if current, err := user.Current(); err == nil && strings.TrimSpace(current.Username) != "" {
		return current.Username
	}
	for _, name := range []string{"USER", "LOGNAME", "USERNAME"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return "unknown"
}
