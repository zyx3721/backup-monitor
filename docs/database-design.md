# Backup Monitor — 数据库规划分析

> **文档说明**：本文档聚焦数据库**设计取舍**，不是完整 schema 清单；完整表结构与列定义以 [docs/manual.md](manual.md) 第 6、12 章为准。本文档回应两个数据库设计问题——①系统配置各模块带子模块时，SQLite 应采用「一张总配置表 / 每模块一表 / 每子模块一表」的哪种规划；②key-value 式配置表在多用户同时保存时的并发影响。结合本项目当前实现逐项对照，并给出已落地与可选的增强项。
>
> 项目概览见 [README.md](../README.md)，历史抽取过程见 [extraction-plan.md](extraction-plan.md)。

## 一、当前表规划清单

### 1.1 配置类（三表统一「name 子模块标识 + config JSON」形态，非扁平 key-value 行）

| 表 | 承载内容 | 存储形态 |
| --- | --- | --- |
| `basic_config` | 基础配置 | 每个子模块一行：`name='brand'`（品牌标识五字段）/ `name='security'`（安全时效七字段）/ `name='thresholds'`（判定与保留七字段，含执行记录保留与审计日志保留）/ `name='agent'`（Agent 通道四字段）/ `name='alarm'`（安全与告警两字段），各自行内 `config` JSON；按 section 保存只写对应行 |
| `auth_config` | 认证配置 | 每个认证方式一行：`name='ldap'/'wecom'`，`title` 为显示名（AD/LDAP、企业微信），`enabled` 开关 + `config` JSON；`bindPassword` 等敏感项以 `enc:v1` 密文存于 JSON 内 |
| `pwd_recover_config` | 找回密码配置 | 每个找回密码媒介一行：`name='email'`（`title='邮件通知'`），`enabled` 即原找回密码邮件开关（后续新增短信/webhook 等子模块时插行即可，无需改表） |
| `system_secrets` | JWT 签名密钥 | 单行，启动时自动生成并持久化 |

### 1.2 业务实体类（独立表，与配置严格分离，已去 `settings_` 前缀）

| 表 | 说明 |
| --- | --- |
| `users` | 用户实体（`username` 建有 `LOWER(username)` 唯一表达式索引）；邮箱、禁用、企微绑定（`wecom_userid` UNIQUE）、创建/更新/最近登录时间全部并入本表，无 usertype 列——管理员身份由「admin 用户名 + 内置 admin 角色」判定 |
| `roles` | 角色实体（`permissions` JSON 数组、`disabled` 状态列，内置 admin/operator/viewer 种子） |
| `user_roles` | 用户-角色多对多关联（必须独立建表，无法并入实体表） |
| `user_groups` | 用户群组 |
| `user_group_members` / `user_group_roles` | 群组成员、群组-角色多对多关联（同上保留独立表） |

### 1.3 审计与流程类

| 表 | 说明 |
| --- | --- |
| `audit_logs` | 审计日志（结构化 module/action/target/detail/result + 原始 SQL，按基础配置中的审计日志保留天数自动清理；`date` 建有索引；由原 `history` 更名） |
| `reset_captchas` / `_requests` / `_send_log` | 找回密码验证码、请求与限频记录 |

### 1.4 时间字段约定

所有时间列统一以 `TEXT` 存储 UTC 毫秒 ISO 文本（如 `2026-10-22T06:39:34.691Z`），定长格式保证字符串排序即时间排序；接口层按原契约返回（审计日志仍返回秒级时间戳数字），前端展示格式不变。

## 二、对照「三方案」的定位分析

| 方案 | 描述 | 本项目采用度 |
| --- | --- | --- |
| 方案一：单张 key-value 总表 | module/sub_module/config_key 三级键，一行一个配置项 | **未采用**，但获得了其核心优点（见下） |
| 方案二：每个一级模块一张配置表 | sys_base_config / sys_auth_config / sys_notify_config | **本质上是它的变体**：每个配置域一张表 |
| 方案三：每个子模块一张表 | 表数量随子模块膨胀 | 未采用（避免了维护成本最高的形态） |

关键差异：本项目的配置表没有走「每行一个 config_key」的扁平 key-value，而是**每个配置域一行（或少数几行）+ 整份 JSON 文档**。这带来三个比扁平 key-value 更优的性质：

1. **新增子模块零迁移**——品牌加字段、通知加媒介，都只是 JSON 内加键，不改表（方案一的优点保留）；
2. **事务更原子**——保存一个区块是一次单行 UPDATE，而不是 N 行 upsert；「同一份配置的多个字段」天然一致，不存在半提交状态；
3. **读写代码更薄**——一次主键点查得到整份配置，无需按 key 拼装。

与豆包结论一致的核心原则本项目严格遵循：**业务实体（用户、群组、角色）绝不混入配置表**，全部独立建表；「用户配置」这个菜单下的用户/群组/角色是实体而非配置项。

## 三、并发对照（多用户同时保存）

| 豆包建议 | 本项目现状 |
| --- | --- |
| 开启 WAL（读不阻塞写） | ✅ `SetupSQLite` 已设 `journal_mode = WAL` + `busy_timeout = 5000` + `synchronous = NORMAL`，且连接固定 `MaxOpenConns(1)` 串行化写请求，从源头规避「database is locked」 |
| 乐观锁 version 防覆盖 | ⚠️ 部分：`basic_config` 按子模块分为 5 行存储（品牌标识、安全时效、判定与保留、Agent 通道、安全与告警），各区块**物理隔离在不同行**，保存互不覆盖（比单行合并更强）；剩余风险仅「两名管理员同时编辑同一子模块」，内部后台（≤10 人）可接受 |
| LDAP 测试连接不得包在保存事务里 | ✅ 测试是独立端点 `POST /api/settings/auth-provider/test`，与保存分离 |
| 事务短小、无 IO | ✅ 保存路径均为单行 UPDATE |
| 用户名唯一索引兜底并发新增 | ✅ 已补：`idx_users_username_lower`（`LOWER(username)` 唯一表达式索引），建库 SQL 与存量迁移 `EnsureUsersUsernameUniqueIndex` 双路径生效 |
| 重要配置修改写审计 | ✅ 配置类保存均产生显式审计事件（含区块细分目标） |
| `updated_at` 自动维护 | ✅ 各表由代码以 `strftime('%s','now')` 写入 |
| 读多写少吃内存缓存 | ✅ 等效达成：读路径为主键点查（WAL 下不阻塞写），前端另有 30 秒品牌缓存与 1.5 秒当前用户缓存；后端再叠一层内存缓存收益极低，**不建议引入**（多进程失效一致性复杂度不划算） |

## 四、结论与可选增强

**结论**：当前「每配置域一表 + JSON 文档列 + 业务实体独立表」的混合规划，同时取得方案二的锁竞争隔离（不同配置域互不相干的单行写）与方案一的扩展性（加子模块零迁移），并规避了方案三的表泛滥；配合 WAL、busy_timeout、单连接串行写、区块合并保存与唯一索引兜底，对内部控制台并发量级（≤10 人同时在线）完全充分，无需迁移到 MySQL/PostgreSQL。

**可选增强**（按需启用，当前默认不做）：

1. **配置乐观锁**：在 `basic_config` / `auth_config` / `pwd_recover_config` 的配置行增加 `version` 列，GET 返回版本、PUT 带 `WHERE version=?`，影响行数为 0 时返回「配置已被其他用户修改，请刷新后重试」。仅当出现「同区块互踩」的真实痛点时启用。
2. **JSON 校验层**：配置保存前用结构体反序列化校验（当前已由 domain 层校验函数承担，无需 JSON Schema）。
3. 若未来出现几十人高频同时编辑的规模，再评估迁移 PostgreSQL（行级锁），届时 repository 层的 SQL 兼容性良好（未使用 SQLite 方言的查询仅限 PRAGMA 与 `strftime` 少量点）。

## 五、备份监控业务表

备份监控数据与基础平台数据同库，由 `repository.EnsureBackupSchema` 在启动时幂等建表：

| 表 | 关键列 | 说明 |
| --- | --- | --- |
| `hosts` | `id`、`hostname` UNIQUE、`node_id`、`ip`、`os`、`role`、`agent_version`、`agent_started_at`、`platform`、`arch`、`os_build`、`status`、`last_report_time`、`created_at` | Agent 上报时 upsert；`node_id` 变化用于识别重装；`role` 仅首次注册写入，之后以管理端配置为准 |
| `tasks` | `id`、`host_id` FK、`name`、`task_key`、`type`、`tool`、`schedule`、`schedule_cron`、`schedule_locked`、`status`、`last_execution_time`、`config` JSON | 唯一约束 `(host_id, name)`；`task_key` 与采集源关联；连续失败次数与成功率按执行记录实时计算，不落库 |
| `executions` | `id`、`task_id` FK、`session_id`、`job_id`、`oib_id`、`state`、`trigger`、`creation_time`、`start_time`、`end_time`、`processed_size`、`transferred_size`、`artifact_path`、`exit_code`、`attempt`、`max_attempts`、`is_last_attempt`、`message` | 唯一约束 `(task_id, session_id)` 保证上报幂等；按 `creation_time` 建索引 |
| `alerts` | `id`、`host_id`、`task_id`、`type`、`level`、`status`、`created_at`、`resolved_at`、`message` | 部分索引覆盖未处理告警；去重时复用同对象同类型的 active 记录 |
| `agent_profiles` | `id`、`name` UNIQUE、`hostname`、`priority`、`enabled`、`config` JSON | 按优先级排序后做通配匹配，`*` 为兜底 |
| `agent_tokens` | `id`、`token` UNIQUE、`name`、`hostname`、`enabled`、`last_seen_at` | 令牌明文仅用于校验，接口可返回给管理员复制 |
| `agent_binaries` | `platform`、`arch`、`variant`、`go_version`、`version`、`sha256`、`size`、`uploaded_at` | 复合主键 `(platform, arch, variant)`；管理端托管 5 个槽位，二进制字节存放于 `data/agents/<platform>/` |
| `notify_channels` | `id`、`type`、`name`、`event_type` JSON、`level`、`remain_days`、`trigger_group`、`remark`、`enabled`、`recover`、`config` JSON | 渠道配置含密钥时复用 `enc:v1` 加密 |
| `notifications` | `id`、`channel_id`、`channel`、`type`、`level`、`recipients`、`subject`、`status`、`error`、`host_name`、`task_name`、`message`、`created_at` | 发送结果审计，失败保留错误原因 |

设计取舍：

- 判定阈值、Agent 通道参数与告警开关不再单独建配置表，直接落在 `basic_config` 的 `thresholds` / `agent` / `alarm` 三行，与控制台「基础配置」的五个面板一一对应。
- 执行记录是增长最快的表，按 `retention_days` 每小时清理一次，并在清理后执行 WAL checkpoint。
- 告警与执行记录均带 `host_id`/`task_id` 冗余字段，删除主机或任务时显式级联删除，避免依赖 SQLite 外键级联的开启状态。
- 任务状态、连续失败次数与成功率是派生数据，由上报与巡检重算后落库，查询时无需聚合全表。

