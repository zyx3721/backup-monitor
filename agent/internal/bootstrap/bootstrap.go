// Package bootstrap 按参数、环境变量、配置文件、默认值的顺序装配本地配置。
package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"backup-monitor-agent/internal/config"
)

// Overrides 保存命令行显式传入的配置覆盖项。
type Overrides struct {
	ServerURL   *string
	Token       *string
	Hostname    *string
	Role        *string
	IntervalSec *int
	ConfigFile  *string
	LogFile     *string
	LogMaxMb    *int
	LogBackups  *int
}

// Result 是配置加载结果，Path 为实际读取的配置文件路径。
type Result struct {
	Config config.Local
	Path   string
}

// Load 加载并校验本地配置。
func Load(overrides Overrides) (Result, error) {
	local := config.DefaultLocal()
	local.Hostname, _ = os.Hostname()
	local.LogFile = defaultLogPath()

	explicit := ""
	if overrides.ConfigFile != nil {
		explicit = strings.TrimSpace(*overrides.ConfigFile)
	}
	if explicit == "" {
		explicit = strings.TrimSpace(os.Getenv("BM_CONFIG_FILE"))
	}

	path := ""
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return Result{}, fmt.Errorf("resolve config file path: %w", err)
		}
		path = abs
		if _, err := os.Stat(path); err != nil {
			return Result{}, fmt.Errorf("read config file %s: %w", path, err)
		}
	} else {
		for _, candidate := range defaultConfigPaths() {
			if candidate == "" {
				continue
			}
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}

	if path != "" {
		loaded, err := readFile(path)
		if err != nil {
			return Result{}, err
		}
		local = merge(local, loaded)
	}

	if err := applyEnv(&local); err != nil {
		return Result{}, err
	}
	applyOverrides(&local, overrides)

	if err := normalize(&local); err != nil {
		return Result{}, err
	}
	return Result{Config: local, Path: path}, nil
}

func readFile(path string) (config.Local, error) {
	file, err := os.Open(path)
	if err != nil {
		return config.Local{}, fmt.Errorf("open config file %s: %w", path, err)
	}
	defer file.Close()

	var loaded config.Local
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&loaded); err != nil {
		return config.Local{}, fmt.Errorf("decode config file %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return config.Local{}, fmt.Errorf("decode config file %s: unexpected trailing data", path)
	}
	return loaded, nil
}

func merge(base, override config.Local) config.Local {
	if override.ServerURL != "" {
		base.ServerURL = override.ServerURL
	}
	if override.Token != "" {
		base.Token = override.Token
	}
	if override.Hostname != "" {
		base.Hostname = override.Hostname
	}
	if override.Role != "" {
		base.Role = override.Role
	}
	if override.IntervalSec != 0 {
		base.IntervalSec = override.IntervalSec
	}
	if override.LogFile != "" {
		base.LogFile = override.LogFile
	}
	if override.LogMaxMb != 0 {
		base.LogMaxMb = override.LogMaxMb
	}
	if override.LogBackups != 0 {
		base.LogBackups = override.LogBackups
	}
	return base
}

func applyEnv(local *config.Local) error {
	if value, ok := os.LookupEnv("BM_SERVER_URL"); ok {
		local.ServerURL = value
	}
	if value, ok := os.LookupEnv("BM_AGENT_TOKEN"); ok {
		local.Token = value
	}
	if value, ok := os.LookupEnv("BM_HOSTNAME"); ok {
		local.Hostname = value
	}
	if value, ok := os.LookupEnv("BM_ROLE"); ok {
		local.Role = value
	}
	if value, ok := os.LookupEnv("BM_INTERVAL_SEC"); ok {
		parsed, err := parseIntEnv("BM_INTERVAL_SEC", value)
		if err != nil {
			return err
		}
		local.IntervalSec = parsed
	}
	if value, ok := os.LookupEnv("BM_LOG_FILE"); ok {
		local.LogFile = value
	}
	if value, ok := os.LookupEnv("BM_LOG_MAX_MB"); ok {
		parsed, err := parseIntEnv("BM_LOG_MAX_MB", value)
		if err != nil {
			return err
		}
		local.LogMaxMb = parsed
	}
	if value, ok := os.LookupEnv("BM_LOG_BACKUPS"); ok {
		parsed, err := parseIntEnv("BM_LOG_BACKUPS", value)
		if err != nil {
			return err
		}
		local.LogBackups = parsed
	}
	return nil
}

func parseIntEnv(name, value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q is not an integer", name, value)
	}
	return parsed, nil
}

func applyOverrides(local *config.Local, overrides Overrides) {
	if overrides.ServerURL != nil {
		local.ServerURL = *overrides.ServerURL
	}
	if overrides.Token != nil {
		local.Token = *overrides.Token
	}
	if overrides.Hostname != nil {
		local.Hostname = *overrides.Hostname
	}
	if overrides.Role != nil {
		local.Role = *overrides.Role
	}
	if overrides.IntervalSec != nil {
		local.IntervalSec = *overrides.IntervalSec
	}
	if overrides.LogFile != nil {
		local.LogFile = *overrides.LogFile
	}
	if overrides.LogMaxMb != nil {
		local.LogMaxMb = *overrides.LogMaxMb
	}
	if overrides.LogBackups != nil {
		local.LogBackups = *overrides.LogBackups
	}
}

func normalize(local *config.Local) error {
	local.ServerURL = strings.TrimRight(strings.TrimSpace(local.ServerURL), "/")
	if local.ServerURL != "" {
		parsed, err := url.Parse(local.ServerURL)
		if err != nil {
			return fmt.Errorf("invalid server URL: %w", err)
		}
		parsed.Scheme = strings.ToLower(parsed.Scheme)
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("invalid server URL: must be an absolute http or https URL")
		}
	}

	role := config.NormalizeRole(local.Role)
	if role == "" {
		return fmt.Errorf("invalid role %q: expected one of %s", local.Role, strings.Join(config.Roles(), ", "))
	}
	local.Role = role

	if local.IntervalSec < 0 {
		return fmt.Errorf("invalid intervalSec: must be zero or positive")
	}
	if local.IntervalSec > 0 && local.IntervalSec < config.MinIntervalSec {
		local.IntervalSec = config.MinIntervalSec
	}
	if local.LogMaxMb < 0 {
		return fmt.Errorf("invalid logMaxMb: must be zero or positive")
	}
	if local.LogBackups < 0 {
		return fmt.Errorf("invalid logBackups: must be zero or positive")
	}
	if local.LogMaxMb == 0 {
		local.LogMaxMb = config.DefaultLogMaxMb
	}
	if local.LogBackups == 0 {
		local.LogBackups = config.DefaultLogBackups
	}

	local.Hostname = strings.TrimSpace(local.Hostname)
	if local.Hostname == "" {
		local.Hostname = "unknown-host"
	}
	local.LogFile = strings.TrimSpace(local.LogFile)
	if local.LogFile == "" {
		local.LogFile = defaultLogPath()
	}
	return nil
}

func defaultConfigPaths() []string {
	paths := make([]string, 0, 3)
	if executable, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(executable), "backup-agent.json"))
	}
	if runtime.GOOS == "windows" {
		if programData := strings.TrimSpace(os.Getenv("ProgramData")); programData != "" {
			paths = append(paths, filepath.Join(programData, "backup-agent", "backup-agent.json"))
			paths = append(paths, filepath.Join(programData, "backup-monitor", "backup-agent.json"))
		}
	} else {
		paths = append(paths, "/etc/backup-agent/backup-agent.json")
	}
	return paths
}

// defaultLogPath 返回日志默认路径：二进制同目录的 agent.log
func defaultLogPath() string {
	if executable, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(executable), "agent.log")
	}
	return "agent.log"
}
