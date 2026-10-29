<div align="center">

<h1>Backup Monitor Agent</h1>

<p><b>独立主机采集 Agent</b> — 只读发现备份任务、状态与产物元数据，HTTP 周期上报，WebSocket 接收配置与更新指令</p>

<p><b>简体中文</b> · <a href="README.en.md">English</a></p>

Go 实现的独立采集 Agent：在目标主机上发现备份任务、读取状态与产物元数据，按周期通过 HTTP 上报，并通过 WebSocket 接收配置变更、修改执行计划与自更新命令。协议以 [`docs/agent-protocol.md`](../docs/agent-protocol.md) 为准。

部署与生产环境的完整步骤见[《完整手册》4.3](../docs/manual.md)，本文只讲 Agent 自身的构建、配置与运维。

<p>
  <a href="../LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?labelColor=1f2937" alt="MIT License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white&labelColor=1f2937" alt="Go 1.22+"></a>
  <img src="https://img.shields.io/badge/Agent-Linux%20%7C%20Windows-0ea5e9?labelColor=1f2937" alt="Agent Linux Windows">
</p>

<p>
  <b><a href="#目录结构">目录结构</a></b> ·
  <b><a href="#系统要求">系统要求</a></b> ·
  <b><a href="#构建">构建</a></b> ·
  <b><a href="#配置">配置</a></b> ·
  <b><a href="#运行">运行</a></b> ·
  <b><a href="#安装为服务">安装为服务</a></b> ·
  <b><a href="#升级与回滚">升级与回滚</a></b> ·
  <b><a href="#排障">排障</a></b>
</p>

</div>

---

**Agent 的边界：**

- **只读采集** — 不修改业务数据，不改服务端、前端或仓库其它目录
- **不上传备份内容** — 产物校验只读文件 mtime 与大小
- **无外部依赖** — 仅依赖 `github.com/gorilla/websocket` 与 `golang.org/x/sys`，不需要数据库与运行时

## 目录结构

```text
agent/
├── backup-agent.example.json       配置示例（所有敏感项留空）
├── go.mod
├── cmd/backup-agent/
│   ├── main.go                     命令行入口与常驻循环
│   └── version_marker.go           版本标记，上传服务端时据此识别版本
└── internal/
    ├── api/                        HTTP 客户端与协议结构
    ├── bootstrap/                  参数 > 环境变量 > 配置文件 > 默认值
    ├── collect/                    任务与执行记录组装
    ├── config/                     本地配置结构、角色与默认值
    ├── cron/                       5 段 cron 解析、最近计划、中文描述
    ├── discover/                   crontab / Windows 计划任务发现与回写
    ├── identity/                   machine-id / MachineGuid / 持久化 UUID
    ├── logging/                    按大小轮转的文件日志（上限与保留个数可配）
    ├── proc/                       进程探测
    ├── service/                    systemd / Windows 服务
    ├── statusfile/                 start/end/size/exit/status 状态文件
    ├── sysinfo/                    主机名、内网 IP、OS
    ├── update/                     版本比较、下载校验、替换与重启
    ├── veeam/                      veeamconfig 表格解析
    └── wshb/                       WebSocket 心跳、命令与指数退避
```

## 系统要求

| 产物 | 最低系统 |
|:--|:--|
| 常规包 | Windows 10 / Server 2016 及以上；Linux 主流发行版（x86_64 / arm64） |
| `-legacy` 包 | Windows 7 SP1 / Server 2008 R2 SP1 及以上，仅提供 amd64 |

常规包由 Go 1.22+ 编译，runtime 已不支持 Windows 7 / 8 / 8.1 与 Server 2008 / 2012。在更低版本上进程会在 runtime 初始化阶段以 `0xc0000005`、`PC=0x0` 崩溃，需改用 `-legacy` 包。

管理端按「平台 + 架构 + 通道」托管 5 个槽位，推送与自动更新会按主机上报的 CPU 架构与 Windows 构建号选包：Build ≥ 10240 用标准包（缺失即跳过，不回退 legacy），低于 10240 用 legacy；未上报的主机按 amd64 标准包下发并标注「未校验」。

Windows 7 / Server 2008 R2 的系统根证书较旧，若服务端使用 HTTPS 且证书链来自较新的 CA，需先更新根证书，否则会卡在证书校验。

## 构建

需要 Go 1.22 或更高版本，以 `agent/go.mod` 的 `go` 指令为准；Windows 7 / Server 2008 R2 的兼容包需 Go 1.20，见下文「Windows legacy 兼容包」。

```bash
cd agent
go build ./...
go vet ./...
go test ./...
```

以上三条也是本目录的验证基线，其余章节不再重复。

交叉编译（服务端托管同时支持 amd64 与 arm64，按目标主机架构选择）：

```bash
# Linux amd64 / arm64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux ./cmd/backup-agent
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux-arm64 ./cmd/backup-agent
```

```powershell
# Windows amd64 / arm64
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-amd64.exe ./cmd/backup-agent

$env:GOOS = "windows"; $env:GOARCH = "arm64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-arm64.exe ./cmd/backup-agent
```

`-trimpath` 去掉构建机路径，`-ldflags "-s -w"` 去掉符号表与调试信息，通常能再瘦 20%~30%，代价是无法用 dlv、gdb 直接调试；需要线上排查时另编一份不裁剪的版本。

### Windows legacy 兼容包

Windows 7 / Server 2008 R2 需要 Go 1.20 编译。`go.legacy.mod` 与 `go.legacy.sum` 是主 `go.mod` / `go.sum` 的副本，仅把 `go` 指令降到 1.20、依赖版本保持一致，主 `go.mod` 不动；构建时用 `-modfile` 指向它：

```powershell
# 首次使用：安装并下载 Go 1.20.14
go install golang.org/dl/go1.20.14@latest
go1.20.14 download

$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "amd64"
go1.20.14 build -modfile "go.legacy.mod" -trimpath -ldflags "-s -w" `
  -o dist/backup-agent-windows-amd64-legacy.exe ./cmd/backup-agent
```

产物文件名固定带 `-legacy`，只提供 amd64，与常规包同版本号，靠文件名与系统要求区分；`-modfile` 与值之间用空格，PowerShell 下写成 `-modfile=go.legacy.mod` 会被拆成两个参数而报错。

**必须使用包目录 `./cmd/backup-agent`**：该目录下的 `version_marker.go` 会把 `backup-agent-version:X.Y.Z` 标记写进产物，服务端上传时据此自动识别版本与架构。只编译 `main.go` 时程序仍能运行，但产物没有该标记，上传时只能手工填写版本号。

发布新版本时同步更新两处版本号：`internal/config/types.go` 的 `Version`（运行期上报用）与 `cmd/backup-agent/version_marker.go` 的 `embeddedVersion`（服务端识别用）。

## 配置

加载顺序为 **命令行参数 > 环境变量 > JSON 配置文件 > 默认值**。

配置文件查找顺序：

1. `-config` 指定的路径（必须存在）
2. 环境变量 `BM_CONFIG_FILE` 指定的路径
3. 可执行文件同目录 `backup-agent.json`
4. Linux `/etc/backup-agent/backup-agent.json`；Windows `%ProgramData%\backup-agent\backup-agent.json`

注意第 3 条指的是**二进制所在目录**。用 `go run ./cmd/backup-agent` 调试时，二进制在系统临时目录，找不到项目目录里的配置，必须显式传 `-config`（或设 `BM_CONFIG_FILE`）；编译后放进二进制目录才能省略。

配置字段：

| 字段 | 说明 | 默认值 |
|:--|:--|:--|
| `serverUrl` | 服务端地址，必须为 http/https 绝对地址 | 空 |
| `token` | `X-Agent-Token` 令牌 | 空 |
| `hostname` | 上报主机名，留空使用系统主机名 | 空 |
| `intervalSec` | 本地上报间隔兜底值，0 表示使用服务端配置 | 600 |
| `role` | 主机角色：`database`、`web`、`application`、`file`、`virtualization` | `database` |
| `logFile` | 日志文件路径，留空写入二进制同目录 `agent.log` | 同目录 `agent.log` |
| `logMaxMb` | 单个日志文件上限（MB），0 或留空表示用默认值 | 10 |
| `logBackups` | 保留的历史日志文件个数，0 或留空表示用默认值；当前文件之外另存 N 个 | 2 |

字段与 `internal/config/types.go` 的 `Local` 一一对应。补充两点：

- `intervalSec` 只是**兜底值**：实际生效间隔依次取「服务端下发的采集配置 > 服务端全局设置 > 本地 `intervalSec` > 默认 600」。填 `0` 表示完全交给服务端；填负数会启动失败；填 `1`～`9` 会被抬到最小值 10
- `role` 非法值会被忽略并回落到 `database`；服务端只在主机首次注册时采用该值，之后角色以管理端配置为准，上报不会覆盖手动修改

### 环境变量

上表中的每个字段都有对应的环境变量（命名不完全一致，以下表为准），容器或服务管理器里注入更方便：

| 变量 | 对应 | 说明 |
|:--|:--|:--|
| `BM_SERVER_URL` | `-server` / `serverUrl` | 服务端地址 |
| `BM_AGENT_TOKEN` | `-token` / `token` | Agent 令牌 |
| `BM_HOSTNAME` | `-hostname` / `hostname` | 上报主机名 |
| `BM_ROLE` | `-role` / `role` | 主机角色 |
| `BM_INTERVAL_SEC` | `-interval` / `intervalSec` | 上报间隔兜底（秒） |
| `BM_CONFIG_FILE` | `-config` | 配置文件路径 |
| `BM_LOG_FILE` | `-log` / `logFile` | 日志文件路径 |
| `BM_LOG_MAX_MB` | `-log-max-mb` / `logMaxMb` | 单个日志文件上限（MB） |
| `BM_LOG_BACKUPS` | `-log-backups` / `logBackups` | 历史日志文件个数 |

整型变量填非整数会直接报错并指出变量名（如 `invalid BM_LOG_MAX_MB: "abc" is not an integer`），不会静默忽略。

### 示例配置

JSON 不支持注释，字段含义见上表：

```json
{
  "serverUrl": "",
  "token": "",
  "hostname": "",
  "intervalSec": 0,
  "role": "database",
  "logFile": "",
  "logMaxMb": 10,
  "logBackups": 2
}
```

典型配置——Linux 服务（放默认位置，安装后无需再传参数）：

```json
{
  "serverUrl": "http://10.0.0.10:8080",
  "token": "在管理端 Agent 令牌页生成",
  "role": "database"
}
```

典型配置——Windows 服务（显式固定主机名，便于多实例区分）：

```json
{
  "serverUrl": "https://backup.example.com",
  "token": "在管理端 Agent 令牌页生成",
  "hostname": "db-prod-02",
  "role": "database"
}
```

`serverUrl` 与 `token` 不应写入仓库，部署时通过配置文件或环境变量注入。Windows 的 JSON 路径需写成双反斜杠。

## 运行

```bash
# 常驻：HTTP 周期上报 + WebSocket 心跳/命令 + 自动更新
./backup-agent

# 只采集并上报一次
./backup-agent -once

# 查看版本
./backup-agent -version
```

常用参数（`-h` 输出的中文说明与此表一致，等价环境变量同名列）：

| 参数 | 说明 | 环境变量 |
|:--|:--|:--|
| `-server` | 服务端地址 | `BM_SERVER_URL` |
| `-token` | Agent 令牌 | `BM_AGENT_TOKEN` |
| `-hostname` | 上报主机名，留空使用系统主机名 | `BM_HOSTNAME` |
| `-role` | 主机角色 | `BM_ROLE` |
| `-interval` | 本地上报间隔兜底（秒），0 表示使用服务端配置 | `BM_INTERVAL_SEC` |
| `-config` | 配置文件路径 | `BM_CONFIG_FILE` |
| `-log` | 日志文件路径，留空写入二进制同目录 `agent.log` | `BM_LOG_FILE` |
| `-log-max-mb` | 单个日志文件上限（MB），留空用 10 | `BM_LOG_MAX_MB` |
| `-log-backups` | 保留的历史日志文件个数，留空用 2（即共 3 个文件） | `BM_LOG_BACKUPS` |
| `-once` | 采集并上报一次后退出 | - |
| `-name` | 服务名，默认 `backup-agent` | `BM_SERVICE_NAME` |
| `-displayname` | 服务显示名，留空按服务名生成（Linux 写入单元 Description） | `BM_DISPLAY_NAME` |
| `-service` | 以服务模式运行（日志不写 stdout），由服务管理器传入；Windows 下单独使用需先安装服务 | - |
| `-install` | 安装系统服务并启动 | - |
| `-uninstall` | 停止系统服务并卸载 | - |
| `-version`、`-v` | 打印版本号后退出 | - |

参数写 `-xxx` 或 `--xxx` 等价（Go flag 两种都接受）。兼容旧写法 `backup-agent run -config xxx.json`：首个 `run` 会被忽略，其余参数照常解析。

常驻模式会先拉取 `/api/agent/config`，再按服务端下发的采集配置执行；拉取失败时继续上报主机心跳并退避重试。`profile` 为 `null` 时只上报主机心跳，不上报任务与执行记录。

## 安装为服务

服务名默认为 `backup-agent`，显示名默认 `Backup Monitor Agent`；多实例自动派生为 `Backup Monitor Agent (<服务名>)`，也可用 `-displayname` 指定（Windows 服务显示名 / Linux 单元 Description）。`-install`、`-uninstall` 与 `-service` 都支持 `-name <服务名>`（也可用 `BM_SERVICE_NAME`），用于同机多实例或与既有命名对齐。

- **已存在不重复创建** — `-install` 会先查同名服务：Linux 依次检查 `/etc/systemd/system`、`/run/systemd/system`、`/usr/lib/systemd/system`、`/lib/systemd/system` 下的 `<name>.service`，Windows 查 SCM；命中则直接提示，不覆盖也不重复创建
- **卸载范围一致** — `-uninstall` 在同样的目录范围内查找单元并删除；Windows 按 SCM 中的服务名停止并删除

安装时会把**执行安装那一刻二进制的绝对路径**写入服务定义，所以放哪个目录都行，只要：

- **路径稳定** — 不要放在 `/tmp`、解压即弃的目录或会被清理的位置
- **先就位再安装** — 移动或改名二进制后必须先 `-uninstall` 再 `-install`，否则服务仍指向旧路径
- **配置放在默认位置** — 服务以 `-service` 启动，单元里不带 `-config`，所以 `backup-agent.json` 必须落在二进制同目录或系统默认目录（见「配置」一节的查找顺序）

推荐部署目录（仅作参考，不是强制）：

| 平台 | 二进制 | 配置 | 日志 |
|:--|:--|:--|:--|
| Linux | `/opt/backup-agent/backup-agent` 或 `/usr/local/bin/backup-agent` | 二进制同目录或 `/etc/backup-agent/backup-agent.json` | 二进制同目录 `agent.log` |
| Windows | `C:\ProgramData\backup-agent\backup-agent.exe` | `C:\ProgramData\backup-agent\backup-agent.json` | 二进制同目录 `agent.log` |

### Linux（systemd）

需要 root 权限，默认单元文件写入 `/etc/systemd/system/backup-agent.service`：

```bash
# 安装（单元记录该文件的绝对路径）
sudo ./backup-agent -install
systemctl status backup-agent

# 自定义服务名
sudo ./backup-agent -install -name backup-agent-2
systemctl status backup-agent-2

# 卸载（只停服务并删单元，配置与日志保留；自定义名需带 -name）
sudo ./backup-agent -uninstall
sudo ./backup-agent -uninstall -name backup-agent-2
```

单元内容等价于（以服务名 `<服务名>` 为例）：

```ini
[Unit]
Description=<服务显示名>
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=<安装时二进制的绝对路径> -service -name <服务名>
Restart=always
RestartSec=10
StandardOutput=null
StandardError=null
SyslogIdentifier=<服务名>

[Install]
WantedBy=multi-user.target
```

`StandardOutput` 与 `StandardError` 均为 `null`，日志只由 Agent 写入二进制同目录的 `agent.log`，不进 journald；`SyslogIdentifier` 跟随服务名，是为将来改回 `journal` 时准备的标识，`journalctl -t <服务名>` 可直接过滤。

### Windows

以管理员身份运行，建议先把二进制放到 `C:\ProgramData\backup-agent\` 再从该目录安装：

```powershell
cd C:\ProgramData\backup-agent
.\backup-agent.exe -install
Get-Service backup-agent

# 自定义服务名
.\backup-agent.exe -install -name backup-agent-2
Get-Service backup-agent-2

.\backup-agent.exe -uninstall
```

服务名默认为 `backup-agent`，显示名默认 `Backup Monitor Agent`（多实例自动追加 `(<服务名>)`，也可用 `-displayname` 指定），启动类型为自动启动，启动参数为 `-service -name <服务名>`。重复执行 `-install` 时若同名服务已存在会直接提示，不会重复创建；非管理员执行会返回明确的权限错误。

## 日志

- 默认写入**二进制同目录** `agent.log`，可用 `-log` / `logFile` / `BM_LOG_FILE` 覆盖
- 轮转按大小触发，两项均可调：
  - 单个文件上限：`-log-max-mb` / `logMaxMb` / `BM_LOG_MAX_MB`，默认 **10**（MB）
  - 历史文件个数：`-log-backups` / `logBackups` / `BM_LOG_BACKUPS`，默认 **2**，即共 3 个文件（`agent.log`、`agent.log.1`、`agent.log.2`）
- 填 `0` 或不填表示使用默认值，填负数会启动失败
- 日志内容为中文；`[ws]` 等固定前缀保留，便于检索
- 服务模式下不写 stdout；文件打开失败时退回 stderr，由服务管理器收集
- 日志随二进制落位，升级替换二进制后同名文件直接续写，不会丢历史
- Windows 下新建日志文件会写入 UTF-8 BOM，记事本可直接打开；Linux 不写 BOM。升级前生成的旧文件没有 BOM，记事本仍可能显示乱码，可用 VS Code / Notepad++ 查看，或删除后重新生成

## 采集说明

本节提到的 `includeKeywords`、`excludeKeywords`、`artifactCheck`、`artifactMaxDepth`、`manualTriggerMinutes` 等，都是**服务端下发的采集配置字段**（管理端「Agent 管理 → 采集配置」），不在本地 `backup-agent.json` 里；服务端未下发时使用内置默认值。

### Linux crontab

读取当前用户 `crontab -l`、`/etc/crontab` 与 `/etc/cron.d/*`，支持 `@daily`、`@hourly`、`@weekly`、`@monthly`、`@yearly`、`@reboot` 等宏。命中 `includeKeywords` / `includePaths` 且未命中 `excludeKeywords` 的任务会被采集。

在计划行上方添加标签可强制纳入并覆盖任务名、稳定 key 与产物信息：

```cron
# backup-monitor: name=每日数据库备份;key=db-daily;status=/data/backup/db.status;artifact=/data/backup
0 23 * * * /opt/backup/db.sh
```

标签值支持 `;`、`,` 或空格分隔的键值对，值中带空格时用引号包裹：

```cron
# backup-monitor: name="每日 数据库备份" key=db-daily status=/data/backup/db.status
0 23 * * * /opt/backup/db.sh
```

| 键 | 说明 |
|:--|:--|
| `name` | 任务名 |
| `key` | 稳定任务标识，改名后仍能匹配同一任务 |
| `script` | 脚本路径，用于状态文件与进程探测 |
| `status` / `statusFile` | 显式状态文件路径 |
| `artifact` / `path` / `dir` | 备份产物文件或目录 |
| `proc` / `process` | 进程关键字；命中本机进程时该任务判为运行中，留空则不探测 |

`proc` 关键字在 Linux 上匹配完整命令行（`/proc/<pid>/cmdline`），Windows 上匹配进程名（`tasklist`）。判定优先级：状态文件已给出成功或失败时以状态文件为准；状态文件无结论且进程命中时判为运行中；进程未命中则不影响原有判定。命中进程但既无状态文件也无计划时刻时不产出执行记录。

内联标签同样支持：

```cron
15 3 * * * /opt/backup/inline.sh # backup-monitor: inline-task
```

未写标签时任务 key 为 `cron:<user>:<cron 表达式>`；写标签时以标签 `key` 为准。

### Windows 计划任务

优先通过 PowerShell 的 `Get-ScheduledTask` + `Get-ScheduledTaskInfo` + `ConvertTo-Json` 采集，失败时回退 `schtasks /query /fo CSV /v`；只有 PowerShell 2.0、且 schtasks 报「无法加载列资源」的 Windows 7 / Server 2008 R2 会再回退到 COM `Schedule.Service`。支持 `Daily`、`Weekly`、`Monthly`、`Time`、`Boot`、`Logon` 触发器，并尽量转换为 cron 表达式；无法表达的计划保留可读描述，`scheduleCron` 为空。

任务描述（`Comment`/`Description`）中同样支持 `# backup-monitor: ...` 标签。默认无标签时按关键字、脚本路径与排除关键字过滤。

### 状态文件

任务配置 `status` / `statusFile` 后，Agent 按行解析（`=` 或 `:` 分隔）：

```text
start=2026-10-20 15:00:01
end=2026-10-20 15:08:35
size=1.2 GB
transferred=1.1 GB
exit=0
status=success
```

支持 `start`、`end`、`time`、`size`、`transferred`、`exit`、`status` 等键，大小支持 B/KB/MB/GB/TB 与二进制单位。

### 备份产物

`artifactCheck=true` 且任务配置了产物路径时，Agent 检查文件 mtime 与大小，生成执行记录：

- 状态文件执行记录：追加 `；产物 <path>（<size>）`
- 无状态文件的产物回退记录：`检测到备份产物：<path>`

目录产物会按 `artifactMaxDepth` 递归查找最新文件，并忽略 `.status`、`.tmp`、`.part` 与 `veeam-status.txt`。

### Veeam

Linux/Windows 下查找 `veeamconfig`（`/usr/local/sbin`、`/usr/sbin`、`/opt/veeam`、`PATH` 及 Windows 安装目录），解析 `job list`、`session list`、`session info` 表格。未安装时静默跳过，不影响其它任务。

Windows 上没有 `veeamconfig` 时，会额外读取 `veeam-status.txt`：

- `%ProgramData%\Veeam\Backup\veeam-status.txt`
- `%ProgramData%\Veeam\Endpoint Backup\veeam-status.txt`
- 可执行文件同目录 `veeam-status.txt`

### 状态与触发语义

- 退出码 `0` 为 `success`
- Windows `0x41301`（267009）为 `running`
- Windows `0x41303`（267011，尚未运行）不生成执行记录
- Windows `0x41306`（267014，被终止）及其它非零码为 `failed`
- 开始时间偏离最近计划时刻超过 `manualTriggerMinutes` 时，`trigger` 为 `manual`，否则为 `scheduled`
- `scheduleCron` 的最近计划时刻回溯窗口为 7 天，与服务端判定保持一致

## 自更新

WebSocket 收到 `update` 命令时：

1. 先回执 `update_result`，`message` 为「开始更新」
2. 下载服务端二进制到临时文件
3. 校验响应头与文件 SHA-256
4. 备份旧二进制为 `<exe>.bak`（新版本首次上报成功后才删除）
5. 替换当前二进制
6. 按运行环境重启：提交重启失败时还原 `.bak` 并返回错误；Linux systemd 的重启是异步提交的，新版本起不来需手工回滚，Windows 由守护子进程校验启动结果并自动回滚，详见「回滚」

| 运行方式 | 重启做法 |
|:--|:--|
| Linux systemd 服务 | 按 `ExecStart` 匹配单元名（可用 `BM_SERVICE_NAME` 覆盖），用 `systemd-run` 在独立单元中执行 `systemctl restart --no-block`，避免重启命令随本单元一起被 systemd 停掉；`systemd-run` 不可用时直接调用，命令被信号终止按“重启已提交”处理 |
| Linux 非服务 / 未装单元 | `exec` 原地替换进程，PID 不变 |
| Windows 服务 | 由旧版本二进制拉起重启守护子进程：等旧进程退出 → 等 SCM 报告 `STOPPED` → 启动 → 校验 `RUNNING`，失败自动还原 `.bak` 并重启旧版本，过程写入可执行文件同目录的 `update.log`；守护进程拉不起来时回退为延时脚本（`sc stop` → 轮询 `STOPPED` → `sc start`） |
| Windows 非服务 | detached 启动同参数新进程，新进程等待旧 PID 退出后继续 |

**未安装服务时**：Linux 以 `-service` 裸跑或跑在容器里，`systemctl restart` 会因单元不存在失败，Agent 自动回退为直接重启进程（日志「未安装系统服务，改为直接重启进程」），不会因此回滚版本；Windows 未注册服务时不会进入更新流程，`-service` 在启动阶段即退出。

**自动检查更新**由服务端下发的采集配置项 `autoUpdateMinutes` 控制（不是本地 JSON 字段）：它在管理端「Agent 管理 → 采集配置」里设置，`> 0` 时 Agent 按该间隔查询 `/api/agent/binary/latest?platform=<linux|windows>`，发现更高版本后执行同样的更新流程；`0` 表示关闭自动更新，仅响应服务端推送。修改该间隔后（含由 `0` 改为非 `0`）Agent 会从修改时刻重新计时，满一个新闻隔后才执行检查。`platform` 由 Agent 按运行平台自动带上。

**失败退避**：自动更新失败后会把「目标版本 + 失败时间」写入 `<exe>.update-state`，同一版本在 30 分钟内不再自动重试；手动推送不受退避限制，更新成功后该记录自动清除。

手动推送不做版本比较，目标版本与当前相同也会重新下发并重启；自动检查只在托管版本更高时触发。

## 升级与回滚

### 升级

两种方式，效果相同：

| 方式 | 做法 |
|:--|:--|
| 服务端推送 | 管理端「Agent 更新」上传新二进制后点「推送更新」，Agent 收到 `update` 命令自动完成下载、校验、替换与重启 |
| 自动检查 | 采集配置中开启 `autoUpdateMinutes`，Agent 按间隔自行查询并升级 |

手动升级（无法连服务端推送时）：停服务 → 用新二进制覆盖原文件 → 启动服务。路径必须是服务注册时的那一个，否则服务仍启动旧文件。

`-legacy` 包可直接上传到管理端的 Windows amd64 legacy 槽位（服务端按 Go 版本自动归档），不会覆盖标准包；推送时 Windows 7 / Server 2008 R2 主机会自动选中它。若走手工升级，停服务后覆盖对应架构的 legacy 产物即可。

```bash
# Linux
sudo systemctl stop backup-agent
sudo install -m 0755 backup-agent /opt/backup-agent/backup-agent
sudo systemctl start backup-agent
sudo systemctl status backup-agent
```

```powershell
# Windows（管理员）
Stop-Service backup-agent
Copy-Item .\backup-agent.exe C:\ProgramData\backup-agent\backup-agent.exe -Force
Start-Service backup-agent
```

### 回滚

自更新替换前会把旧二进制改名为 `<exe>.bak`（如 `/opt/backup-agent/backup-agent.bak`、`C:\ProgramData\backup-agent\backup-agent.exe.bak`），且：

- 新二进制替换失败时自动把 `.bak` 还原
- 重启失败时：Linux 提交重启命令失败会还原 `.bak`，新版本起不来需按下文手工回滚；Windows 由守护子进程校验启动结果并自动还原
- 新版本首次上报成功后才删除 `.bak`；起不来或启动后崩溃时该文件会保留，正好用于手工回滚

因此新版本起不来时，把 `.bak` 改回原文件名并重启服务即可：

```bash
# Linux
sudo systemctl stop backup-agent
sudo mv /opt/backup-agent/backup-agent.bak /opt/backup-agent/backup-agent
sudo systemctl start backup-agent
```

```powershell
# Windows（管理员）
Stop-Service backup-agent
Move-Item C:\ProgramData\backup-agent\backup-agent.exe.bak C:\ProgramData\backup-agent\backup-agent.exe -Force
Start-Service backup-agent
```

回滚后建议在管理端把该平台托管的二进制改回旧版本，避免自动检查再次把主机升级回去。

## 排障

日志是第一落点：默认在二进制同目录 `agent.log`，服务模式下不写 stdout。

**启动即退出**

前台执行 `./backup-agent -once` 可直接看到错误。配置加载阶段的错误只写 stderr（此时日志文件还没打开），运行期的错误才进 `agent.log`。常见原因：

- `server URL is required` — 配置没读到，或 `serverUrl` 为空
- `invalid BM_LOG_MAX_MB: "abc" is not an integer` — 环境变量不是整数
- `invalid role "xxx"` — 角色不在允许集合内
- 配置没读到最常见的两种情况：服务模式只按默认路径查找（见「安装为服务」），或 `go run` 时二进制在临时目录

**启动即崩溃 0xc0000005**

进程还没进入 `main` 就退出，控制台或事件查看器可见 `Exception 0xc0000005`、`PC=0x0`，栈顶为 `internal/runtime/syscall/windows_amd64.s`。这是 Go runtime 与系统版本不匹配：常规包要求 Windows 10 / Server 2016 及以上，Windows 7 / Server 2008 R2 需改用 `-legacy` 包。

**主机不上线**

1. Agent 进程是否在跑：`systemctl status backup-agent` / `Get-Service backup-agent`
2. `agent.log` 里是否有连接失败、401/403、配置拉取失败
3. `serverUrl` 从该主机是否可达（`curl <serverUrl>/api/health`）
4. 启用令牌校验时，`token` 要与「Agent 管理 → Agent 令牌」里的一致

**服务装不上**

- Linux 需要 root（`sudo`），Windows 需要管理员 PowerShell
- 服务已存在时会明确提示，先 `-uninstall` 再装
- 容器里没有 systemd，`-install` 不适用，直接把 `-service` 交给容器编排常驻

**改了配置不生效**

服务以 `-service` 启动且不带 `-config`，配置文件必须落在默认查找位置（二进制同目录或系统默认目录）；改完需要重启服务：`systemctl restart backup-agent` / `Restart-Service backup-agent`。

## 相关文档

| 文档 | 内容 |
|:--|:--|
| [docs/agent-protocol.md](../docs/agent-protocol.md) | 上报、配置拉取、WebSocket 下行与回执的完整字段 |
| [docs/manual.md](../docs/manual.md) | 第 2.4 节本地调试、第 4.3 节生产部署与 systemd 示例 |
| [README.md](../README.md) | 项目整体介绍、部署方式与常见问题 |

## 已知限制与风险

- Windows 的 `set_cron` 暂不支持回写计划任务，会返回明确的中文提示；Linux 支持当前用户 crontab、`/etc/crontab` 与 `/etc/cron.d/*`
- `/etc/crontab` 与 `/etc/cron.d/*` 回写需要 root 权限，且原子替换可能改变文件的 SELinux 上下文
- `schtasks` CSV 回退输出受系统代码页影响，中文任务名可能乱码；优先使用 PowerShell 路径
- Veeam CLI 输出格式随版本与语言变化，解析器做了表头与位置回退，但复杂表格仍可能识别不到
- Windows 产物与计划任务时间均按本机时区解析，跨时区部署需统一主机时区
- 自动更新依赖服务端托管的二进制与 SHA-256；校验失败会保留旧二进制
