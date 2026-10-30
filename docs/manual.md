# Backup Monitor — 完整手册

> 备份监控平台（Go + React + 主机 Agent）。本手册面向部署、运维与二次开发者，覆盖本地开发、生产部署、备份监控业务、Agent 协议、接口清单、数据库表说明与常见问题。

## 目录

- [一、项目介绍](#一项目介绍)
- [二、本地开发快速启动](#二本地开发快速启动)
- [三、Docker Compose 快速部署（推荐）](#三docker-compose-快速部署推荐)
- [四、生产环境部署](#四生产环境部署)
- [五、API 文档](#五api-文档)
- [六、数据库说明](#六数据库说明)
- [七、常见问题](#七常见问题)
- [八、安全建议](#八安全建议)
- [九、许可证](#九许可证)
- [十、致谢](#十致谢)
- [十一、联系方式](#十一联系方式)
- [十二、备份监控业务与 Agent](#十二备份监控业务与-agent)

# 一、项目介绍

## 1.1 项目简介

Backup Monitor 是面向 Linux / Windows 主机的自托管备份监控平台：Agent 采集 crontab、计划任务与 Veeam 备份作业，服务端统一判定漏执行、连续失败与主机失联，并按渠道发送告警。在备份监控业务之外，平台仍保留一套完整的通用底座能力：

- **备份监控**：主机、任务、执行记录、告警、通知渠道、采集配置、Agent 令牌与二进制分发，以及概览趋势与状态分布
- **认证与会话**：本地密码（bcrypt）、AD/LDAP、企业微信扫码登录（直连与统一认证中心两种模式）、JWT 会话（密钥自动生成并持久化到数据库）、修改密码、退出登录、密码登录失败锁定
- **找回密码**：图形验证码 → 身份校验 → 邮件验证码（限频与冷却）→ 重置密码，全流程时效可调
- **审计日志**：登录 / 登出 / 改密 / 配置变更等显式事件写入，支持筛选、清空与前端导出
- **用户与权限**：用户、用户群组、角色与权限管理（23 项权限，按侧边栏集合划分，manage 隐含 read）
- **系统配置**：基础配置（品牌标识、安全时效、判定与保留、Agent 通道、安全与告警）、认证配置（AD/LDAP 与企业微信及连通性测试）、找密配置（找回密码邮件 SMTP 与测试发信）
- **基础设施**：健康检查、Swagger API 文档、CORS、请求超时、SQLite（WAL）单文件库

后端使用纯 Go 的 SQLite 驱动，**不需要 CGO、不需要 GCC、不需要外部数据库**；前端是 React 19 控制台，构建后由 Nitro 提供 SSR。

## 1.2 核心功能

| 能力 | 说明 |
| --- | --- |
| 登录方式 | 本地密码（bcrypt）、AD/LDAP、企业微信扫码（直连 / 统一认证中心 SSO） |
| 会话 | JWT（golang-jwt/v5）+ 服务端会话表 `sessions`，默认 12 小时，`SESSION_TTL_HOURS` 可调；密钥留空自动生成并持久化，注销即吊销对应令牌 |
| 登录失败锁定 | 默认连续 5 次锁 2 分钟，可在安全时效配置 3-10 次 / 1-10 分钟，内置管理员不受限 |
| 找回密码 | 图形验证码 → 身份校验 → 邮件验证码（发送冷却与限频）→ 重置密码，各环节时效可调 |
| 权限模型 | 23 项权限按 `模块.资源.操作` 成对设计，`manage` 隐含 `read`；按侧边栏集合划分，备份监控 13 项、审计 2 项、系统配置 8 项；用户直接授权 + 群组授权 |
| 审计日志 | 模块 / 操作 / 目标 / 详情 / 结果全量记录，支持筛选、清空与前端导出（xlsx/csv 等） |
| 系统配置 | 基础配置（品牌标识、安全时效、判定与保留、Agent 通道、安全与告警）、用户配置、认证配置（LDAP / 企业微信）、找密配置（邮件媒介）四组，按权限可见 |
| 品牌标识 | 站点名 / 登录页名称 / 控制台名称 / 图标，SSR 阶段实时拉取直出首帧 |
| API 文档 | Swagger UI 在线查看，`swag init` 自动生成 |

## 1.3 数据与安全边界

- SQLite 单文件数据库是业务数据的事实来源，默认路径 `backend/data/backup_monitor.db`，运行日志在 `data/logs`。
- JWT 签名密钥持久化在 `system_secrets` 表；LDAP 绑定密码、SMTP 密码与企业微信 Secret 以 AES-GCM 加密后落库，接口返回时脱敏。
- 只有健康检查、登录、公开认证方式、企微回调、找回密码流程与品牌信息接口可匿名访问，其余接口一律要求 `Authorization: Bearer <token>`。
- 仓库禁止提交 Token、密码与真实业务数据；`.env` 已在 `.gitignore` 中排除。

## 1.4 技术栈

### 1.4.1 后端

| 项 | 选型 |
| --- | --- |
| 语言 | Go 1.25+ |
| HTTP 路由 | [go-chi/chi](https://github.com/go-chi/chi) v5 |
| 数据库 | SQLite（[modernc.org/sqlite](https://gitlab.com/cznic/sqlite) 纯 Go 驱动，连接固定 `MaxOpenConns(1)`） |
| 认证与加密 | JWT（golang-jwt/v5）、AD/LDAP（go-ldap/ldap v3）、企业微信 OAuth 扫码登录（直连 / 统一认证中心 SSO）、bcrypt 口令散列、AES 敏感配置加密 |
| API 文档 | swag + http-swagger（Swagger UI） |

### 1.4.2 前端

| 项 | 选型 |
| --- | --- |
| 框架 | React 19 + TanStack Start（SSR）/ Router / Query |
| 语言与构建 | TypeScript 5 + Vite 7 + Nitro |
| 样式与组件 | Tailwind CSS v4、Radix UI、lucide-react、sonner |
| 导出 | SheetJS（审计日志 xlsx/csv 等） |

## 1.5 项目结构

```text
backup-monitor/
├── backend/                       Go 后端服务（HTTP API 与巡检调度）
│   ├── cmd/server/                进程入口：命令行参数、配置加载与启动（含 Swagger 主注解）
│   ├── config/                    SERVER_ADDR / DB_PATH / JWT_SECRET 等环境变量与 .env 加载，输出类型化 Config
│   ├── docs/                      Swagger 生成产物（docs.go / swagger.json / swagger.yaml，勿手改）
│   ├── internal/
│   │   ├── buildinfo/             编译期 Version / Commit / BuildDate（CI 以 -ldflags 注入）
│   │   ├── common/                跨模块工具与时间归一化；localizer/ 做错误文案中文化
│   │   ├── domain/                请求载荷与校验规则（auth / user / base / backup）
│   │   ├── repository/            契约与共享实现；backup_*.go 为备份域建表与各表 SQL
│   │   ├── security/              password.go bcrypt 散列、secret.go 设置项 AES-GCM 加解密
│   │   └── service/               工作流与备份编排：auth/user、agent_hub 长连接、backup_* 告警巡检
│   ├── router/
│   │   ├── app.go                 App 结构体、JWT 密钥解析与 Run() 启动
│   │   ├── routes.go              chi 路由总注册：全局中间件与各域子路由挂载
│   │   ├── middleware/            JWT 认证与权限校验中间件
│   │   ├── common/                响应与 IP、DecodeJSON、认证上下文、SQLite 建立与迁移
│   │   ├── auth/                  认证域：登录与限流、找回密码、改密、企业微信
│   │   ├── backup/                备份域：agent_api 上报下载、manage 主机任务、settings* 参数审计
│   │   ├── settings/              配置域：基础配置、用户/角色/群组、认证、邮件；含权限目录与建表
│   │   ├── system/                系统域：审计日志、导出、建库与迁移编排
│   │   └── swagger/               OpenAPI 注解与文档模型（含备份监控域）
│   ├── data/                      SQLite 数据库与日志目录（运行时生成）
│   ├── .air.toml                  热重载配置（air）
│   └── .env.example               环境变量模板
├── agent/                         Go 采集端（独立二进制与 go.mod）
│   ├── cmd/backup-agent/          Agent 入口：参数解析、运行模式与版本标记
│   ├── internal/
│   │   ├── api/                   与服务端交互：拉配置、上报结果、下载新版本
│   │   ├── bootstrap/             按参数 → 环境变量 → 配置文件 → 默认值装配本地配置
│   │   ├── collect/               采集计划任务、状态文件、备份产物与 Veeam 会话
│   │   ├── config/                本地配置结构、角色枚举与共享默认值
│   │   ├── cron/                  5 段 cron 解析与最近计划时刻
│   │   ├── discover/              发现并解析 Linux crontab / Windows 计划任务
│   │   ├── identity/              稳定主机标识，缺失时生成 UUID 并持久化
│   │   ├── logging/               按大小轮转的文件日志
│   │   ├── proc/                  关键字探测本机进程，判断任务是否仍在运行
│   │   ├── service/               安装与管理 systemd / Windows 服务
│   │   ├── statusfile/            解析备份状态文件的时间、大小与退出码
│   │   ├── sysinfo/               主机名、首选内网地址与操作系统名称
│   │   ├── update/                自更新、版本比较与重启
│   │   ├── veeam/                 调用 veeamconfig 采集作业与会话
│   │   └── wshb/                  WebSocket 长连接、心跳与指数退避重连
│   ├── backup-agent.example.json  本地配置模板
│   └── README.md                  Agent 部署与排障说明
├── frontend/                      React 控制台
│   ├── public/                    静态资源（favicon 等）
│   └── src/
│       ├── routes/                文件路由：根布局、登录态布局与各业务页、公开页
│       ├── features/              业务域：auth、agents、backup、audit、settings（含 panels/、dialogs/）
│       ├── components/            跨域复用：layout/、ui/、data/、dialogs/、feedback/
│       ├── lib/                   跨页面能力层：API 传输、会话、权限、品牌、导出、导航与刷新
│       ├── router.tsx             TanStack Router 实例
│       ├── routeTree.gen.ts       路由树（插件生成，禁止手改）
│       ├── server.ts              SSR 服务端入口
│       ├── start.ts               客户端入口
│       └── styles.css             全局设计系统（bm-* 前缀、明暗主题）
├── deploy/                        Dockerfile、docker-compose、entrypoint、nginx、supervisord
├── docs/                          本手册与 Agent 协议、数据库设计等文档
├── scripts/                       前后端运行脚本、升级脚本与 Veeam 备份触发脚本
└── .github/workflows/             CI/CD 工作流（推送 v* Tag 自动构建打包发布）
```

# 二、本地开发快速启动

## 2.1 环境要求

- Go 1.25+（后端）
- Node.js 22+（前端）

> 后端使用纯 Go SQLite 驱动（`modernc.org/sqlite`），无需安装 GCC 或 CGO 环境。

## 2.2 克隆项目

```bash
git clone https://github.com/zyx3721/backup-monitor.git
cd backup-monitor
```

## 2.3 后端配置与启动

1. 进入后端目录下载依赖：

```bash
cd backend
go mod download
```

2. 配置环境变量：

```bash
# 步骤1：复制模板文件
cp .env.example .env

# 步骤2：编辑 .env，按实际环境修改监听地址、密钥等信息
vim .env
# 后端监听地址
SERVER_ADDR=127.0.0.1:8080

# SQLite 数据库文件路径
DB_PATH=./data/backup_monitor.db

# 鉴权与接口行为
# 留空则启动时自动生成随机密钥并持久化到数据库（删除数据库后所有会话自动失效）
JWT_SECRET=
SESSION_TTL_HOURS=12
CORS_ORIGINS=*
```

**配置参数说明**：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | 后端监听地址；未设置时退回 `ADDR`，再按 `127.0.0.1:$PORT` 组装（`PORT` 默认 `8080`） |
| `DB_PATH` | `./data/backup_monitor.db` | SQLite 数据库文件路径，相对后端运行目录 |
| `JWT_SECRET` | 空 | JWT 签名密钥；留空则首次启动自动生成随机密钥并持久化到数据库，建议显式设置 |
| `SESSION_TTL_HOURS` | `12` | 登录会话（JWT）有效期，单位小时，正整数 |
| `CORS_ORIGINS` | `*` | 允许的跨域来源，多个用逗号分隔 |

后端启动时自动按自定义 `-env` 路径 → `.env` → `backend/.env` → 可执行文件同目录 `.env` 的顺序读取首个存在的文件（环境变量中已显式设置的项优先）。

后端同时支持命令行参数，显式传入的参数优先于环境变量与 `.env` 文件；`./backup-monitor -v` 可查看版本信息（版本、commit、构建时间），`./backup-monitor -h` 查看全部参数：

| 参数 | 等价环境变量 | 说明 |
| --- | --- | --- |
| `-addr` | `SERVER_ADDR` | 后端监听地址 |
| `-port` | `PORT` | 后端监听端口，未设置 `-addr` 时生效 |
| `-db` | `DB_PATH` | SQLite 数据库路径 |
| `-jwt-secret` | `JWT_SECRET` | JWT 签名密钥 |
| `-session-ttl` | `SESSION_TTL_HOURS` | 登录会话有效期（小时） |
| `-cors-origins` | `CORS_ORIGINS` | 允许的跨域来源，多个用逗号分隔 |
| `-env` | 无 | 指定 `.env` 配置文件路径 |
| `-v`、`-version` | 无 | 显示版本信息并退出 |

3. 运行后端服务：

```bash
# 方式1：前台运行（终端关闭则服务停止）
go run cmd/server/main.go

# 方式2：后台运行（日志输出到 app.log）
nohup go run cmd/server/main.go > app.log 2>&1 &
```

后端服务默认运行在 `http://localhost:8080` ，如需指定地址和端口，可修改环境变量文件内的 `SERVER_ADDR` 参数，或直接使用 `-addr` 命令行参数（优先级高于环境变量）。首次启动会自动创建数据库和默认管理员账户 `admin / admin123` 。

## 2.4 Agent 配置与启动

**注：这一步只在备份服务器上，并只需配置启动 Agent。**

1. 在备份服务器上进入 Agent 目录，下载相关依赖：

```bash
cd agent
go mod download
```

2. 创建 `backup-agent.json` 文件配置信息：

```bash
vim backup-agent.json
# Agent 配置
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

**配置参数说明**：

| 字段 | 说明 | 默认值 | 等价环境变量 |
|:--|:--|:--|:--|
| `serverUrl` | 服务端地址，必须是 http/https 绝对地址 | 空 | `BM_SERVER_URL` |
| `token` | Agent 接入令牌，对应请求头 `X-Agent-Token` | 空 | `BM_AGENT_TOKEN` |
| `hostname` | 上报主机名，留空使用系统主机名 | 空 | `BM_HOSTNAME` |
| `intervalSec` | 本地上报间隔兜底值（秒）；0 表示完全使用服务端配置，1～9 会按最小值 10 处理 | 600 | `BM_INTERVAL_SEC` |
| `role` | 主机角色：`database`、`web`、`application`、`file`、`virtualization`，非法值回落 `database`；仅首次注册写入，之后以管理端配置为准 | `database` | `BM_ROLE` |
| `logFile` | 日志文件路径，留空写入二进制同目录 `agent.log` | 同目录 `agent.log` | `BM_LOG_FILE` |
| `logMaxMb` | 单个日志文件上限（MB），0 或留空用默认值，负数启动失败 | 10 | `BM_LOG_MAX_MB` |
| `logBackups` | 保留的历史日志文件个数，0 或留空用默认值，负数启动失败 | 2（共 3 个文件） | `BM_LOG_BACKUPS` |

配置加载优先级为**命令行参数 > 环境变量 > JSON 配置文件 > 默认值**。配置文件查找顺序：`-config` 指定路径 → 环境变量 `BM_CONFIG_FILE` → 可执行文件同目录 `backup-agent.json` → Linux `/etc/backup-agent/backup-agent.json`（Windows 为 `%ProgramData%\backup-agent\backup-agent.json`）。

3. 运行 Agent 服务：

```bash
# 本地开发：显式指定当前目录的配置文件
go run ./cmd/backup-agent --config backup-agent.json

# 兼容旧版写法，run 子命令会被自动忽略（编译出二进制后再运行）
go build -o backup-agent ./cmd/backup-agent
./backup-agent run --config backup-agent.json
```

> **说明**：`go run` 编译出的临时可执行文件在系统临时目录下，因此「可执行文件同目录 `backup-agent.json`」这条默认查找**在 `go run` 下不生效**，本地开发必须用 `--config` 或环境变量 `BM_CONFIG_FILE` 指定配置；编译成二进制并放在配置文件同目录后，才可以省掉 `--config`。
>
> **注意**：`go run ./cmd/backup-agent` 指向**包目录**，会编译该目录下全部源文件（含 `version_marker.go` 的版本标记）；`go run ./cmd/backup-agent/main.go` 只编译单个文件，虽能启动，但产物不含版本标记，上传到管理端时无法自动识别版本。对外发布的二进制请始终使用包目录构建，详见 [4.3](#43-agent-构建与配置)。
>
> 日志默认写入二进制同目录的 `agent.log`，轮转上限与保留个数可调：`-log-max-mb`（默认 10 MB）、`-log-backups`（默认 2，即共 3 个文件），对应环境变量 `BM_LOG_MAX_MB`、`BM_LOG_BACKUPS`；日志路径可用 `-log` 或 `logFile` 覆盖。需要安装为系统服务时用 `-install`，见 [4.3](#43-agent-构建与配置)。
>

## 2.5 前端配置与启动

1. 进入前端目录下载相关依赖：

```bash
cd frontend
npm install
```

2. 配置 API 地址（可选）：

```bash
# 配置说明：
# - 后端端口 = 8080：无需创建 .env 文件（默认值为 http://127.0.0.1:8080）
# - 后端端口 ≠ 8080：需要创建 .env 文件（指定正确端口，例如后端端口改为 8090）
#   创建 .env 文件，例如：
echo "VITE_API_BASE_URL=http://localhost:8080" > .env
```

3. 启动前端服务：

```bash
# 方式1：前台运行（终端关闭则服务停止）
npm run dev
# 如果要指定外部访问和监听端口，可执行例如：
npm run dev -- --host --port 5173

# 方式2：后台运行（日志输出到 bm-frontend.log）
nohup npm run dev > bm-frontend.log 2>&1 &
```

前端服务默认运行在 `http://localhost:5173/` 。

## 2.6 访问系统

- **首页**：`http://localhost:5173`
  - **默认用户名**：`admin`
  - **默认密码**：`admin123`
- **API 文档**：`http://localhost:8080/swagger/index.html`
- **健康检查**：`http://localhost:8080/health`

# 三、Docker Compose 快速部署（推荐）

## 3.1 部署目录结构

Docker Compose 部署相关文件统一放在 `deploy/` 目录下。`backup-monitor` 单镜像内包含 Go 后端、Nginx 和前端 Nitro SSR 服务，并通过 Supervisor 管理多进程。

仓库内置文件结构：

```bash
deploy/
├── Dockerfile            # 多阶段镜像构建：前端构建、后端构建、运行时镜像
├── docker-compose.yml    # 服务编排配置
├── entrypoint.sh         # 容器启动入口，交给 Supervisor 拉起各进程
├── nginx.conf            # 容器内 Nginx 配置，负责静态资源、API 与页面反代
├── supervisord.conf      # 容器内多进程管理配置
└── .env.example          # 环境变量模板
```

首次部署时需要从 `.env.example` 复制生成 `.env`，运行后会在 `deploy/` 下生成持久化目录：

```bash
deploy/
├── .env                  # 实际环境变量文件
├── data/                 # 应用数据挂载目录
│   ├── backup_monitor.db # SQLite 数据库（首次启动自动创建）
│   └── logs/             # 运行日志
```

镜像构建时会分别生成前端 `.output` 产物和后端二进制；运行时由 Supervisor 同时管理 Go 后端、Nginx 和前端 Nitro SSR 服务。

运行时只复制前端 `.output` 产物，并在 `/app/frontend` 执行 `node .output/server/index.mjs`。Nginx 直接托管 `.output/public/assets` 等静态资源，并将 `/api/`、`/api/events` 和 `/swagger/` 反向代理到后端。

## 3.2 准备配置文件

进入 `deploy` 目录，创建 `.env` 环境变量文件：

```bash
cd deploy
vim .env
```

`.env` 文件内容参考：

```bash
# 鉴权与接口行为
JWT_SECRET=backup-monitor-change-me
CORS_ORIGINS=*
VITE_API_BASE_URL=http://127.0.0.1:8080
```

**配置参数说明详情见 [2.3](#23-后端配置与启动)。** 容器内建议显式设置 `JWT_SECRET` 为足够随机的长字符串。

## 3.3 构建镜像（可选）

如果不想使用阿里云镜像仓库的镜像，可直接在本地手动构建（默认使用阿里云镜像仓库地址）：

```bash
# 在 deploy/ 目录下构建（构建上下文为项目根目录）
cd deploy
docker build \
  -f Dockerfile \
  -t backup-monitor:latest \
  --build-arg ALPINE_MIRROR=mirrors.aliyun.com \
  ..
```

然后修改 `deploy/docker-compose.yml` 中 `backup-monitor` 服务的 `image` 字段为 `backup-monitor:latest` 。

常用构建参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `ALPINE_MIRROR` | 空 | Alpine apk 镜像源（如 `mirrors.aliyun.com`） |
| `NPM_REGISTRY` | `https://registry.npmmirror.com` | 前端 npm 镜像源 |
| `GOPROXY` | `https://goproxy.cn,direct` | Go 模块代理 |
| `APP_VERSION` / `APP_COMMIT` / `APP_BUILD_DATE` | `dev` / `unknown` | 版本信息注入（CI 构建时传入） |

## 3.4 启动服务

```bash
cd deploy
docker compose up -d
```

compose 文件默认将 `80` 端口映射到宿主机，并把 `./data` 挂载到容器内 `/app/data`；如需修改端口或数据目录，编辑 `docker-compose.yml` 后重新执行 `docker compose up -d`。

## 3.5 服务管理

```bash
# 查看服务状态
docker compose ps

# 查看实时日志
docker compose logs -f

# 重启服务
docker compose restart

# 停止所有服务
docker compose down

# 停止并删除数据卷（谨慎！数据会丢失）
docker compose down -v
```

## 3.6 访问系统

服务启动后，访问以下地址：

- **首页**：`http://your-domain.com`
  - **默认用户名**：`admin`
  - **默认密码**：`admin123`
- **API 文档**：`http://your-domain.com/swagger/index.html`
- **健康检查**：`https://your-domain.com/health`

## 3.7 宿主机 Nginx 反代（可选）

如需通过宿主机 Nginx 统一配置公网域名、HTTPS 证书或多站点入口，可将 `deploy/docker-compose.yml` 中的端口映射改为非 80 端口（如 `8080:80`），再由宿主机 Nginx 反向代理到容器内 Nginx。

此时请求链路为：

```text
浏览器 -> 宿主机 Nginx -> backup-monitor 容器内 Nginx -> Go 后端 / 前端 SSR 服务
```

### 3.7.1 HTTP 示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 限制上传文件大小（可选）
    client_max_body_size 500m;

    # 日志配置
    access_log /usr/local/nginx/logs/backup-access.log;
    error_log /usr/local/nginx/logs/backup-error.log warn;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时配置
        proxy_connect_timeout 600s;
        proxy_send_timeout 600s;
        proxy_read_timeout 600s;
    }
}
```

### 3.7.2 HTTPS 示例

> HTTPS 示例（含 80→443 跳转，请替换证书路径）：

```nginx
# HTTP 80端口配置，自动重定向到HTTPS
server {
    listen 80;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：backup.cn
    return 301 https://$host$request_uri;
}

# backup-monitor 站点 HTTPS 配置
server {
    # listen 443 ssl http2;  # Nginx 1.25 以下版本写法
    listen 443 ssl;
    http2 on;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：backup.cn

    # 证书路径（替换为实际证书文件）
    ssl_certificate     /usr/local/nginx/ssl/your-domain.com.pem;  # 例如：/usr/local/nginx/ssl/backup.cn.pem
    ssl_certificate_key /usr/local/nginx/ssl/your-domain.com.key;  # 例如：/usr/local/nginx/ssl/backup.cn.key

    # SSL安全优化
    ssl_protocols              TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers  on;
    ssl_ciphers                ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_session_timeout        10m;
    ssl_session_cache          shared:SSL:10m;

    # 限制上传文件大小（可选）
    client_max_body_size 500m;

    # 日志配置
    access_log /usr/local/nginx/logs/backup-access.log;
    error_log /usr/local/nginx/logs/backup-error.log warn;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时配置
        proxy_connect_timeout 600s;
        proxy_send_timeout 600s;
        proxy_read_timeout 600s;
    }
}
```

# 四、生产环境部署

## 4.1 克隆项目

```bash
git clone https://github.com/zyx3721/backup-monitor.git /data/backup-monitor
cd /data/backup-monitor
```

## 4.2 后端构建与配置

1. 进入后端目录下载相关依赖：

```bash
cd backend
go mod download
```

2. 配置环境变量：

```bash
# 步骤1：复制模板文件
cp .env.example .env

# 步骤2：编辑 .env，按实际环境修改监听地址、密钥等信息
vim .env
# 后端监听地址
SERVER_ADDR=127.0.0.1:8080

# SQLite 数据库文件路径
DB_PATH=./data/backup_monitor.db

# 鉴权与接口行为
# 留空则启动时自动生成随机密钥并持久化到数据库（删除数据库后所有会话自动失效）
JWT_SECRET=
SESSION_TTL_HOURS=12
CORS_ORIGINS=*
```

**配置参数说明详情见 [2.3](#23-后端配置与启动)。**

3. 构建后端可执行文件：

```bash
go build -o backup-monitor cmd/server/main.go
```

4. 运行后端服务： 

```bash
# 方式1：前台运行（终端关闭则服务停止）
./backup-monitor

# 方式2：后台运行（日志输出到 app.log）
nohup ./backup-monitor > app.log 2>&1 &
```

`./backup-monitor -v` 可查看版本信息（版本、commit、构建时间），`-h` 查看全部命令行参数，各参数优先级高于同名环境变量，详见 [2.3](#23-后端配置与启动) 的命令行参数表。

```ini
# 方法3：加入 systemd 管理启动运行
# 服务配置参考如下，请自行修改相应目录路径
cat > /etc/systemd/system/backup-monitor.service <<EOF
[Unit]
Description=Backup Monitor Golang Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/data/backup-monitor/backend
ExecStart=/data/backup-monitor/backend/backup-monitor
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
StandardOutput=journal
StandardError=journal
SyslogIdentifier=backup-monitor

[Install]
WantedBy=multi-user.target
EOF

# 重载服务配置并启动
systemctl daemon-reload
systemctl start backup-monitor

# 设置开机自启
systemctl enable --now backup-monitor
```

## 4.3 Agent 构建与配置

**注：这一步只在备份服务器上，并只需配置启动 Agent。**

1. 在备份服务器上进入 Agent 目录，下载相关依赖：

```bash
cd agent
go mod download
```

2. 创建 `backup-agent.json` 文件配置信息：

```bash
vim backup-agent.json
# Agent 配置
{
  "serverUrl": "",
  "token": "",
  "hostname": "",
  "intervalSec": 0,
  "role": "database",
  "logFile": ""
}
```

**配置参数说明详情见 [2.4](#24-Agent 配置与启动)。**

3. 构建后端可执行文件：

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux ./cmd/backup-agent

# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o dist/backup-agent-linux-arm64 ./cmd/backup-agent

# Windows amd64（先设置环境变量再执行）
$env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="amd64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-amd64.exe ./cmd/backup-agent

# Windows arm64（先设置环境变量再执行）
$env:CGO_ENABLED="0"; $env:GOOS="windows"; $env:GOARCH="arm64"
go build -trimpath -ldflags "-s -w" -o dist/backup-agent-windows-arm64.exe ./cmd/backup-agent
```

> **参数解释**：
>
> - `CGO_ENABLED`：是否启用 CGO，`0` 表示关闭。关掉后不链接系统 libc，产物是纯静态二进制，放到任何发行版（含 Alpine、精简镜像）都能直接跑，不依赖目标机的 glibc 版本
> - `GOOS`：目标操作系统，`linux` 或 `windows`。交叉编译时在这里指定，与当前构建机无关
> - `GOARCH`：目标 CPU 架构，`amd64` 对应 x86_64，`arm64` 对应 ARM 64 位（鲲鹏、飞腾、Apple 芯片等）
> - `-trimpath`：编译时抹掉源码绝对路径，产物里只留模块相对路径。既避免泄露构建机目录，也让同样源码多次构建的结果可复现
> - `-ldflags`：传给链接器的参数，这里用于裁剪体积
>   - `-s`：去掉符号表与调试符号表
>   - `-w`：去掉 DWARF 调试信息
>   - 两项合用通常能再缩小 20%~30% 体积，代价是产物无法用 dlv、gdb 直接调试；需要线上排查时请另编一份不带裁剪的版本
>
> **说明**：**必须使用包目录 `./cmd/backup-agent`**：该目录下的 `version_marker.go` 会写入 `backup-agent-version:X.Y.Z` 标记，管理端上传二进制时据此自动识别版本。若只编译 `cmd/backup-agent/main.go`，程序仍能运行，但产物不含该标记，上传时只能手工填写版本号。
>
> 发布新版本时同步修改两处版本号：`internal/config/types.go` 的 `Version`（运行期上报）与 `cmd/backup-agent/version_marker.go` 的 `embeddedVersion`（服务端识别）。

4. 运行 Agent 服务：

**Linux**：

```bash
./backup-agent -install
systemctl status backup-agent
```

`-install` 会自动写入 `/etc/systemd/system/backup-agent.service`（`ExecStart` 为安装时二进制的绝对路径并带 `-service`、`StandardOutput` 与 `StandardError` 均为 `null`、`SyslogIdentifier=backup-agent`），并把日志留给 Agent 自己写入二进制同目录的 `agent.log`。

二进制放在哪个目录都可以（当前目录、`/opt/backup-agent/`、`/usr/local/bin/` 均可），只要满足：

- **路径稳定**：不要放在 `/tmp`、解压即弃的目录或会被清理的位置
- **就位后再安装**：安装后再移动或改名二进制，需 `-uninstall` 重新安装
- **配置放在默认位置**：单元只带 `-service`，不带 `-config`，配置文件必须落在二进制同目录或 `/etc/backup-agent/`

若希望手工接管单元文件，等价写法如下：

```ini
# 配置 systemd
cat > /etc/systemd/system/backup-agent.service <<EOF
[Unit]
Description=Backup Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
# 改成二进制的实际绝对路径
ExecStart=/data/backup-monitor/agent/backup-agent -service
Restart=always
RestartSec=10
StandardOutput=null
StandardError=null
SyslogIdentifier=backup-agent

[Install]
WantedBy=multi-user.target
EOF
# 重载服务配置并启动
systemctl daemon-reload
systemctl start backup-agent
# 设置开机自启
systemctl enable --now backup-agent
```

**Windows（管理员 PowerShell）**：

```powershell
cd C:\ProgramData\backup-agent
.\backup-agent-windows-amd64.exe -install
Get-Service backup-agent
```

服务名固定为 `backup-agent`，显示名 `Backup Monitor Agent`，启动类型自动，启动参数 `-service`；重复执行 `-install` 不会重复创建。卸载用 `-uninstall`，配置与日志文件保留。

**前台调试（两种平台通用，按 `Ctrl+C` 退出）**：

```bash
./backup-agent -once        # 采集并上报一次后退出
./backup-agent              # 常驻：周期上报 + WebSocket 心跳
```

5. 升级与回滚：

升级可由管理端「Agent 更新 → 推送更新」下发，或由采集配置的自动更新检查触发；服务端按「平台 + 架构 + 通道」托管 5 个槽位（`linux/amd64`、`linux/arm64`、`windows/amd64`、`windows/arm64` 为标准包，`windows/amd64` 另有 legacy）。推送与自动更新都会逐台按主机架构与 Windows 构建号选包：Build ≥ 10240 使用标准包（缺失即跳过，不回退 legacy），低于 10240 使用 legacy；无匹配包的主机跳过并在结果中说明，架构或版本未上报的主机按 amd64 标准包下发并标注「未校验」。自更新前会把旧二进制备份为 `<二进制>.bak`，新版本起不来时可把 `.bak` 改回原文件名并重启服务。服务模式下按当前二进制对应的 systemd 单元（`ExecStart` 匹配，可用 `BM_SERVICE_NAME` 覆盖）执行 `systemctl restart`，不会重启同机其它服务。完整步骤见 [agent/README.md](../agent/README.md) 的「升级与回滚」。

## 4.4 前端构建与配置

1. 进入前端目录下载相关依赖：

```bash
cd frontend
npm install
```

2. 构建前端项目：

```bash
npm run build
```

构建产物在 `.output` 目录。当前前端使用 TanStack React Start + Nitro，构建后会生成可直接运行的 Node 服务端入口和静态资源目录：

- `.output/server/index.mjs`：生产环境 Node SSR 入口；
- `.output/public/`：浏览器静态资源，包含 JS、CSS、favicon 等文件；
- 生产环境浏览器侧的 `/api/` 请求统一通过 Nginx 将 `/api/` 反向代理到后端，前端无需单独配置 API 地址；但 SSR 进程启动时会主动访问后端拉取品牌配置，把站点名与图标直出进首帧 HTML。

因此生产部署时需要先启动 `.output/server/index.mjs`，再由 Nginx 将页面请求反向代理到该前端服务；不要只把 `.output/public` 配置为 Nginx 静态根目录，否则服务端渲染页面无法正常返回。

3. 启动前端 SSR 服务：

```bash
# 方式1：前台运行（终端关闭则服务停止）
HOST=127.0.0.1 PORT=5173 npm run start

# 方式2：后台运行（日志输出到 bm-frontend.log）
nohup env HOST=127.0.0.1 PORT=5173 npm run start > bm-frontend.log 2>&1 &
```

SSR 拉取品牌配置的后端地址按以下顺序确定：运行时环境变量 `SSR_API_ORIGIN` → 入口文件 `index.mjs` 所在目录向上任意一层的 `.env` 文件（首个存在的生效，可与后端共用部署根目录的同一份 `.env`）→ 默认 `http://127.0.0.1:8080`（开发模式下会先回退到 `VITE_API_BASE_URL`，与 dev 代理同源）。请确保该地址对 SSR 进程可达，否则首屏会先显示默认品牌、加载后再切换为配置值，且 SSR 进程日志会输出 `[bm-ssr] fetch brand failed` 警告。

## 4.5 配置 Nginx 反向代理

在服务器上准备前端目录（例如 `/data/backup-monitor/frontend/.output`），**将本地 `.output` 目录中的所有文件和子目录整体上传到该目录**，保持 `public/` 与 `server/` 结构不变，例如：

```bash
/data/backup-monitor/frontend/.output/
├── public/
│   ├── assets/             # 前端浏览器端 JS/CSS 静态资源
│   └── favicon.svg         # 站点图标
└── server/
    └── index.mjs           # Nitro 生产 SSR 入口
```

上传完成后，在 `.output` 所属的前端项目目录执行 `HOST=127.0.0.1 PORT=5173 npm run start` 启动前端服务。Nginx 的 `/` 请求应反向代理到该服务，例如下方示例中的 `127.0.0.1:5173`；`/api/` 和 `/swagger/` 仍反向代理到 Go 后端 `127.0.0.1:8080`。

`/assets/` 下带扩展名的构建产物（JS/CSS/字体/图片等）可由 Nginx 直接读取 `.output/public` 返回，避免静态资源经过前端 SSR 服务，并可为带 hash 的构建资源启用长期缓存；`/favicon.svg` 同理。注意资产模块的页面路由同样位于 `/assets/` 前缀下（如 `/assets/software`），因此静态资源 location 必须按扩展名正则匹配，不能用 `^~ /assets/` 前缀匹配加 `=404`，否则这些页面的整页请求（新标签打开、刷新、直链）会被静态块拦截返回 404；也不要改用 `try_files $uri @ssr` 回退，该块的一年期强缓存响应头会同样作用于回退返回的 SSR 页面。`/crl/`、`/ocsp` 与 `/ocsp/` 必须直接反向代理到 Go 后端，保留原始路径、请求方法和 `Content-Type`，以支持 CRL 分发、RFC 6960 二进制请求及 JSON 状态查询。示例中的 `root /data/backup-monitor/admin/.output/public;` 请按实际上传目录替换。

### 4.5.1 HTTP 示例

> 配置 Nginx （按需替换域名/路径/证书），`HTTP 示例` ：

```nginx
server {
    listen 80;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：backup.cn
    
    # 限制上传文件大小（可选）
    client_max_body_size 500m;
    
    # 日志配置
    access_log /usr/local/nginx/logs/backup-access.log;
    error_log /usr/local/nginx/logs/backup-error.log warn;

    # 前端静态资源：直接读取 .output/public，避免 JS/CSS 经过 SSR 服务
    location ^~ /assets/ {
        root /data/backup-monitor/frontend/.output/public;
        try_files $uri =404;
        access_log off;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # 站点图标
    location = /favicon.svg {
        root /data/backup-monitor/frontend/.output/public;
        try_files $uri =404;
        access_log off;
        expires 7d;
        add_header Cache-Control "public";
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 后端 API 文档
    location /swagger/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 前端 Nitro SSR 服务
    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 健康检查
    location = /health {
        proxy_pass http://127.0.0.1:8080/api/health;
    }
}
```

### 4.5.2 HTTPS 示例

> HTTPS 示例（含 80→443 跳转，请替换证书路径）：

```nginx
# HTTP 80端口配置，自动重定向到HTTPS
server {
    listen 80;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：backup.cn
    return 301 https://$host$request_uri;
}

# backup-monitor 站点 HTTPS 配置
server {
    # listen 443 ssl http2;  # Nginx 1.25 以下版本写法
    listen 443 ssl;
    http2 on;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：backup.cn

    # 证书路径（替换为实际证书文件）
    ssl_certificate     /usr/local/nginx/ssl/your-domain.com.pem;  # 例如：/usr/local/nginx/ssl/backup.cn.pem
    ssl_certificate_key /usr/local/nginx/ssl/your-domain.com.key;  # 例如：/usr/local/nginx/ssl/backup.cn.key
    
    # SSL安全优化
    ssl_protocols              TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers  on;
    ssl_ciphers                ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_session_timeout        10m;
    ssl_session_cache          shared:SSL:10m;
    
    # 限制上传文件大小（可选）
    client_max_body_size 500m;
    
    # 日志配置
    access_log /usr/local/nginx/logs/backup-access.log;
    error_log /usr/local/nginx/logs/backup-error.log warn;

    # 前端静态资源：直接读取 .output/public，避免 JS/CSS 经过 SSR 服务
    location ^~ /assets/ {
        root /data/backup-monitor/frontend/.output/public;
        try_files $uri =404;
        access_log off;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # 站点图标
    location = /favicon.svg {
        root /data/backup-monitor/frontend/.output/public;
        try_files $uri =404;
        access_log off;
        expires 7d;
        add_header Cache-Control "public";
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 后端 API 文档
    location /swagger/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 前端 Nitro SSR 服务
    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 健康检查
    location = /health {
        proxy_pass http://127.0.0.1:8080/api/health;
    }
}
```

## 4.6 访问系统

服务启动后，访问以下地址：

- **首页**：`http://your-domain.com`
  - **默认用户名**：`admin`
  - **默认密码**：`admin123`
- **API 文档**：`http://your-domain.com/swagger/index.html`
- **健康检查**：`http://your-domain.com/health`

# 五、API 文档

后端已集成 Swagger/OpenAPI 文档，启动后可通过以下地址查看在线接口文档：

- **Swagger UI**：`http://localhost:8080/swagger/index.html`
- **OpenAPI JSON**：`http://localhost:8080/swagger/doc.json`
- **健康检查**：`GET /health`、`GET /api/health`

除 `POST /api/auth/login`、`GET /api/auth/providers`、`GET /api/auth/wecom/authorize`、`POST /api/auth/wecom/callback`、`POST /api/auth/wecom/sso/callback`、`GET /api/auth/password-reset/captcha`、`POST /api/auth/password-reset/verify`、`POST /api/auth/password-reset/send`、`POST /api/auth/password-reset/confirm`、`GET /api/public/base`、`GET /health` 和 `GET /api/health` 外，其他接口均需要在请求头中携带 `Authorization: Bearer <token>`。

登录请求示例：

```json
{
  "username": "admin",
  "password": "admin123",
  "mode": "local"
}
```

## 5.1 权限模型

接口按角色权限逐个校验，无权限返回 403；默认管理员 `admin` 用户拥有全部权限。权限 Key 形如 `模块.资源.操作`，共 23 项，权限目录的「分类」与侧边栏集合一一对应：

- **概览**：`backup.dashboard.read`（查看概览）
- **备份主机**：`backup.hosts.read`（查看主机）、`backup.hosts.manage`（修改主机角色、删除与导出主机，隐含 `backup.hosts.read`）
- **备份任务**：`backup.tasks.read`（查看任务）、`backup.tasks.manage`（修改执行计划、删除与导出任务，隐含 `backup.tasks.read`）
- **执行记录**：`backup.executions.read`（查看执行记录）、`backup.executions.manage`（删除与导出执行记录，隐含 `backup.executions.read`）
- **告警中心**：`backup.alerts.read`（查看告警事件）、`backup.alerts.manage`（恢复、删除与导出告警事件，隐含 `backup.alerts.read`）
- **通知管理**：`backup.notifications.read`（查看通知渠道与订阅配置）、`backup.notifications.manage`（维护并测试通知渠道，隐含 `.read`）
- **Agent 管理**：`backup.agents.read`（查看采集配置、Agent 令牌与二进制）、`backup.agents.manage`（维护采集配置、令牌并推送更新，隐含 `.read`）
- **审计日志**：`audit.read`（查看审计日志）、`audit.manage`（导出与清空审计日志，隐含 `audit.read`）
- **系统配置**：`settings.{base|users|auth|notifications}.read` / `.manage`，分别对应基础配置（品牌标识、安全时效与备份参数）、用户配置（用户 / 群组 / 角色）、认证配置（AD/LDAP 与企业微信）、通知配置（邮件）
- **隐含规则**：勾选任一 `manage` 时，服务端保存角色与登录下发权限时会自动补全其隐含的 `read` 权限
- **只读边界**：仅持有 `read` 的角色可查看列表与详情，写操作按钮不渲染；「Agent 管理 → 采集配置 / Agent 令牌」在无管理权限时整列隐藏「操作」列
- **无权限访问**：未持有某集合的 `read` 时侧边栏不显示入口，直接访问该集合页面或详情链接只展示「暂无<集合名称>查看权限，如需查看请联系管理员分配」占位，且不发起该页数据请求；顶部告警铃与全局搜索只按已授权集合展示；缺少概览权限但持有其他集合权限时，登录后自动进入第一个可访问集合，仅当账号没有任何查看权限时才停在概览并展示该提示
- **身份说明**：默认管理员 `admin` 用户拥有全部权限且不可禁用、不可删除；内置角色 `admin` 持有全部权限；内置角色 `operator` 可管理主机、任务、执行记录、告警、通知渠道与 Agent 配置，但不能修改系统配置与清空审计；`viewer` 仅具备备份监控与系统配置的查看权限
- **权限目录**：`GET /api/settings/permissions` 返回当前权限目录全量清单

## 5.2 接口清单

以下接口清单与 Swagger 文档（`/swagger/index.html`）一致，按模块分组列出；除标注「无需认证」的接口外，均需在请求头携带 `Authorization: Bearer <token>`。Agent 通道（5.2.6）使用 `X-Agent-Token` 而非 Bearer 令牌。

### 5.2.1 健康检查

- `GET /health` - 健康检查：返回服务健康状态，无需认证
- `GET /api/health` - 健康检查：同 `/health`，供容器 HEALTHCHECK 与 Nginx 健康检查使用，无需认证

### 5.2.2 认证与会话

- `POST /api/auth/login` - 用户登录：支持本地密码（`mode=local`，默认）与 AD/LDAP（`mode=ldap`）两种方式，成功返回令牌与用户信息（含权限清单与企微绑定状态）；同一账号连续密码失败达到「安全时效」配置的锁定次数后返回 429 并提示剩余等待分钟数，锁定时长内即使密码正确也会被拒绝，登录成功或锁定窗口过期后重新计数，内置管理员 admin 不受限，无需认证
- `GET /api/auth/providers` - 获取公开认证方式：获取登录页可用的认证方式与找回密码开关，无需认证
- `GET /api/auth/wecom/authorize` - 获取企业微信扫码登录地址：生成企业微信 Web 扫码登录页地址（含防伪 state）与内嵌二维码渲染参数（`iframe_url`、`callback_path`）；直连模式内嵌扫码回跳 `/wecom-qr-callback`，统一认证中心模式回跳 `/login`。登录页默认内嵌渲染二维码，配置异常时自动回退整页跳转；需已启用企业微信认证，无需认证
- `POST /api/auth/wecom/callback` - 企业微信扫码登录回调：校验 state 后用授权码换取企业微信成员身份，按绑定关系登录并返回令牌与用户信息；未绑定时返回 401，无需认证
- `POST /api/auth/wecom/sso/callback` - 统一认证中心扫码登录回调：统一认证（SSO）模式下校验认证中心回跳的一次性 ticket 后按绑定关系登录并返回令牌与用户信息；需已启用统一认证模式，无需认证
- `GET /api/auth/password-reset/captcha` - 获取找回密码图形验证码：获取找回密码图形验证码，无需认证
- `POST /api/auth/password-reset/verify` - 校验找回密码身份：校验用户名与图形验证码，换取找回密码流程令牌，无需认证
- `POST /api/auth/password-reset/send` - 发送找回密码验证码：向校验通过的邮箱发送找回密码验证码，受发送冷却与限流窗口约束，无需认证
- `POST /api/auth/password-reset/confirm` - 确认找回密码：凭邮箱验证码完成找回密码，重置账号密码并写入审计，若该账号存在登录失败锁定记录则一并解除，并退出该账号全部登录会话，无需认证
- `GET /api/auth/me` - 获取当前用户：获取当前登录用户的资料、直接角色、有效角色与权限清单
- `POST /api/auth/logout` - 登出当前会话：退出当前会话，并写入用户注销审计
- `POST /api/auth/change-password` - 修改当前用户密码：修改当前登录用户的密码，成功后需重新登录
- `GET /api/auth/wecom/bind-url` - 获取企业微信绑定扫码地址：为当前登录用户生成企业微信绑定扫码地址，state 绑定当前用户
- `POST /api/auth/wecom/bind` - 绑定企业微信账号：校验绑定 state 后用授权码换取企业微信成员 userid，与当前用户建立绑定；该企微已绑定其他用户时返回 409
- `POST /api/auth/wecom/sso/bind` - 统一认证中心扫码绑定：统一认证（SSO）模式下校验认证中心一次性 ticket 后与当前登录用户建立绑定；该企微已绑定其他用户时返回 409
- `DELETE /api/auth/wecom/bind` - 解绑企业微信账号：解除当前登录用户的企业微信绑定
- `GET /api/public/base` - 获取公开品牌标识：获取登录页与启动屏使用的品牌标识（站点名、登录页名称、控制台名称、图标），前端 SSR 阶段实时拉取直出首帧，无需认证

### 5.2.3 系统设置

**基础配置**（权限 `settings.base`）

- `GET /api/settings/base` - 获取基础配置：读取品牌标识（站点名 / 登录页 / 控制台名称 / 图标）与安全时效（登录失败锁定与找回密码时效）
- `PUT /api/settings/base` - 更新基础配置：更新品牌标识与安全时效并写入审计
- 备份参数（判定阈值、保留策略与 Agent 通道参数）位于 `GET /api/backup/settings` / `PUT /api/backup/settings`，同样以 `settings.base.read` / `settings.base.manage` 校验，见 [12.4 协议与接口](#124-协议与接口)

**权限目录与角色**（权限 `settings.users`）

- `GET /api/settings/permissions` - 获取权限目录：返回 23 项权限的全量清单（Key、名称、描述、分类、隐含关系）
- `GET /api/settings/roles` - 获取角色列表：返回内置与自定义角色及其权限集合
- `POST /api/settings/roles` - 创建角色：创建自定义角色并勾选权限，勾选 `manage` 时自动补全对应 `read`
- `PUT /api/settings/roles/{id}` - 更新角色：更新角色名称、描述与权限集合；内置角色的权限集合随权限目录自动刷新
- `POST /api/settings/roles/{id}/disabled` - 启用/禁用角色：切换角色启用状态
- `DELETE /api/settings/roles/{id}` - 删除角色：删除自定义角色；内置角色不可删除

**用户群组**（权限 `settings.users`）

- `GET /api/settings/user-groups` - 获取用户群组列表：返回群组及其成员与授予的角色
- `POST /api/settings/user-groups` - 创建用户群组：创建群组，名称唯一
- `PUT /api/settings/user-groups/{id}` - 更新用户群组：更新群组名称、描述、成员与授予的角色
- `DELETE /api/settings/user-groups/{id}` - 删除用户群组：删除群组并解除成员与角色关联

**用户配置**（权限 `settings.users`）

- `GET /api/settings/users` - 获取用户列表：返回用户资料、直接角色、有效角色与权限清单
- `POST /api/settings/users` - 创建用户：创建本地用户并设置初始密码
- `PUT /api/settings/users/{id}` - 更新用户：更新用户资料、密码与直接角色
- `POST /api/settings/users/{id}/disabled` - 启用/禁用用户：切换用户启用状态；默认管理员 admin 不可禁用
- `DELETE /api/settings/users/{id}` - 删除用户：删除用户及其角色关联；默认管理员 admin 不可删除

**认证配置**（权限 `settings.auth`）

- `GET /api/settings/auth-provider` - 获取 AD/LDAP 认证配置：读取 LDAP 连接参数与启用状态，绑定密码脱敏返回
- `PUT /api/settings/auth-provider` - 更新 AD/LDAP 认证配置：保存连接参数（绑定密码 AES-GCM 加密落库）并写入审计
- `POST /api/settings/auth-provider/test` - 测试 LDAP 连通性：按当前配置发起连通性测试
- `GET /api/settings/auth/wecom` - 获取企业微信认证配置：读取直连 / 统一认证中心两种模式的参数与启用状态，Secret 脱敏返回
- `PUT /api/settings/auth/wecom` - 更新企业微信认证配置：保存企业 ID、AgentID、Secret（加密落库）与认证方式并写入审计

**通知配置**（权限 `settings.notifications`）

- `GET /api/settings/email` - 获取邮件配置：读取找回密码邮件的 SMTP 参数与启用状态，密码脱敏返回
- `PUT /api/settings/email` - 更新邮件配置：保存 SMTP 参数（密码加密落库）并写入审计
- `POST /api/settings/email/test` - 测试邮件发送：按当前配置发送测试邮件

### 5.2.4 审计日志

- `GET /api/history` - 查询审计日志：默认返回全部保留记录，可选 `limit` 限制条数；按模块、操作、目标、操作人与时间在前端筛选与分页（权限 `audit.read`）
- `POST /api/history/export` - 记录数据导出：备份主机、备份任务、执行记录、告警中心与审计日志导出成功后回写一条审计，模块取对应业务域、目标为导出条数（`N 条`）、详情为「所有<对象>数据已导出」
- `POST /api/history/clear` - 清空审计日志：清空全部审计记录并写入清空动作本身的审计（权限 `audit.manage`）

### 5.2.5 备份监控

**概览**（权限 `backup.dashboard`）

- `GET /api/backup/dashboard` - 概览统计：返回主机、任务、今日执行与告警的汇总、状态分布、备份方式分布与每日趋势

**备份主机**（权限 `backup.hosts`）

- `GET /api/backup/hosts` - 获取主机列表：返回主机及其状态、最近上报时间与任务数；`ids` 逗号分隔时只返回这些主机（权限 `.read`）
- `GET /api/backup/hosts/refresh-plan` - 获取主机刷新计划：按主机实际匹配到的上报间隔分组，供前端按组局部刷新（权限 `.read`）
- `GET /api/backup/hosts/{id}` - 获取主机详情：返回指定主机及其名下任务列表（权限 `.read`）
- `PUT /api/backup/hosts/{id}/role` - 修改主机角色：更新角色并写入审计；角色保存后不再被 Agent 上报覆盖（权限 `.manage`）
- `DELETE /api/backup/hosts/{id}` - 删除主机：级联删除其任务、执行记录与告警，审计详情记录级联条数（权限 `.manage`）

**备份任务**（权限 `backup.tasks`）

- `GET /api/backup/tasks` - 获取任务列表：返回全部任务及其实计划、状态与所属主机（权限 `.read`）
- `GET /api/backup/tasks/{id}` - 获取任务详情：返回指定任务（权限 `.read`）
- `GET /api/backup/tasks/{id}/executions` - 获取任务执行历史：返回该任务的执行记录列表（权限 `.read`）
- `PUT /api/backup/tasks/{id}/schedule` - 修改执行计划：更新 cron 并下发 Agent，仅主机心跳在线时成功；计划未变化时直接返回（权限 `.manage`）
- `DELETE /api/backup/tasks/{id}` - 删除任务：级联删除执行记录与告警，审计详情记录级联条数（权限 `.manage`）

**执行记录**（权限 `backup.executions`）

- `GET /api/backup/executions` - 获取执行记录列表：返回全部执行记录及其任务、主机与结果（权限 `.read`）
- `GET /api/backup/executions/{id}` - 获取执行记录详情（权限 `.read`）
- `DELETE /api/backup/executions/{id}` - 删除执行记录：审计详情含主机、任务与执行时间（权限 `.manage`）

**告警中心**（权限 `backup.alerts`）

- `GET /api/backup/alerts` - 获取告警事件列表：返回全部告警及其状态、级别与关联任务（权限 `.read`）
- `POST /api/backup/alerts/{id}/resolve` - 恢复告警：手动恢复指定告警并写入审计（权限 `.manage`）
- `DELETE /api/backup/alerts/{id}` - 删除告警：删除指定告警并写入审计（权限 `.manage`）

**通知管理**（权限 `backup.notifications`）

- `GET /api/backup/notify/channels` - 获取通知渠道列表（权限 `.read`）
- `POST /api/backup/notify/channels` - 创建通知渠道（权限 `.manage`）
- `PUT /api/backup/notify/channels/{id}` - 更新通知渠道：更新或启停用，未改动时不写审计（权限 `.manage`）
- `POST /api/backup/notify/channels/{id}/test` - 测试通知渠道：按渠道配置发送测试通知，成功与失败均写入审计（权限 `.manage`）
- `DELETE /api/backup/notify/channels/{id}` - 删除通知渠道（权限 `.manage`）
- `GET /api/backup/notifications?limit=&channel=` - 获取通知发送记录：默认 100 条，最多 500 条；`channel` 传通知渠道 ID 时只返回该渠道的记录，省略时返回全部（权限 `.read`）

**通知渠道配置说明**

- 企业微信与钉钉支持「文本（默认）/ Markdown」；飞书支持「文本（默认）/ 富文本（post）/ 卡片（interactive）」。
- 切换消息类型会用该类型的默认模板重置请求体，但保留企业微信 `touser`、钉钉 `userid_list`、飞书 `receive_id` 的原值。
- 飞书 `receive_id_type` 是发送时的 URL 查询参数，不放在请求体里；`receive_id` 填对应类型的 ID 值。
- 飞书 `receive_id` 支持多个接收人，用英文或中文逗号、分号、空格或换行分隔；后端逐个发送，任一失败则该次发送记录为失败。
- 飞书卡片 `header.template` 默认使用 `{{颜色}}`：测试与恢复为 `green`，告警为 `red`。
- 企业微信、钉钉的 Markdown 请求体与测试文案按各自规则生成；钉钉 Markdown 会对换行做加倍处理，避免 Markdown 渲染把空行折叠。
- 发送结果统一在「通知管理 → 发送记录」查看，失败记录会保留错误原因。

**Agent 管理**（权限 `backup.agents`）

- `GET /api/backup/profiles` - 获取采集配置列表（权限 `.read`）
- `POST /api/backup/profiles` - 创建采集配置（权限 `.manage`）
- `PUT /api/backup/profiles/{id}` - 更新采集配置：未改动时不写审计（权限 `.manage`）
- `DELETE /api/backup/profiles/{id}` - 删除采集配置（权限 `.manage`）
- `GET /api/backup/tokens` - 获取 Agent 令牌列表（权限 `.read`）
- `POST /api/backup/tokens` - 创建 Agent 令牌：可选传入 `token` 自定义令牌值（16-128 位可见 ASCII，留空自动生成 48 位随机值），仅在创建响应中返回明文令牌（权限 `.manage`）
- `PUT /api/backup/tokens/{id}` - 更新 Agent 令牌：更新名称、绑定主机或启用状态（权限 `.manage`）
- `DELETE /api/backup/tokens/{id}` - 删除 Agent 令牌（权限 `.manage`）
- `GET /api/backup/agent-binaries` - 获取已托管二进制：返回各槽位（平台 + 架构 + 通道）的版本、Go 版本、大小与 SHA-256（权限 `.read`）
- `POST /api/backup/agent-binary?platform=&version=&variant=` - 上传二进制：请求体为原始字节，服务端校验 ELF/PE、解析架构与 Go 版本，`variant` 缺省时 Windows amd64 且 Go < 1.21 归入 legacy，单次不超过 128MB（权限 `.manage`）
- `PUT /api/backup/agent-update/push` - 推送自更新：按平台与全部或 IP 条件筛选主机，逐台按架构与 Windows 版本选包，无匹配包的主机跳过并返回原因（权限 `.manage`）

**备份参数**（权限 `settings.base`）

- `GET /api/backup/settings` - 获取备份参数：判定阈值、保留策略与 Agent 通道参数（权限 `.read`）
- `PUT /api/backup/settings` - 更新备份参数：按 `section` 更新判定与保留、Agent 通道或安全与告警参数并写入审计；WS 重连退避需满足上限不小于下限（权限 `.manage`）

### 5.2.6 Agent 通道

Agent 通道使用 `X-Agent-Token` 认证头（未启用令牌校验时可省略），不使用 Bearer 令牌：

- `POST /api/ingest/report` - Agent 上报数据：上报主机、任务与执行记录，按 `task + session` 幂等更新
- `GET /api/agent/config?hostname=&platform=` - 拉取 Agent 配置：返回全局参数、采集配置与二进制更新信息
- `GET /api/agent/ws?hostname=&nodeId=&platform=` - Agent 心跳长连接：WebSocket，承载心跳、配置变更与 `set_cron`、`update` 回执
- `GET /api/agent/download?platform=&arch=&variant=` - 下载托管二进制：供 Agent 自更新使用；arch 与 variant 可选
- `GET /api/agent/binary/latest?platform=&arch=&osBuild=` - 查询最新版本：返回版本号、CPU 架构、大小与 SHA-256；arch 与 osBuild 可选
- `GET /download/backup-agent?platform=&arch=&variant=` - 公开下载二进制：无需任何凭证，供首次部署直接拉取，响应头含版本与 SHA-256；arch 与 variant 可选

上报间隔按「采集配置 `reportIntervalSec`（非 0）→ 全局默认上报间隔 → Agent 本地 `intervalSec` → 600 秒」依次取值，任何来源低于 10 秒都按 10 秒执行；`GET /api/backup/hosts/refresh-plan` 按该结果分组，控制台据此按组局部刷新主机列表，其余页面按最小间隔刷新。

# 六、数据库说明

使用 SQLite 单文件数据库，默认路径 `backend/data/backup_monitor.db`，共 25 张表（16 张平台表、9 张备份监控表）。数据库连接固定 `MaxOpenConns(1)`，启用 WAL 模式。所有时间列统一以 UTC 毫秒 ISO 文本存储（如 `2026-10-22T06:39:34.691Z`），接口返回与前端展示格式保持不变。

## 6.1 用户与权限表

| 表名 | 说明 |
|:----:|:----:|
| `users` | 系统用户（username 唯一、bcrypt 口令散列、email、disabled、wecom_userid 唯一绑定、last_login_at） |
| `roles` | 用户角色（key 唯一、permissions JSON 数组、builtin 内置标记、disabled 启用状态） |
| `user_roles` | 用户 ↔ 角色直接授权关系 |
| `user_groups` | 用户群组（name 唯一、disabled） |
| `user_group_members` | 群组 ↔ 用户成员关系 |
| `user_group_roles` | 群组 ↔ 角色授权关系 |

## 6.2 系统与配置表

| 表名 | 说明 |
|:----:|:----:|
| `system_secrets` | JWT 签名密钥（单行表，留空时首次启动自动生成并持久化） |
| `sessions` | JWT 登录会话（`jti` 主键、`user_id`、`username`、`source`、`created_at`、`expires_at`），注销即吊销对应令牌 |
| `basic_config` | 基础配置（`brand` 品牌标识、`security` 安全时效、`thresholds` 判定与保留、`agent` Agent 通道、`alarm` 安全与告警，name 唯一 + 各行 config JSON） |
| `auth_config` | 认证配置（`ldap` 与 `wecom` 两行种子记录，config JSON，绑定密码 / Secret 加密存储） |
| `pwd_recover_config` | 找回密码配置（`email` 找回密码邮件 SMTP 参数，config JSON） |

## 6.3 认证与找回密码表

| 表名 | 说明 |
|:----:|:----:|
| `login_failure_log` | 登录密码失败记录（登录失败锁定计数，成功登录或锁定窗口过期后重新计数） |
| `reset_captchas` | 找回密码图形验证码（token_hash 主键、answer_hash、expires_at） |
| `reset_requests` | 找回密码流程请求（token_hash 主键、邮箱验证码 code_hash、expires_at、used、last_sent_at） |
| `reset_send_log` | 找回密码验证码发送记录（按邮箱实现发送冷却与限频） |

## 6.4 审计表

| 表名 | 说明 |
|:----:|:----:|
| `audit_logs` | 审计日志（date、authuser 操作人、ip、module 模块、action 操作、target 目标、detail 详情、result 结果） |

# 七、常见问题

## 7.1 忘记管理员密码怎么办？

停掉后端服务后执行：

```bash
# 生成 bcrypt 哈希（需 Go 1.25+，在 backend 目录执行）
cd backend && go run - <<'EOF'
package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	h, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	fmt.Println(string(h))
}
EOF
```

```bash
# 将生成的哈希写入数据库（推荐，不丢数据）
sqlite3 backend/data/backup_monitor.db "UPDATE users SET pass = '<生成的哈希>' WHERE username = 'admin';"
```

重启后用 `admin / admin123` 登录并立即修改密码。注意：`users.pass` 必须是合法的 bcrypt 哈希，直接写入明文会导致无法登录。

## 7.2 主机一直显示离线怎么办？

先看是「异常」还是「离线」：Agent 超过 **3 分钟**没上报判为异常，超过 **6 分钟**判为离线（阈值在「系统配置 → 基础配置 → 判定与保留」调整）。

按顺序排查：

1. **Agent 是否在跑** — Linux `systemctl status backup-agent`，Windows `Get-Service backup-agent`；没跑起来先看服务安装日志
2. **Agent 日志** — 二进制同目录 `agent.log`，里面有连接失败、令牌被拒、配置拉取失败的具体原因
3. **地址与令牌** — `serverUrl` 必须可达且是 http/https 绝对地址；启用令牌校验时 `token` 要与「Agent 管理 → Agent 令牌」里的一致
4. **主机名是否被排除** — 采集配置按 `hostname` 通配匹配，一条都不命中时 Agent 只发心跳、不采集，但主机仍应出现在列表里

## 7.3 备份任务被判为「未执行」，但任务确实跑过？

按判定链路逐段核对：

- **宽限期**：计划时刻后 **30 分钟**仍无执行记录才判未执行，可在「判定与保留」调整；执行记录晚于窗口落库时不会回填判定
- **匹配窗口**：执行记录与计划的匹配窗口为 **60 分钟**，跨窗口的历史记录无法自动关联
- **任务是否被采集到**：`crontab` 需要带约定标签、Windows 计划任务需存在且可读、Veeam Job 需能从 `veeamconfig` 解析；「备份任务 → 采集配置」可查看实际下发的规则
- **采集配置是否命中**：`hostname` 通配不匹配时该主机不会采集任务，只上报心跳
- **手工登记任务**：没有采集源的任务需由 Agent 或接口自行上报执行记录，服务端不会凭空生成

判定规则与状态取值见 [12.2](#122-状态判定)。

## 7.4 告警收不到、或者重复推送？

- **渠道是否订阅该事件**：通知渠道按事件类型（备份失败、未执行、主机异常）与级别过滤，没勾选就不会发
- **发送记录**：「通知管理 → 发送记录」能看到每条通知的发送结果与失败原因
- **邮件配置**：邮件渠道不单独配置 SMTP，统一取「系统配置 → 找密设置 → 邮件媒介」，在那里可以点「发送测试邮件」验证
- **认证报 `unencrypted connection`**：SMTP 主机未启用 TLS/STARTTLS 时，需要在「找密设置 → 邮件媒介」勾选「允许明文认证」，否则找密邮件与告警邮件的发送都会被拒绝
- **重复推送**：告警默认去重，同一任务同一类型在去重窗口内只发一次；需要每次都提醒可在「安全与告警」关闭去重
- **恢复通知**：任务恢复后是否补发恢复通知由渠道配置决定

## 7.5 上传的 Agent 二进制识别不到版本？

构建时要用**包目录**：`go build ./cmd/backup-agent`。该目录下的 `version_marker.go` 会把 `backup-agent-version:X.Y.Z` 标记写进产物，服务端上传时据此自动识别版本与架构。

只编译 `cmd/backup-agent/main.go` 时程序仍能运行，但产物没有该标记，上传时只能手工填写版本号；发布新版本时记得同时改 `internal/config/types.go` 的 `Version` 与 `version_marker.go` 的 `embeddedVersion`。

## 7.6 Agent 自更新失败怎么回滚？

自更新会先把旧二进制改名为 `<二进制>.bak`，再替换并重启：

- 替换失败：自动把 `.bak` 还原回原名
- 重启失败：同样尝试还原，并在日志中记录失败原因
- 新版本能起来但行为异常：停止服务，把 `<二进制>.bak` 改回原文件名（Windows 为 `backup-agent.exe.bak` → `backup-agent.exe`），再启动服务

服务定义里记录的是安装时的绝对路径，回滚后路径不变，无需重新 `-install`。

## 7.7 执行记录、发送记录和审计日志保留多久？怎么清理？

默认均保留 **90 天**，在「系统配置 → 基础配置 → 判定与保留」中调整；填 `0` 表示不自动清理。

清理在巡检周期（默认 30 秒一轮）内按保留天数删除过期数据，不做物理收缩，SQLite 文件不会立即变小；需要回收空间时停服后执行 `VACUUM`。告警记录不参与自动清理，需在「告警中心」手工处理或随主机/任务级联删除。

## 7.8 数据库文件可以直接复制迁移吗？

可以。SQLite 是单文件数据库，**停服后**复制 `backend/data/backup_monitor.db`（连同同目录的 `-wal`、`-shm`）即可，`data/agents/` 下的 Agent 二进制按需一并带走。

运行中复制请用 SQLite 的在线备份方式（如 `sqlite3 backup_monitor.db ".backup copy.db"`），直接拷贝正在写入的文件可能拿到不一致快照。

## 7.9 如何修改 JWT 有效期？

默认 12 小时，通过环境变量 `SESSION_TTL_HOURS`（单位小时，正整数）调整，修改后重启后端生效。登录会话持久化在服务端 `sessions` 表中，注销后对应令牌立即失效；重设 `JWT_SECRET` 可使全部已签发会话立即失效。

## 7.10 如何调整登录失败锁定策略？

「系统配置 → 基础配置 → 安全时效」中可调整锁定阈值（3-10 次）与锁定时长（1-10 分钟），保存后即时生效；内置管理员 admin 不受限。失败记录在登录成功或锁定窗口过期后重新计数。

## 7.11 如何启用 LDAP 登录？

第一步在「系统配置 → 认证配置」填好 LDAP 连接参数并启用；第二步在「系统配置 → 用户配置」中创建与 LDAP `sAMAccountName` 同名的用户。本地用户表中不存在该用户名时，即使 LDAP 密码正确也无法登录（后端返回「用户未开通」）。

## 7.12 如何启用企业微信扫码登录？

在企业微信管理后台「应用管理」创建自建应用，记录 AgentID 与 Secret；在应用的「网页授权及 JS-SDK」中将本系统访问域名配置为可信回调域名，并在「企业可信 IP」中加入本服务出口 IP。再到「系统配置 → 认证配置 → 企业微信」填写企业 ID（corpid）、AgentID、Secret 并启用（回调地址前缀可留空，按当前访问地址自动推断）。用户先用账号密码登录，在右上角菜单「绑定企微」扫码完成账号关联，之后即可在登录页选择企业微信扫码登录。登录页选择企业微信后默认在页面内内嵌渲染扫码二维码，配置异常时自动回退为整页跳转。

若企业内部已部署统一认证中心（wecom-auth-center），可在企业微信认证配置中将「认证方式」切换为「统一认证中心」，填写认证中心地址、应用标识与应用密钥；认证中心侧为本系统配置 `domain`（本系统外部访问地址）与 `callback_path`（固定填 `/login`），多个内部系统可共用同一套企微应用配置，绑定与登录交互保持一致。

认证中心扫码确认后会携带一次性 ticket 回跳 `/login`，由登录页完成登录或绑定。`/wecom-qr-callback` 是**直连模式**下登录页内嵌扫码的内部中转路由，只在系统内部使用，不要填到认证中心——填错会导致扫码后无法回跳。

## 7.13 如何修改品牌标识？

「系统配置 → 基础配置」中维护站点名、登录页名称、控制台名称、控制台品牌副标题与图标，保存后刷新即生效；品牌信息由前端 SSR 阶段实时拉取 `/api/public/base` 直出首帧，不落本地缓存。

副标题建议写成定位短语（如 `Backup Control Plane`），不要与站点名重复；留空则回落到默认值。

## 7.14 容器或主机时间不对，页面时间差 8 小时？

数据库统一以 **UTC 毫秒 ISO 文本**存储（如 `2026-10-22T06:39:34.691Z`），前端按浏览器本地时区展示，所以页面时间通常是对的；差 8 小时一般出在容器：

- 容器内 `TZ` 未设置时进程按 UTC 运行，日志时间会与本地时间差 8 小时
- 在 `deploy/.env` 或 `docker run -e` 中加入 `TZ=Asia/Shanghai` 后重建容器
- 主机时区用 `timedatectl set-timezone Asia/Shanghai` 校准

## 7.15 端口被占用，或者需要换监听地址？

后端监听地址由 `SERVER_ADDR`（或 `-addr`）决定，未设置时按 `ADDR`、`127.0.0.1:$PORT` 依次回退，默认 `127.0.0.1:8080`。

- 换端口：设置 `PORT=8090` 或 `SERVER_ADDR=127.0.0.1:8090`，重启后端；前端 `.env` 的 `VITE_API_BASE_URL` 同步改
- 容器内要对外暴露：`SERVER_ADDR` 设 `0.0.0.0:8080`，由容器内 Nginx 反代
- 查看占用：Linux `ss -lntp | grep 8080`，Windows `netstat -ano | findstr :8080`

## 7.16 为什么前端构建后不能只配静态目录？

因为控制台是 TanStack Start + Nitro 的 SSR 应用，页面由 `node .output/server/index.mjs` 返回，静态目录只提供 `/assets/` 等资源。SSR 服务同样需要被 Nginx 反代（见 [4.5](#45-配置-nginx-反向代理)）。

# 八、安全建议

- **先改默认密码** — 首次部署后立即修改 `admin` 的默认口令。
- **显式设置 JWT 密钥** — 不设置时系统会自动生成随机密钥并持久化（删库即全体会话失效）；多实例部署必须显式设置 `JWT_SECRET`。
- **启用 HTTPS** — 生产环境通过 Nginx 配置证书，参考 [3.7.2](#372-https-示例) 或 [4.4.2](#442-https-示例)。
- **收紧跨域** — 生产环境按需配置 `CORS_ORIGINS`，不要保留 `*`。
- **限制来源** — 通过 Nginx 或防火墙限定访问 IP 范围。
- **保护数据目录** — `data/` 内含数据库与日志，目录权限只授予运行服务的账号；`.env` 已在 `.gitignore` 中排除。
- **审计留痕** — 登录、改密、找回密码与配置变更全量写入审计日志，按「系统配置 → 基础配置 → 判定与保留」中的审计日志记录保留天数自动清理（0 表示不自动清理），导出归档由前端导出功能完成。

# 九、许可证

本项目采用 [MIT License](../LICENSE) 开源协议，可自由使用、复制、修改、合并、发布、分发、再许可与销售，只需在所有副本或重要部分中保留版权声明与许可声明。

# 十、致谢

感谢以下开源项目与技术社区：

- [go-chi/chi](https://github.com/go-chi/chi) — 轻量高效的 Go HTTP 路由框架
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — 纯 Go 实现的 SQLite 驱动
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket 客户端与服务端
- [swaggo/swag](https://github.com/swaggo/swag) — Swagger 文档生成工具
- [TanStack](https://tanstack.com/) — Router / Query / Start 前端框架套件
- [Tailwind CSS](https://tailwindcss.com/) — 原子化 CSS 框架
- [Radix UI](https://www.radix-ui.com/) — 无样式、可访问的组件基础

# 十一、联系方式

- **Email**：416685476@qq.com
- **GitHub Issues**：[zyx3721/backup-monitor/issues](https://github.com/zyx3721/backup-monitor/issues)
- **项目主页**：[github.com/zyx3721/backup-monitor](https://github.com/zyx3721/backup-monitor)

# 十二、备份监控业务与 Agent

## 12.1 业务对象

| 对象 | 关键字段 | 说明 |
| --- | --- | --- |
| 主机 `hosts` | id、hostname、node_id、ip、os、role、agent_version、agent_started_at、platform、arch、os_build、status、last_report_time、created_at | 以 hostname 为唯一键；节点 ID 变化视为重装 Agent；`platform` 由操作系统推断，空系统记为空（未知），未知平台不参与按平台推送 |
| 任务 `tasks` | id、host_id、host_name、name、task_key、type、tool、schedule、schedule_cron、schedule_locked、status、last_execution_time、config、created_at | `task_key` 用于与 crontab 行 / 计划任务稳定关联；连续失败次数与成功率按执行记录实时计算，不落库 |
| 执行记录 `executions` | task_id、session_id、job_id、oib_id、state、trigger、creation_time、start/end_time、sizes、artifact_path、exit_code、attempt、max_attempts、is_last_attempt | 以 `task_id + session_id` 幂等去重；Shell 等单次执行 `is_last_attempt=1`，Veeam 取会话详情 |
| 告警 `alerts` | host_id、task_id、type、level、status、message | 类型：`backup_failed` / `not_run` / `report_timeout` |
| 采集配置 `agent_profiles` | name、hostname（通配）、priority、enabled、config | 按优先级与匹配具体度选择 |
| Agent 令牌 `agent_tokens` | name、token、hostname、enabled、last_seen_at | 启用校验后所有 Agent 请求必须携带 |
| Agent 二进制 `agent_binaries` | platform、arch、variant、go_version、version、sha256、size、uploaded_at | 复合主键 `(platform, arch, variant)`；管理端托管 5 个槽位 |
| 通知渠道 `notify_channels` | type、event_type、level、remain_days、trigger_group、remark、config、enabled、recover | 支持 mail/webhook/wecom/dingtalk/feishu |
| 通知发送记录 `notifications` | channel、channel_id、channel_name、type、level、recipients、subject、status、error、host_name、task_name、message、created_at | 每次发送告警或测试通知写入一条，`status` 为 `sent` / `failed`，查询按创建时间倒序 |

## 12.2 状态判定

| 状态 | 条件 |
| --- | --- |
| 正常 `normal` | 最近一次计划窗口内存在成功执行，且主机在线 |
| 运行中 `running` | 存在未关闭的会话（`creation_time` 在窗口内） |
| 失败 `failed` | 最近一次执行为失败，或连续失败次数大于 0 |
| 未执行 `not_run` | 计划时刻 + 宽限期已过，窗口内没有任何执行记录 |
| 无数据 `no_data` | 任务尚无执行记录且未到判定时刻 |

主机状态由最近上报时间决定：超过「上报超时」判为异常，超过「离线阈值」判为离线。所有阈值位于 `basic_config` 的 `thresholds` 行（Agent 通道参数位于 `agent` 行），可在控制台调整，保存后通过 WebSocket 通知在线 Agent。

## 12.3 Agent 部署

```bash
# Linux：先写配置，再安装 systemd 服务
sudo mkdir -p /etc/backup-agent
sudo tee /etc/backup-agent/backup-agent.json >/dev/null <<'EOF'
{"serverUrl":"http://10.0.0.10:8080","token":"<TOKEN>","hostname":"","role":"database"}
EOF
sudo ./backup-agent -install
sudo systemctl status backup-agent

# Windows（管理员）：配置写入 %ProgramData%\backup-agent\backup-agent.json
.\backup-agent.exe -install
Get-Service backup-agent
```

常用参数：`-server`、`-token`、`-hostname`、`-role`、`-interval`、`-config`、`-log`、`-once`、`-service`、`-install`、`-uninstall`、`-version`。配置文件默认位于 `/etc/backup-agent/backup-agent.json`（Windows 为 `%ProgramData%\backup-agent\backup-agent.json`），示例见 `agent/backup-agent.example.json`；加载顺序为命令行 > 环境变量（`BM_SERVER_URL`、`BM_AGENT_TOKEN` 等）> 配置文件 > 默认值。

## 12.4 协议与接口

Agent 通道（认证头 `X-Agent-Token`，完整清单见 [5.2.6](#526-agent-通道)）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/ingest/report` | 上报主机、任务、执行记录 |
| GET | `/api/agent/config?hostname=&platform=` | 拉取全局参数与采集配置 |
| GET | `/api/agent/ws?hostname=&nodeId=&platform=` | WebSocket 长连接 |
| GET | `/api/agent/download?platform=&arch=&variant=` | 下载指定平台/架构/通道托管的 Agent 二进制（platform 必填，arch、variant 可选） |
| GET | `/api/agent/binary/latest?platform=&arch=&osBuild=` | 查询指定平台最新版本、架构与 SHA-256（platform 必填，arch、osBuild 可选） |

公开下载（无需令牌，供首次部署直接拉取）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/download/backup-agent?platform=&arch=&variant=` | 下载指定平台/架构/通道托管的 Agent 二进制，响应头含版本与 SHA-256 |

管理端接口（需登录与对应权限，完整清单见 [5.2.5](#525-备份监控)）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/backup/dashboard` | 概览统计与趋势 |
| GET | `/api/backup/hosts`、`/api/backup/hosts/{id}` | 主机列表与详情 |
| PUT | `/api/backup/hosts/{id}/role` | 修改主机角色 |
| GET | `/api/backup/tasks`、`/api/backup/tasks/{id}/executions` | 任务与执行历史 |
| PUT | `/api/backup/tasks/{id}/schedule` | 修改 cron 计划并下发 Agent |
| GET | `/api/backup/executions`、`/api/backup/executions/{id}` | 执行记录 |
| GET | `/api/backup/alerts`、`POST /api/backup/alerts/{id}/resolve` | 告警查询与恢复 |
| GET/POST/PUT/DELETE | `/api/backup/notify/channels` | 通知渠道维护 |
| GET | `/api/backup/notifications?limit=&channel=` | 通知发送记录（默认 100 条，最多 500 条；channel 可选） |
| GET/PUT | `/api/backup/settings` | 备份参数（判定阈值与保留策略，权限 `settings.base.read` / `settings.base.manage`） |
| GET/POST/PUT/DELETE | `/api/backup/profiles` | 采集配置 |
| GET/POST/PUT/DELETE | `/api/backup/tokens` | Agent 令牌 |
| GET | `/api/backup/agent-binaries` | 已托管二进制 |
| POST | `/api/backup/agent-binary?platform=&version=&variant=` | 上传二进制（原始字节，platform 与 version 必填，variant 可选；服务端校验 ELF/PE 并按 Go 版本归档槽位） |
| PUT | `/api/backup/agent-update/push` | 按平台与 IP 条件筛选主机，逐台按架构与 Windows 版本选包推送 |

## 12.5 常见运维操作

1. **新主机接入**：部署 Agent → 创建令牌（或关闭令牌校验）→ 确认「备份主机」出现记录。
2. **调整采集规则**：在「Agent 管理 → 采集配置」新建/修改 profile，保存后在线 Agent 立即重新拉取。
3. **修改执行计划**：在任务详情中编辑 cron，服务端通过 WebSocket 下发并等待 `set_cron_result` 回执。
4. **升级 Agent**：上传对应平台二进制 → 在「Agent 管理 → Agent 更新」选择平台，再按全部或 IP 范围推送 → 查看逐主机回执。Windows 首次需手工安装服务，之后可远程更新；旧版 Agent 未带平台参数，首次需手动推送或手工升级。
5. **告警排查**：按主机查看执行记录与产物路径；确认恢复后手动恢复告警，或等待下次成功执行自动恢复。

## 12.6 故障排查

| 现象 | 排查方向 |
| --- | --- |
| 主机一直离线 | Agent 是否运行、`server` 地址与网络连通性、反向代理 WebSocket 升级、令牌是否正确 |
| 任务显示未执行 | crontab 是否有对应行、`labelPrefix` 是否匹配、计划 cron 与服务器时区是否一致 |
| 修改计划失败 | Agent 是否在线、任务是否有 `task_key`、cron 是否为 5 段式 |
| 收不到通知 | 渠道是否启用、事件类型/级别是否匹配、SMTP 或 WebHook 连通性、通知渠道配置 |
| 二进制上传失败 | 平台参数是否为 `linux`/`windows`（缺失或非法返回 400）、文件类型是否匹配平台（Linux ELF / Windows PE）、文件是否超过 128MB |

