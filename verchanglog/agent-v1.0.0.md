# backup-monitor Agent agent-v1.0.0 版本更新日志

**发布日期**: 2026-10-26  
**提交数量**: 首个版本，随 v1.0.0 一同发布  
**版本类型**: 首个正式版本

---

## 📋 版本概述

备份监控 Agent 的第一个正式版本，随服务端 v1.0.0 一同发布。

它是一个部署在被监控主机上的单文件 Go 程序，职责只有两件事：**把「计划」读出来，把「实际执行」读出来**，两者一起上报给服务端。它不修改、不注入、不接管任何既有备份脚本——crontab 只读解析，Windows 计划任务优先通过 PowerShell 查询（`schtasks` / COM 回退），Veeam 作业读取配置与状态文件。即使 Agent 停止运行，原有备份作业也照常执行。

版本号记录在源码两处并保持一致：`internal/config/types.go` 的 `Version` 用于运行期输出与心跳上报，`cmd/backup-agent/version_marker.go` 的版本标记供服务端扫描二进制识别版本。

---

## ✨ 新增特性

### 任务发现

- **Linux crontab**：解析 `/etc/crontab`、`/etc/cron.d/*` 与各用户 crontab，支持 `@daily` 等特殊表达式，按标签与关键字过滤出备份相关任务
- **Windows 计划任务**：优先通过 PowerShell `Get-ScheduledTask` 采集，失败时回退 `schtasks`，PowerShell 2.0 环境再回退 COM `Schedule.Service`；解析触发时间、执行命令与上次运行结果
- **Veeam Job**：读取 Veeam 配置与状态，识别作业名、计划、上次结果与产物路径
- **产物校验**：按采集配置的路径与深度扫描备份产物，读取文件大小与修改时间，用于判断备份是否真正落盘

### 上报与指令

- **周期上报**：通过 `POST /api/ingest/report` 上报主机信息、任务清单与执行记录；未接入服务端时只发心跳
- **配置拉取**：`GET /api/agent/config` 按主机名与平台拉取全局参数与匹配的采集配置（`autoUpdateMinutes`、`artifactCheck`、`includeKeywords` 等由服务端下发，无需本地配置）
- **WebSocket 长连接**：`GET /api/agent/ws` 承载三类下行指令——配置变更（`config_changed`）、在线改计划（`set_cron`）、自更新（`update`），执行后回执 `set_cron_result` / `update_result`
- **令牌认证**：请求携带 `X-Agent-Token` 头，启用令牌校验时必填

### 运行与运维

- **服务化**：Linux 用 `-install` 注册 systemd 单元（记录二进制绝对路径，配置了 `SyslogIdentifier`），Windows 注册系统服务；`-uninstall` 卸载
- **前台试跑**：`-once` 执行一轮采集上报后退出，便于部署前验证配置
- **配置来源**：命令行参数 > 环境变量 > JSON 配置文件 > 默认值；`BM_*` 环境变量可覆盖配置文件
- **状态文件**：解析任务配置指定的 `start` / `end` / `size` / `transferred` / `exit` / `status` 等字段，用于生成执行记录；不持久化 Agent 自身状态
- **日志轮转**：按大小与保留份数滚动，路径可配
- **主机身份**：生成并持久化节点 ID，主机名变更后可识别为同一主机
- **平台信息上报**：上报 `arch`、`osBuild` 与 `agentStartedAt`，供服务端选择安装包、校验 Windows 兼容性并判断更新后是否完成重启
- **自更新**：比对服务端下发版本，下载新二进制并校验 SHA-256，通过临时文件替换后重启自身

### 平台与构建

- 独立 Go module（`backup-monitor-agent`，go 1.22），15 个内部包：`api`、`bootstrap`、`collect`、`config`、`cron`、`discover`、`identity`、`logging`、`proc`、`service`、`statusfile`、`sysinfo`、`update`、`veeam`、`wshb`
- 平台差异通过 `_linux.go` / `_windows.go` 构建标签隔离，交叉编译覆盖 linux/windows × amd64/arm64 四平台
- 管理端按「平台 + 架构 + 通道」托管 5 个槽位：linux/amd64、linux/arm64、windows/amd64、windows/arm64 为标准包，windows/amd64 另有 legacy（Go 1.20，兼容 Windows 7 / Server 2008 R2）
- 构建使用 `-trimpath -ldflags "-s -w"`，产物内附 `backup-agent.example.json` 与 `README.txt`；legacy 包文件名固定带 `-legacy`

---

## 🐛 Bug 修复

首个版本，无对外历史缺陷修复项。

---

## 🔧 重构优化

首个版本，无历史结构重构。

设计上值得说明的两点：

- 采集与发现分离：`discover` 只负责「读出计划」，`collect` 只负责「读出执行与产物」，两者互不依赖，新增采集源时不必改动上报链路
- 平台差异集中在少量 `_linux.go` / `_windows.go` 文件，公共逻辑不写 `runtime.GOOS` 分支

---

## 🔄 升级说明

**全新安装，无历史版本需要迁移。**

1. 从 Releases 下载对应平台压缩包（`backup-agent_1.0.0_<os>_<arch>.tar.gz` 或 `.zip`；Windows 7 / Server 2008 R2 使用 `backup-agent_1.0.0_windows_amd64_legacy.zip`），解压得到 `backup-agent` 可执行文件、`backup-agent.example.json` 与 `README.txt`
2. 复制 `backup-agent.example.json` 为 `backup-agent.json`，填入 `serverUrl` 与在管理端「Agent 管理」创建的令牌
3. 前台试跑确认能连上服务端：`./backup-agent -once`
4. 安装为系统服务：`./backup-agent -install`；安装后**不要移动或改名二进制文件**，systemd 单元记录的是安装时的绝对路径
5. Windows 以管理员身份运行 `backup-agent.exe -install`
6. 也可不用手动下载：在管理端上传二进制后，由服务端分发给各主机

兼容性：Agent 发布线与服务端 v1.0.0 配套；协议新增字段向后兼容，服务端忽略未知字段。完整协议见 `docs/agent-protocol.md`，构建与排障见 `agent/README.md` 与 `agent/README.en.md`。

---

## 👥 贡献者

- Jerion

---

**完整提交记录**: 首个版本，随 v1.0.0 一同发布
