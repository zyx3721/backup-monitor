// Package update 实现 Agent 自更新、版本比较与重启。
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/service"
)

// Options 是执行自更新所需的信息。
type Options struct {
	Client        *api.Client
	ServerURL     string
	Token         string
	Version       string
	SHA256        string
	DownloadURL   string
	ServiceMode   bool
	Logger        *log.Logger
	ExitOnRestart bool
}

// DownloadPath 拼接当前主机对应的二进制下载地址。
func DownloadPath(platform, arch string, osBuild int) string {
	query := url.Values{}
	query.Set("platform", platform)
	if strings.TrimSpace(arch) != "" {
		query.Set("arch", arch)
	}
	if osBuild > 0 {
		query.Set("osBuild", strconv.Itoa(osBuild))
	}
	return "/api/agent/download?" + query.Encode()
}

// Apply 下载并校验新二进制，替换当前文件后重启进程。
func Apply(ctx context.Context, options Options) error {
	expected := strings.ToLower(strings.TrimSpace(options.SHA256))
	if expected == "" {
		return fmt.Errorf("更新校验失败：缺少 sha256")
	}
	if strings.TrimSpace(options.DownloadURL) == "" {
		return fmt.Errorf("更新失败：缺少下载地址")
	}

	client := options.Client
	if client == nil {
		created, err := api.New(options.ServerURL, options.Token, 10*time.Minute)
		if err != nil {
			return fmt.Errorf("创建更新客户端失败：%w", err)
		}
		client = created
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败：%w", err)
	}
	temporary := executable + ".new"
	backup := executable + ".bak"
	if err := os.Remove(temporary); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理旧更新文件失败：%w", err)
	}

	if options.Logger != nil {
		options.Logger.Printf("开始下载 Agent 更新：版本 %s", options.Version)
	}
	result, err := client.Download(ctx, options.DownloadURL, temporary)
	if err != nil {
		return fmt.Errorf("下载 Agent 更新失败：%w", err)
	}
	defer os.Remove(temporary)

	if header := strings.TrimSpace(result.Header); header != "" && !strings.EqualFold(header, expected) {
		return fmt.Errorf("下载校验头不一致：实际 %s，期望 %s", header, expected)
	}
	actual, err := fileSHA256(temporary)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("更新文件校验失败：实际 %s，期望 %s", actual, expected)
	}
	if err := os.Chmod(temporary, 0o755); err != nil {
		return fmt.Errorf("设置更新文件权限失败：%w", err)
	}

	if err := clearBackup(backup); err != nil {
		return err
	}
	if err := os.Rename(executable, backup); err != nil {
		return fmt.Errorf("备份当前程序失败：%w", err)
	}
	if err := os.Rename(temporary, executable); err != nil {
		if restoreErr := restoreExecutable(executable, backup); restoreErr != nil {
			return fmt.Errorf("替换当前程序失败：%w（恢复旧版失败：%v，备份保留在 %s）", err, restoreErr, backup)
		}
		return fmt.Errorf("替换当前程序失败：%w", err)
	}
	info, statErr := os.Stat(executable)
	if statErr != nil || info.Size() == 0 {
		if restoreErr := restoreExecutable(executable, backup); restoreErr != nil {
			return fmt.Errorf("更新文件校验失败（恢复旧版失败：%v，备份保留在 %s）", restoreErr, backup)
		}
		if statErr != nil {
			return fmt.Errorf("更新文件不可用：%w", statErr)
		}
		return fmt.Errorf("更新文件为空")
	}
	if options.Logger != nil {
		options.Logger.Printf("Agent 程序已更新：版本 %s", options.Version)
	}

	if err := restartAt(executable, options.ServiceMode, options.Logger); err != nil {
		if restoreErr := restoreExecutable(executable, backup); restoreErr != nil {
			return fmt.Errorf("重启更新后的 Agent 失败：%w（恢复旧版失败：%v，备份保留在 %s）", err, restoreErr, backup)
		}
		if startErr := restartAt(executable, options.ServiceMode, options.Logger); startErr != nil {
			return fmt.Errorf("重启更新后的 Agent 失败：%w（旧版已恢复，但启动失败：%v）", err, startErr)
		}
		return fmt.Errorf("重启更新后的 Agent 失败，已恢复旧版：%w", err)
	}
	ClearFailure()
	if options.ExitOnRestart {
		os.Exit(0)
	}
	return nil
}

// restartAt 使用替换前记录的可执行文件路径重启；服务未安装时回退为直接启动新进程。
func restartAt(executable string, serviceMode bool, logger *log.Logger) error {
	if serviceMode {
		err := service.Restart()
		if err == nil {
			return nil
		}
		if !errors.Is(err, service.ErrNotInstalled) {
			return err
		}
		if logger != nil {
			logger.Printf("未安装系统服务，改为直接重启进程")
		}
	}
	return spawnRestart(executable, os.Args[1:])
}

// clearBackup 清理上一次更新的备份文件；无法删除时改名让位，避免阻塞本次更新。
func clearBackup(path string) error {
	if err := os.Remove(path); err == nil || os.IsNotExist(err) {
		return nil
	}
	stale := fmt.Sprintf("%s.stale-%d", path, time.Now().UnixNano())
	if err := os.Rename(path, stale); err != nil {
		return fmt.Errorf("清理旧备份失败：%w", err)
	}
	return nil
}

// restoreExecutable 用备份恢复主程序，优先原子改名，失败时复制覆盖。
func restoreExecutable(executable, backup string) error {
	if err := os.Rename(backup, executable); err == nil {
		return nil
	}
	if err := os.Remove(executable); err == nil || os.IsNotExist(err) {
		if err := os.Rename(backup, executable); err == nil {
			return nil
		}
	}
	if err := copyFile(backup, executable); err != nil {
		return err
	}
	_ = os.Remove(backup)
	return nil
}

// copyFile 将备份内容复制到目标路径。
func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return err
	}
	return target.Close()
}

// CleanupTempFiles 删除上一轮更新遗留的临时文件，失败不影响启动
func CleanupTempFiles() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(executable + ".new")
}

// CleanupBackup 删除旧版本备份，返回是否实际删除
func CleanupBackup() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	return removeFile(executable + ".bak")
}

func removeFile(path string) bool {
	return os.Remove(path) == nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open update file: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash update file: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
