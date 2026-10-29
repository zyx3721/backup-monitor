// Package main 是 backup-monitor Agent 的命令行入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"backup-monitor-agent/internal/api"
	"backup-monitor-agent/internal/bootstrap"
	"backup-monitor-agent/internal/collect"
	"backup-monitor-agent/internal/config"
	"backup-monitor-agent/internal/identity"
	"backup-monitor-agent/internal/logging"
	"backup-monitor-agent/internal/service"
	"backup-monitor-agent/internal/sysinfo"
	"backup-monitor-agent/internal/update"
	"backup-monitor-agent/internal/wshb"
)

func main() {
	os.Exit(run())
}

// stripRunSubcommand 兼容 `backup-agent run -config x` 旧写法，忽略首个 run 子命令
func stripRunSubcommand() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
}

func run() int {
	stripRunSubcommand()
	serverFlag := flag.String("server", "", "服务端地址，等价环境变量 BM_SERVER_URL（默认空）")
	tokenFlag := flag.String("token", "", "Agent 令牌，等价环境变量 BM_AGENT_TOKEN（默认空）")
	hostnameFlag := flag.String("hostname", "", "上报主机名，留空使用系统主机名，等价环境变量 BM_HOSTNAME（默认空）")
	roleFlag := flag.String("role", "", "主机角色：database、web、application、file、virtualization，等价环境变量 BM_ROLE（默认 database）")
	intervalFlag := flag.Int("interval", 0, "本地上报间隔兜底值（秒），0 表示使用服务端配置，等价环境变量 BM_INTERVAL_SEC（默认 0）")
	configFlag := flag.String("config", "", "配置文件路径，留空按同目录、系统目录顺序查找，等价环境变量 BM_CONFIG_FILE（默认空）")
	logFlag := flag.String("log", "", "日志文件路径，留空写入二进制同目录 agent.log，等价环境变量 BM_LOG_FILE（默认空）")
	logMaxMbFlag := flag.Int("log-max-mb", 0, "单个日志文件上限（MB），留空用 10，等价环境变量 BM_LOG_MAX_MB（默认 0）")
	logBackupsFlag := flag.Int("log-backups", 0, "日志保留的历史文件个数，留空用 2（即共 3 个文件），等价环境变量 BM_LOG_BACKUPS（默认 0）")
	onceFlag := flag.Bool("once", false, "采集并上报一次后立即退出（默认 false）")
	serviceFlag := flag.Bool("service", false, "以服务方式运行，日志不写 stdout（由服务管理器传入，Windows 下单独使用需先安装服务，默认 false）")
	installFlag := flag.Bool("install", false, "安装并启动系统服务（默认 false）")
	uninstallFlag := flag.Bool("uninstall", false, "停止并卸载系统服务（默认 false）")
	nameFlag := flag.String("name", "", "服务名，默认 backup-agent，等价环境变量 BM_SERVICE_NAME（默认空）")
	displayNameFlag := flag.String("displayname", "", "服务显示名，留空按服务名生成，Linux 写入单元 Description，等价环境变量 BM_DISPLAY_NAME（默认空）")
	versionFlag := flag.Bool("version", false, "显示版本信息并退出（默认 false）")
	versionShortFlag := flag.Bool("v", false, "显示版本信息并退出（-version 的缩写）")
	restartHelperFlag := flag.String("restart-helper", "", "内部使用：Windows 服务重启守护的状态文件路径")
	flag.Parse()

	if *versionFlag || *versionShortFlag {
		fmt.Printf("backup-agent %s %s/%s\n", config.Version, runtime.GOOS, runtime.GOARCH)
		return 0
	}

	if strings.TrimSpace(*restartHelperFlag) != "" {
		if err := service.RunRestartHelper(*restartHelperFlag); err != nil {
			fmt.Fprintf(os.Stderr, "重启守护失败：%v\n", err)
			return 1
		}
		return 0
	}

	serviceName := service.ResolveName(*nameFlag)
	displayName := service.ResolveDisplayName(*displayNameFlag, serviceName)
	if strings.TrimSpace(*nameFlag) != "" || strings.TrimSpace(os.Getenv("BM_SERVICE_NAME")) != "" {
		service.UseName(serviceName)
	}

	overrides := bootstrap.Overrides{}
	flag.Visit(func(item *flag.Flag) {
		switch item.Name {
		case "server":
			overrides.ServerURL = serverFlag
		case "token":
			overrides.Token = tokenFlag
		case "hostname":
			overrides.Hostname = hostnameFlag
		case "role":
			overrides.Role = roleFlag
		case "interval":
			overrides.IntervalSec = intervalFlag
		case "config":
			overrides.ConfigFile = configFlag
		case "log":
			overrides.LogFile = logFlag
		case "log-max-mb":
			overrides.LogMaxMb = logMaxMbFlag
		case "log-backups":
			overrides.LogBackups = logBackupsFlag
		}
	})

	loaded, err := bootstrap.Load(overrides)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败：%v\n", err)
		return 1
	}

	logger, closeLog, logErr := logging.Setup(loaded.Config.LogFile, loaded.Config.LogMaxMb, loaded.Config.LogBackups)
	if logErr != nil {
		logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("日志文件不可用，改为输出到标准输出：%v", logErr)
		closeLog = func() error { return nil }
	}
	defer closeLog()
	logger.Printf("backup-agent-version:%s", config.Version)
	if strings.TrimSpace(loaded.Path) != "" {
		logger.Printf("配置文件 %s", loaded.Path)
	} else {
		logger.Printf("配置文件 未找到，使用默认配置")
	}

	if *installFlag {
		message, installErr := service.Install(executablePath(), serviceName, displayName)
		if installErr != nil {
			logger.Printf("安装服务失败：%v", installErr)
			return 1
		}
		logger.Printf("%s", message)
		return 0
	}
	if *uninstallFlag {
		message, uninstallErr := service.Uninstall(serviceName)
		if uninstallErr != nil {
			logger.Printf("卸载服务失败：%v", uninstallErr)
			return 1
		}
		logger.Printf("%s", message)
		return 0
	}

	if strings.TrimSpace(loaded.Config.ServerURL) == "" {
		logger.Printf("未配置服务端地址")
		return 1
	}
	update.WaitForPreviousProcess()
	update.CleanupTempFiles()
	serviceMode := *serviceFlag || service.IsService()

	nodeID, idErr := identity.Get()
	if idErr != nil {
		logger.Printf("读取机器标识失败，使用备用标识：%v", idErr)
	}
	hostname := strings.TrimSpace(loaded.Config.Hostname)
	if hostname == "" {
		hostname = sysinfo.Hostname()
	}
	tokenState := "未配置"
	if strings.TrimSpace(loaded.Config.Token) != "" {
		tokenState = "已配置"
	}
	logger.Printf("服务器 %s，主机名 %s（token %s）", loaded.Config.ServerURL, hostname, tokenState)

	application := newAgent(loaded.Config, logger, nodeID, hostname, serviceMode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *onceFlag {
		onceCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := application.runOnce(onceCtx); err != nil {
			logger.Printf("单次采集失败：%v", err)
			return 1
		}
		logger.Printf("单次采集完成")
		return 0
	}

	runner := func(runCtx context.Context) error {
		return application.run(runCtx)
	}
	if serviceMode {
		if err := service.Run(ctx, runner); err != nil {
			logger.Printf("服务退出：%v", err)
			return 1
		}
		return 0
	}
	if err := runner(ctx); err != nil {
		logger.Printf("Agent 退出：%v", err)
		return 1
	}
	return 0
}

func executablePath() string {
	executable, err := os.Executable()
	if err != nil {
		return os.Args[0]
	}
	return executable
}

type agent struct {
	cfg            config.Local
	logger         *log.Logger
	nodeID         string
	hostname       string
	platform       string
	serviceMode    bool
	client         *api.Client
	clientTimeout  time.Duration
	lastConfig     *api.ConfigResponse
	lastError      string
	reload         chan struct{}
	wsWake         chan struct{}
	mutex          sync.Mutex
	wsCancel       context.CancelFunc
	wsKey          string
	lastUpdate     time.Time
	autoUpdate     int
	startedAt      time.Time
	updateCleanup  sync.Once
	loggedInterval time.Duration
}

func newAgent(cfg config.Local, logger *log.Logger, nodeID, hostname string, serviceMode bool) *agent {
	instance := &agent{
		cfg:         cfg,
		logger:      logger,
		nodeID:      nodeID,
		hostname:    hostname,
		platform:    sysinfo.Platform(),
		serviceMode: serviceMode,
		startedAt:   time.Now(),
		reload:      make(chan struct{}, 1),
		wsWake:      make(chan struct{}, 1),
	}
	client, err := api.New(cfg.ServerURL, cfg.Token, time.Duration(config.DefaultHTTPTimeoutSec)*time.Second)
	if err != nil {
		logger.Printf("创建 HTTP 客户端失败：%v", err)
	} else {
		instance.client = client
		instance.clientTimeout = time.Duration(config.DefaultHTTPTimeoutSec) * time.Second
	}
	return instance
}

func (a *agent) runOnce(ctx context.Context) error {
	a.fetchConfig(ctx)
	profile := a.profile()
	report := a.collect(ctx, profile)
	if a.client == nil {
		return errors.New("HTTP client is not available")
	}
	response, err := a.client.Report(ctx, report)
	if err != nil {
		return err
	}
	a.notifyReportSuccess()
	a.cleanupUpdateBackup()
	a.logger.Printf("%s 上报完成：任务 %d 个，新增执行 %d 条",
		a.hostname, len(report.Tasks), response.NewExecutions)
	return nil
}

func (a *agent) run(ctx context.Context) error {
	a.loggedInterval = a.reportInterval()
	a.logger.Printf("Agent 已启动，初始上报间隔 %s（收到服务端配置后自动调整）", a.loggedInterval)
	defer func() {
		a.stopWebSocket()
		a.logger.Printf("Agent 已停止")
	}()

	for {
		if ctx.Err() != nil {
			return nil
		}
		a.fetchConfig(ctx)
		a.ensureWebSocket(ctx)

		profile := a.profile()
		report := a.collect(ctx, profile)
		if a.client == nil {
			a.logger.Printf("跳过上报：HTTP 客户端不可用")
		} else {
			response, err := a.client.Report(ctx, report)
			if err != nil {
				a.logfThrottled("%v", err)
			} else {
				a.lastError = ""
				a.notifyReportSuccess()
				a.cleanupUpdateBackup()
				a.logger.Printf("%s 上报完成：任务 %d 个，新增执行 %d 条",
					a.hostname, len(report.Tasks), response.NewExecutions)
			}
		}

		a.maybeAutoUpdate(ctx)

		timer := time.NewTimer(a.reportInterval())
		select {
		case <-ctx.Done():
			timer.Stop()
			a.logger.Printf("收到退出信号，正在停止…")
			return nil
		case <-a.reload:
			timer.Stop()
			a.logger.Printf("配置已变更，重新加载")
		case <-timer.C:
		}
	}
}

func (a *agent) collect(ctx context.Context, profile *api.ProfileConfig) api.IngestReport {
	options := collect.Options{}
	if profile != nil {
		options.Profile = *profile
	}
	if a.lastConfig != nil {
		options.Settings = a.lastConfig.Settings
	}
	collector := collect.Collector{
		Hostname:  a.hostname,
		NodeID:    a.nodeID,
		Role:      a.cfg.Role,
		Version:   config.Version,
		StartedAt: a.startedAt,
		Logger:    a.logger,
	}
	report, err := collector.Collect(ctx, options)
	if err != nil {
		a.logfThrottled("采集失败：%v", err)
	}
	return report
}

func (a *agent) fetchConfig(ctx context.Context) {
	client := a.apiClient(a.httpTimeoutSec())
	if client == nil {
		return
	}
	response, err := client.Config(ctx, a.hostname, a.platform)
	if err != nil {
		a.logfThrottled("%v", err)
		return
	}
	a.lastConfig = response
	a.lastError = ""
	a.syncAutoUpdateInterval(response)
	if interval := a.reportInterval(); interval != a.loggedInterval {
		a.logger.Printf("上报间隔已调整为 %s", interval)
		a.loggedInterval = interval
	}
	a.apiClient(response.Settings.Agent.HTTPTimeoutSec)
}

func (a *agent) profile() *api.ProfileConfig {
	if a.lastConfig == nil || a.lastConfig.Profile == nil || !a.lastConfig.Profile.Enabled {
		return nil
	}
	profile := a.lastConfig.Profile.Config
	return &profile
}

func (a *agent) apiClient(timeoutSeconds int) *api.Client {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(config.DefaultHTTPTimeoutSec) * time.Second
	}
	if a.client != nil && a.clientTimeout == timeout {
		return a.client
	}
	client, err := api.New(a.cfg.ServerURL, a.cfg.Token, timeout)
	if err != nil {
		a.logger.Printf("创建 HTTP 客户端失败：%v", err)
		return a.client
	}
	a.client = client
	a.clientTimeout = timeout
	return client
}

func (a *agent) httpTimeoutSec() int {
	if a.lastConfig != nil && a.lastConfig.Settings.Agent.HTTPTimeoutSec > 0 {
		return a.lastConfig.Settings.Agent.HTTPTimeoutSec
	}
	return config.DefaultHTTPTimeoutSec
}

func (a *agent) reportInterval() time.Duration {
	seconds := 0
	if a.lastConfig != nil {
		if profile := a.profile(); profile != nil && profile.ReportIntervalSec > 0 {
			seconds = profile.ReportIntervalSec
		}
		if seconds == 0 {
			seconds = a.lastConfig.Settings.Agent.ReportIntervalSec
		}
	}
	if seconds == 0 {
		seconds = a.cfg.IntervalSec
	}
	if seconds == 0 {
		seconds = config.DefaultIntervalSec
	}
	if seconds < config.MinIntervalSec {
		seconds = config.MinIntervalSec
	}
	return time.Duration(seconds) * time.Second
}

func (a *agent) ensureWebSocket(ctx context.Context) {
	heartbeat := config.DefaultHeartbeatIntervalSec
	backoffMin := config.DefaultWSBackoffMinSec
	backoffMax := config.DefaultWSBackoffMaxSec
	if a.lastConfig != nil {
		if profile := a.profile(); profile != nil && profile.HeartbeatIntervalSec > 0 {
			heartbeat = profile.HeartbeatIntervalSec
		}
		if a.lastConfig.Settings.Agent.WSBackoffMinSec > 0 {
			backoffMin = a.lastConfig.Settings.Agent.WSBackoffMinSec
		}
		if a.lastConfig.Settings.Agent.WSBackoffMaxSec > 0 {
			backoffMax = a.lastConfig.Settings.Agent.WSBackoffMaxSec
		}
	}
	if backoffMax < backoffMin {
		backoffMax = backoffMin
	}
	key := fmt.Sprintf("%d/%d/%d", heartbeat, backoffMin, backoffMax)
	a.mutex.Lock()
	if a.wsCancel != nil && a.wsKey == key {
		a.mutex.Unlock()
		return
	}
	if a.wsCancel != nil {
		a.wsCancel()
	}
	wsCtx, cancel := context.WithCancel(ctx)
	a.wsCancel = cancel
	a.wsKey = key
	a.mutex.Unlock()

	client := wshb.New(wshb.Options{
		ServerURL:    a.cfg.ServerURL,
		Token:        a.cfg.Token,
		Hostname:     a.hostname,
		NodeID:       a.nodeID,
		Platform:     a.platform,
		Arch:         sysinfo.Arch(),
		OSBuild:      sysinfo.OSBuild(),
		AgentVersion: config.Version,
		Heartbeat:    time.Duration(heartbeat) * time.Second,
		BackoffMin:   time.Duration(backoffMin) * time.Second,
		BackoffMax:   time.Duration(backoffMax) * time.Second,
		Logger:       a.logger,
		Wake:         a.wsWake,
		OnConfigChanged: func() {
			select {
			case a.reload <- struct{}{}:
			default:
			}
		},
		Update: update.Options{
			ServerURL:     a.cfg.ServerURL,
			Token:         a.cfg.Token,
			ServiceMode:   a.serviceMode,
			Logger:        a.logger,
			ExitOnRestart: true,
		},
	})
	go client.Run(wsCtx)
}

func (a *agent) stopWebSocket() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.wsCancel != nil {
		a.wsCancel()
		a.wsCancel = nil
		a.wsKey = ""
	}
}

// syncAutoUpdateInterval 记录下发的自动更新间隔，变化后从当前时刻重新计时
func (a *agent) syncAutoUpdateInterval(response *api.ConfigResponse) {
	minutes := autoUpdateMinutesOf(response)
	if minutes == a.autoUpdate {
		return
	}
	a.autoUpdate = minutes
	a.lastUpdate = time.Now()
	if minutes > 0 {
		a.logger.Printf("自动更新间隔已调整为 %d 分钟，下次检查在 %d 分钟后", minutes, minutes)
		return
	}
	a.logger.Printf("自动更新已关闭")
}

// autoUpdateMinutesOf 返回采集配置下发的自动更新间隔，配置未启用时为 0
func autoUpdateMinutesOf(response *api.ConfigResponse) int {
	if response == nil || response.Profile == nil || !response.Profile.Enabled {
		return 0
	}
	minutes := response.Profile.Config.AutoUpdateMinutes
	if minutes < 0 {
		return 0
	}
	return minutes
}

func (a *agent) maybeAutoUpdate(ctx context.Context) {
	profile := a.profile()
	if profile == nil || profile.AutoUpdateMinutes <= 0 {
		return
	}
	now := time.Now()
	if !a.lastUpdate.IsZero() && now.Sub(a.lastUpdate) < time.Duration(profile.AutoUpdateMinutes)*time.Minute {
		return
	}
	a.lastUpdate = now

	client, err := api.New(a.cfg.ServerURL, a.cfg.Token, time.Minute)
	if err != nil {
		a.logfThrottled("create update client failed: %v", err)
		return
	}
	latest, err := client.LatestBinary(ctx, a.platform, sysinfo.Arch(), sysinfo.OSBuild())
	if err != nil {
		if !errors.Is(err, api.ErrNoBinary) {
			a.logfThrottled("query latest agent binary failed: %v", err)
		}
		return
	}
	if update.CompareVersion(latest.Version, config.Version) <= 0 {
		return
	}
	if update.FailureBlocked(latest.Version, now) {
		a.logfThrottled("自动更新 %s 失败退避中，暂不重试", latest.Version)
		return
	}

	a.logger.Printf("发现新版本：当前 %s，最新 %s", config.Version, latest.Version)
	options := update.Options{
		ServerURL:     a.cfg.ServerURL,
		Token:         a.cfg.Token,
		Version:       latest.Version,
		SHA256:        latest.SHA256,
		DownloadURL:   update.DownloadPath(a.platform, sysinfo.Arch(), sysinfo.OSBuild()),
		ServiceMode:   a.serviceMode,
		Logger:        a.logger,
		ExitOnRestart: true,
	}
	if err := update.Apply(ctx, options); err != nil {
		a.logfThrottled("apply agent update failed: %v", err)
		if recordErr := update.RecordFailure(latest.Version); recordErr != nil {
			a.logfThrottled("record agent update failure failed: %v", recordErr)
		}
	}
}

func (a *agent) logfThrottled(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if message == a.lastError {
		return
	}
	a.lastError = message
	a.logger.Printf("%s", message)
}

func (a *agent) notifyReportSuccess() {
	select {
	case a.wsWake <- struct{}{}:
	default:
	}
}

// cleanupUpdateBackup 在新版本首次上报成功后删除旧版本备份
func (a *agent) cleanupUpdateBackup() {
	a.updateCleanup.Do(func() {
		if update.CleanupBackup() {
			a.logger.Printf("新版本首次上报成功，已删除旧版本备份")
		}
	})
}
