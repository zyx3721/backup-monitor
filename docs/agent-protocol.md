# Agent 通信协议（v2）

本文件是服务端与 Agent 的冻结契约，服务端与 `agent/` 实现必须同时满足。

## 1. 认证

- 配置拉取、上报、二进制下载、二进制版本查询、WebSocket 均通过请求头 `X-Agent-Token` 携带令牌。
- 平台设置 `requireAgentToken=true` 时，缺失或无效令牌一律 `401 {"message":"Agent 令牌无效"}`。
- 令牌关闭时，携带有效令牌仍会刷新 `lastSeenAt`，无效令牌不阻断。

## 2. 上报

`POST /api/ingest/report`

```json
{
  "host": {
    "hostname": "db01",
    "nodeId": "稳定机器标识",
    "ip": "10.0.0.8",
    "os": "linux",
    "arch": "amd64",
    "osBuild": 0,
    "role": "database",
    "agentVersion": "1.0.0",
    "agentStartedAt": "2026-10-30T00:58:28.829824Z"
  },
  "tasks": [
    {
      "name": "mysqldump-daily",
      "key": "cron:root:0 23 * * *",
      "type": "shell",
      "tool": "",
      "schedule": "每天 23:00",
      "scheduleCron": "0 23 * * *",
      "config": {"source": "crontab", "user": "root", "command": "/opt/backup/db.sh", "script": "/opt/backup/db.sh"}
    }
  ],
  "executions": [
    {
      "taskName": "mysqldump-daily",
      "sessionId": "可选稳定会话标识",
      "state": "success",
      "trigger": "scheduled",
      "creationTime": "2026-10-20T15:00:00Z",
      "startTime": "2026-10-20T15:00:01Z",
      "endTime": "2026-10-20T15:08:35Z",
      "processedSize": "1.2 GB",
      "transferredSize": "1.1 GB",
      "artifactPath": "/data/backup/db.tar.gz",
      "exitCode": 0,
      "message": "状态文件：/data/backup/db.status"
    }
  ]
}
```

响应：`{"ok":true,"hostId":"...","newExecutions":3}`

约束：

- `host.hostname`、`host.ip` 必填；`nodeId` 用于主机改名后保持同一条记录。
- `host.arch` 可选，取 `amd64|arm64`；服务端按它筛选推送目标，缺省时该主机不做架构校验（推送结果会标注「未上报架构」）。
- `host.osBuild` 可选，Windows 构建号（如 `7601`、`19045`），Linux 为 `0`；服务端用它校验安装包的 Go 版本是否与该系统兼容，缺省时不做系统版本校验。
- `host.agentStartedAt` 可选，当前 Agent 进程启动时间（RFC3339Nano）；每次进程重启都会变化，服务端用它判断推送后是否真正完成重启，旧版 Agent 缺省时退回按上报时间判断。
- `host.role` 仅在主机首次注册时写入；主机已登记后以管理端配置为准，后续上报不覆盖手动修改的角色。
- `tasks[].type` 取值 `shell|veeam`（服务端还会派生 `powershell|batch|exe`）。
- `executions[].state` 取值 `success|running|failed`；`creationTime` 必须为可解析时间。
- `executions[].trigger` 取值 `scheduled|manual`：按本条执行的开始时间回溯 `scheduleCron` 最近计划时刻，落在 `manualTriggerMinutes` 内为 `scheduled`，否则为 `manual`。
- 单次执行（Shell、状态文件、产物、Windows 计划任务）不产生重试次数，`isLastAttempt=true`，`attempt`/`maxAttempts` 缺省；Veeam 执行从 `veeamconfig session info` 读取 `jobId`、`oibId`、`attempt`、`maxAttempts`、`isLastAttempt`。
- 同一 `taskId+sessionId`（无 sessionId 时用 `taskId+creationTime`）幂等更新。
- 服务端按任务最新执行重算状态、产生/恢复告警并触发通知。

## 3. 配置拉取

`GET /api/agent/config?hostname=<hostname>&platform=<linux|windows>`

```json
{
  "hostname": "db01",
  "serverTime": "2026-10-21T05:00:00Z",
  "settings": {
    "reportTimeoutMinutes": 3,
    "offlineTimeoutMinutes": 6,
    "notRunGraceMinutes": 30,
    "scheduleMatchWindowMinutes": 60,
    "schedulerIntervalSec": 30,
    "dedupeAlerts": true,
    "retentionDays": 90,
    "requireAgentToken": true,
    "agent": {"reportIntervalSec": 600, "httpTimeoutSec": 0, "wsBackoffMinSec": 0, "wsBackoffMaxSec": 0}
  },
  "profile": {
    "id": "...",
    "name": "默认采集配置",
    "hostname": "*",
    "enabled": true,
    "priority": 0,
    "config": {
      "reportIntervalSec": 0,
      "collectCrontab": true,
      "collectVeeam": true,
      "artifactCheck": true,
      "labelPrefix": "# backup-monitor:",
      "includeKeywords": ["backup"],
      "includePaths": ["/data/scripts"],
      "excludeKeywords": ["logrotate"],
      "maxArtifactAgeHours": 30,
      "heartbeatIntervalSec": 30,
      "artifactMaxDepth": 3,
      "maxSessionsPerJob": 5,
      "maxSessionInfoCalls": 10,
      "maxJobInfoCalls": 20,
      "manualTriggerMinutes": 30,
      "artifactPreToleranceMinutes": 10,
      "autoUpdateMinutes": 0
    }
  }
}
```

`profile` 为 `null` 表示没有匹配配置：Agent 仍上报主机心跳，但不上报任务与执行。

上报间隔取值优先级：`profile.config.reportIntervalSec`（非 0）→ `settings.agent.reportIntervalSec` → Agent 本地 `intervalSec` → 600 秒；任何来源低于 10 秒都按 10 秒执行。

`settings.agent.wsBackoffMinSec` / `wsBackoffMaxSec` 是 WebSocket 重连退避的下限与上限，保存时要求上限不小于下限；任一项为 0 时由 Agent 按默认 2 秒 / 30 秒处理。

## 4. WebSocket 心跳与命令

`GET /api/agent/ws?hostname=<hostname>&nodeId=<nodeId>&platform=<linux|windows>`（Upgrade，`X-Agent-Token`）

WebSocket 只接受已登记主机：主机不存在时握手返回 `404`，Agent 按退避重试；待下一轮上报创建主机后再注册。

服务端 → Agent：

```json
{"type":"config_changed","at":"ISO"}
{"type":"set_cron","requestId":"uuid","key":"任务 key","cron":"0 23 * * *"}
{"type":"update","requestId":"uuid","version":"1.0.0","sha256":"...","url":"/api/agent/download?platform=linux"}
```

Agent → 服务端：

```json
{"type":"set_cron_result","requestId":"uuid","ok":true,"message":"已更新当前用户 crontab"}
{"type":"update_result","requestId":"uuid","ok":true,"message":"开始更新"}
```

Agent 每 30 秒发送一次文本 `ping` 或任意消息维持心跳；服务端 90 秒无消息强制断开，断开 15 秒宽限后判离线。

## 5. 二进制

- 托管槽位由 `platform + arch + variant` 确定，共 5 个：`linux/amd64`、`linux/arm64`、`windows/amd64`、`windows/arm64` 为 `standard`，`windows/amd64` 另有 `legacy`（Go 1.20 构建，面向 Windows 7 / Server 2008 R2）。
- `platform` 为必填参数，取值 `linux` 或 `windows`；缺失或非法返回 `400`。
- `GET /api/agent/binary/latest?platform=&arch=&osBuild=` → `{"version":"1.0.0","arch":"amd64","variant":"standard","goVersion":"go1.26.0","sha256":"...","size":123}`；`arch` 与 `osBuild` 可选，缺省按 amd64 选包，无匹配槽位时 404。
- `GET /api/agent/download?platform=&arch=&variant=` → 二进制流，响应头 `X-Agent-Binary-Version`、`X-Agent-Binary-Sha256`、`X-Agent-Binary-Variant`、`X-Agent-Binary-Arch`，需携带 `X-Agent-Token`。只带 `platform` 时按 amd64 标准包选择；也接受 `osBuild` 由服务端选包。
- 公开下载：`GET /download/backup-agent?platform=&arch=&osBuild=` → 同一二进制流，**无需令牌**，供目标主机首次部署时直接用 `curl` / `wget` 拉取安装包。
- 管理端上传：`POST /api/backup/agent-binary?platform=linux|windows&version=1.0.0[&variant=standard|legacy]`（`application/octet-stream`）；服务端校验 ELF/PE、解析架构与内嵌 Go 版本，`variant` 缺省时 Windows amd64 且 Go < 1.21 自动归入 `legacy`。
- 管理端推送：`PUT /api/backup/agent-update/push` 逐台按主机 `arch` 与 `osBuild` 选包：Windows Build ≥ 10240 取标准包（缺失即跳过，不回退 legacy），Build < 10240 取 legacy 包；结果逐台返回 `version` / `arch` / `variant`，无匹配包的主机在 `results[].reason` 说明原因；`results[].agentStartedAt` 是推送时的进程启动时间，管理端据此判断是否真正完成重启。
- 未带 `arch` / `osBuild` 的旧版 Agent 仍可自动更新，服务端按 amd64 标准包选择，结果标注「未校验」。

## 6. 本地配置

Agent 读取顺序：命令行参数 > 环境变量 > 配置文件 > 默认值。

配置文件默认路径：可执行文件同目录 `backup-agent.json`，找不到时回退 `/etc/backup-agent/backup-agent.json`（Linux）。

```json
{
  "serverUrl": "http://backup.its.sunline.cn:3007",
  "token": "",
  "hostname": "",
  "intervalSec": 0,
  "role": "database",
  "logFile": ""
}
```

环境变量：`BM_SERVER_URL`、`BM_AGENT_TOKEN`、`BM_HOSTNAME`、`BM_ROLE`、`BM_CONFIG_FILE`。

命令行：`-server -token -hostname -role -interval -config -log -once -service -install -uninstall -version`。

`role` 只在主机首次注册时写入服务端；主机登记后角色以管理端配置为准，Agent 后续上报不会覆盖。

## 7. 判定语义（服务端）

- `reportTimeoutMinutes` 内无上报 → `abnormal`；超过 `offlineTimeoutMinutes` → `offline`。
- 计划任务：最近一次计划时刻 + 宽限期后，若窗口 `[计划-10min, 下一次计划)` 内无执行记录 → `not_run`。
- 执行失败 → `backup_failed`（critical）；恢复成功 → 自动恢复 `backup_failed/not_run`。
- 告警去重：同 scope 同类型仅保留一条 `active`；关闭去重时仍有 1 小时抑制窗口。
