package swagger

// swaggerBackupHosts documents GET /api/backup/hosts.
// @Summary 获取备份主机列表
// @Tags 备份主机
// @Produce json
// @Param ids query string false "主机 ID，多个用逗号分隔；省略时返回全部"
// @Success 200 {array} swaggerHost
// @Failure 403 {object} swaggerErrorResponse
// @Description 返回备份主机及其状态、最近上报时间与任务数，可用 ids 只取指定主机（权限 backup.hosts.read）
// @Security BearerAuth
// @Router /api/backup/hosts [get]
func swaggerBackupHosts() {}

// swaggerBackupHostRefreshPlan documents GET /api/backup/hosts/refresh-plan.
// @Summary 获取主机刷新计划
// @Tags 备份主机
// @Produce json
// @Success 200 {object} swaggerHostRefreshPlan
// @Failure 403 {object} swaggerErrorResponse
// @Description 按主机实际匹配到的上报间隔分组，供前端按组局部刷新（权限 backup.hosts.read）
// @Security BearerAuth
// @Router /api/backup/hosts/refresh-plan [get]
func swaggerBackupHostRefreshPlan() {}

// swaggerBackupHostDetail documents GET /api/backup/hosts/{id}.
// @Summary 获取备份主机详情
// @Tags 备份主机
// @Produce json
// @Param id path string true "主机 ID"
// @Success 200 {object} swaggerHostDetail
// @Failure 404 {object} swaggerErrorResponse
// @Description 返回指定主机及其名下任务列表（权限 backup.hosts.read）
// @Security BearerAuth
// @Router /api/backup/hosts/{id} [get]
func swaggerBackupHostDetail() {}

// swaggerBackupHostRole documents PUT /api/backup/hosts/{id}/role.
// @Summary 修改主机角色
// @Tags 备份主机
// @Accept json
// @Produce json
// @Param id path string true "主机 ID"
// @Param request body swaggerHostRoleRequest true "主机角色"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 修改主机角色并写入审计（权限 backup.hosts.manage）
// @Security BearerAuth
// @Router /api/backup/hosts/{id}/role [put]
func swaggerBackupHostRole() {}

// swaggerBackupHostDelete documents DELETE /api/backup/hosts/{id}.
// @Summary 删除备份主机
// @Tags 备份主机
// @Produce json
// @Param id path string true "主机 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除主机并级联清理其任务、执行记录与告警，详见审计详情（权限 backup.hosts.manage）
// @Security BearerAuth
// @Router /api/backup/hosts/{id} [delete]
func swaggerBackupHostDelete() {}

// swaggerBackupTasks documents GET /api/backup/tasks.
// @Summary 获取备份任务列表
// @Tags 备份任务
// @Produce json
// @Success 200 {array} swaggerTask
// @Description 返回全部备份任务及其计划、状态与所属主机（权限 backup.tasks.read）
// @Security BearerAuth
// @Router /api/backup/tasks [get]
func swaggerBackupTasks() {}

// swaggerBackupTaskDetail documents GET /api/backup/tasks/{id}.
// @Summary 获取备份任务详情
// @Tags 备份任务
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {object} swaggerTask
// @Failure 404 {object} swaggerErrorResponse
// @Description 返回指定备份任务（权限 backup.tasks.read）
// @Security BearerAuth
// @Router /api/backup/tasks/{id} [get]
func swaggerBackupTaskDetail() {}

// swaggerBackupTaskExecutions documents GET /api/backup/tasks/{id}/executions.
// @Summary 获取任务执行历史
// @Tags 备份任务
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {array} swaggerExecution
// @Failure 404 {object} swaggerErrorResponse
// @Description 返回指定任务的执行记录列表（权限 backup.tasks.read）
// @Security BearerAuth
// @Router /api/backup/tasks/{id}/executions [get]
func swaggerBackupTaskExecutions() {}

// swaggerBackupTaskDelete documents DELETE /api/backup/tasks/{id}.
// @Summary 删除备份任务
// @Tags 备份任务
// @Produce json
// @Param id path string true "任务 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除任务并级联清理执行记录与告警（权限 backup.tasks.manage）
// @Security BearerAuth
// @Router /api/backup/tasks/{id} [delete]
func swaggerBackupTaskDelete() {}

// swaggerBackupTaskSchedule documents PUT /api/backup/tasks/{id}/schedule.
// @Summary 修改执行计划
// @Tags 备份任务
// @Accept json
// @Produce json
// @Param id path string true "任务 ID"
// @Param request body swaggerTaskScheduleRequest true "新的 cron 表达式"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 修改 cron 计划并下发 Agent，仅主机心跳在线时成功；计划未变化时直接返回（权限 backup.tasks.manage）
// @Security BearerAuth
// @Router /api/backup/tasks/{id}/schedule [put]
func swaggerBackupTaskSchedule() {}

// swaggerBackupExecutions documents GET /api/backup/executions.
// @Summary 获取执行记录列表
// @Tags 执行记录
// @Produce json
// @Param limit query int false "返回条数上限，0 或省略表示全部"
// @Success 200 {array} swaggerExecution
// @Description 返回全部执行记录及其任务、主机与结果（权限 backup.executions.read）
// @Security BearerAuth
// @Router /api/backup/executions [get]
func swaggerBackupExecutions() {}

// swaggerBackupExecutionDetail documents GET /api/backup/executions/{id}.
// @Summary 获取执行记录详情
// @Tags 执行记录
// @Produce json
// @Param id path string true "执行记录 ID"
// @Success 200 {object} swaggerExecution
// @Failure 404 {object} swaggerErrorResponse
// @Description 返回指定执行记录（权限 backup.executions.read）
// @Security BearerAuth
// @Router /api/backup/executions/{id} [get]
func swaggerBackupExecutionDetail() {}

// swaggerBackupExecutionDelete documents DELETE /api/backup/executions/{id}.
// @Summary 删除执行记录
// @Tags 执行记录
// @Produce json
// @Param id path string true "执行记录 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除指定执行记录并写入审计（权限 backup.executions.manage）
// @Security BearerAuth
// @Router /api/backup/executions/{id} [delete]
func swaggerBackupExecutionDelete() {}

// swaggerBackupAlerts documents GET /api/backup/alerts.
// @Summary 获取告警事件列表
// @Tags 告警中心
// @Produce json
// @Success 200 {array} swaggerAlert
// @Description 返回全部告警事件及其状态、级别与关联任务（权限 backup.alerts.read）
// @Security BearerAuth
// @Router /api/backup/alerts [get]
func swaggerBackupAlerts() {}

// swaggerBackupAlertResolve documents POST /api/backup/alerts/{id}/resolve.
// @Summary 恢复告警
// @Tags 告警中心
// @Produce json
// @Param id path string true "告警 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 手动恢复指定告警并写入审计（权限 backup.alerts.manage）
// @Security BearerAuth
// @Router /api/backup/alerts/{id}/resolve [post]
func swaggerBackupAlertResolve() {}

// swaggerBackupAlertDelete documents DELETE /api/backup/alerts/{id}.
// @Summary 删除告警
// @Tags 告警中心
// @Produce json
// @Param id path string true "告警 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除指定告警并写入审计（权限 backup.alerts.manage）
// @Security BearerAuth
// @Router /api/backup/alerts/{id} [delete]
func swaggerBackupAlertDelete() {}

// swaggerBackupDashboard documents GET /api/backup/dashboard.
// @Summary 获取概览统计
// @Tags 概览
// @Produce json
// @Success 200 {object} swaggerGenericObject
// @Description 返回主机、任务、今日执行与告警的汇总、状态分布与每日趋势（权限 backup.dashboard.read）
// @Security BearerAuth
// @Router /api/backup/dashboard [get]
func swaggerBackupDashboard() {}

// swaggerBackupSettings documents GET /api/backup/settings.
// @Summary 获取备份参数
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerGenericObject
// @Description 返回判定阈值、保留策略与 Agent 通道参数（权限 settings.base.read）
// @Security BearerAuth
// @Router /api/backup/settings [get]
func swaggerBackupSettings() {}

// swaggerBackupSettingsPut documents PUT /api/backup/settings.
// @Summary 更新备份参数
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerGenericObject true "备份参数，section 取值 thresholds/agent/alarm"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Description 按 section 更新判定与保留、Agent 通道或安全与告警参数并写入审计（权限 settings.base.manage）
// @Security BearerAuth
// @Router /api/backup/settings [put]
func swaggerBackupSettingsPut() {}

// swaggerBackupProfiles documents GET /api/backup/profiles.
// @Summary 获取采集配置列表
// @Tags Agent 管理
// @Produce json
// @Success 200 {array} swaggerAgentProfile
// @Description 返回全部采集配置及其匹配主机、优先级与采集项（权限 backup.agents.read）
// @Security BearerAuth
// @Router /api/backup/profiles [get]
func swaggerBackupProfiles() {}

// swaggerBackupProfileCreate documents POST /api/backup/profiles.
// @Summary 创建采集配置
// @Tags Agent 管理
// @Accept json
// @Produce json
// @Param request body swaggerAgentProfileRequest true "采集配置"
// @Success 200 {object} swaggerAgentProfile
// @Failure 400 {object} swaggerErrorResponse
// @Description 创建采集配置并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/profiles [post]
func swaggerBackupProfileCreate() {}

// swaggerBackupProfileUpdate documents PUT /api/backup/profiles/{id}.
// @Summary 更新采集配置
// @Tags Agent 管理
// @Accept json
// @Produce json
// @Param id path string true "采集配置 ID"
// @Param request body swaggerAgentProfileRequest true "采集配置"
// @Success 200 {object} swaggerAgentProfile
// @Failure 404 {object} swaggerErrorResponse
// @Description 更新采集配置并写入审计，未改动时不写审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/profiles/{id} [put]
func swaggerBackupProfileUpdate() {}

// swaggerBackupProfileDelete documents DELETE /api/backup/profiles/{id}.
// @Summary 删除采集配置
// @Tags Agent 管理
// @Produce json
// @Param id path string true "采集配置 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除采集配置并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/profiles/{id} [delete]
func swaggerBackupProfileDelete() {}

// swaggerBackupTokens documents GET /api/backup/tokens.
// @Summary 获取 Agent 令牌列表
// @Tags Agent 管理
// @Produce json
// @Success 200 {array} swaggerAgentToken
// @Description 返回 Agent 访问令牌及其启用状态与最近使用时间（权限 backup.agents.read）
// @Security BearerAuth
// @Router /api/backup/tokens [get]
func swaggerBackupTokens() {}

// swaggerBackupTokenCreate documents POST /api/backup/tokens.
// @Summary 创建 Agent 令牌
// @Tags Agent 管理
// @Accept json
// @Produce json
// @Param request body swaggerAgentTokenCreateRequest true "令牌名称、绑定主机与可选的自定义令牌"
// @Success 200 {object} swaggerAgentTokenCreateResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Description 创建 Agent 访问令牌，token 留空时自动生成 48 位随机值，自定义值需 16-128 位可见 ASCII 且不得重复；仅在创建响应中返回明文令牌并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/tokens [post]
func swaggerBackupTokenCreate() {}

// swaggerBackupTokenUpdate documents PUT /api/backup/tokens/{id}.
// @Summary 更新 Agent 令牌
// @Tags Agent 管理
// @Accept json
// @Produce json
// @Param id path string true "令牌 ID"
// @Param request body swaggerAgentTokenRequest true "令牌名称、绑定主机与启用状态"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 更新令牌名称、绑定主机或启用状态并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/tokens/{id} [put]
func swaggerBackupTokenUpdate() {}

// swaggerBackupTokenDelete documents DELETE /api/backup/tokens/{id}.
// @Summary 删除 Agent 令牌
// @Tags Agent 管理
// @Produce json
// @Param id path string true "令牌 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除 Agent 令牌并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/tokens/{id} [delete]
func swaggerBackupTokenDelete() {}

// swaggerBackupChannels documents GET /api/backup/notify/channels.
// @Summary 获取通知渠道列表
// @Tags 通知管理
// @Produce json
// @Success 200 {array} swaggerNotifyChannel
// @Description 返回通知渠道及其方式、订阅事件、级别与启用状态（权限 backup.notifications.read）
// @Security BearerAuth
// @Router /api/backup/notify/channels [get]
func swaggerBackupChannels() {}

// swaggerBackupChannelCreate documents POST /api/backup/notify/channels.
// @Summary 创建通知渠道
// @Tags 通知管理
// @Accept json
// @Produce json
// @Param request body swaggerNotifyChannel true "通知渠道"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Description 创建通知渠道并写入审计（权限 backup.notifications.manage）
// @Security BearerAuth
// @Router /api/backup/notify/channels [post]
func swaggerBackupChannelCreate() {}

// swaggerBackupChannelUpdate documents PUT /api/backup/notify/channels/{id}.
// @Summary 更新通知渠道
// @Tags 通知管理
// @Accept json
// @Produce json
// @Param id path string true "通知渠道 ID"
// @Param request body swaggerNotifyChannel true "通知渠道"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 更新或启停用通知渠道并写入审计；未改动时不写审计（权限 backup.notifications.manage）
// @Security BearerAuth
// @Router /api/backup/notify/channels/{id} [put]
func swaggerBackupChannelUpdate() {}

// swaggerBackupChannelDelete documents DELETE /api/backup/notify/channels/{id}.
// @Summary 删除通知渠道
// @Tags 通知管理
// @Produce json
// @Param id path string true "通知渠道 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 删除通知渠道并写入审计（权限 backup.notifications.manage）
// @Security BearerAuth
// @Router /api/backup/notify/channels/{id} [delete]
func swaggerBackupChannelDelete() {}

// swaggerBackupChannelTest documents POST /api/backup/notify/channels/{id}/test.
// @Summary 测试通知渠道
// @Tags 通知管理
// @Produce json
// @Param id path string true "通知渠道 ID"
// @Success 200 {object} swaggerGenericObject
// @Description 按渠道配置发送测试通知，成功与失败均写入审计（权限 backup.notifications.manage）
// @Security BearerAuth
// @Router /api/backup/notify/channels/{id}/test [post]
func swaggerBackupChannelTest() {}

// swaggerBackupNotifications documents GET /api/backup/notifications.
// @Summary 获取通知发送记录
// @Tags 通知管理
// @Produce json
// @Param limit query int false "返回条数上限，默认 100，最大 500"
// @Param channel query string false "按通知渠道 ID 过滤，省略时返回全部"
// @Success 200 {array} swaggerNotificationRecord
// @Description 返回通知发送记录及其渠道、状态与错误信息（权限 backup.notifications.read）
// @Security BearerAuth
// @Router /api/backup/notifications [get]
func swaggerBackupNotifications() {}

// swaggerBackupAgentBinaries documents GET /api/backup/agent-binaries.
// @Summary 获取已托管 Agent 二进制
// @Tags Agent 管理
// @Produce json
// @Success 200 {array} swaggerAgentBinaryMeta
// @Description 返回各槽位（平台 + 架构 + 通道）已托管的版本、Go 版本、大小与 SHA-256（权限 backup.agents.read）
// @Security BearerAuth
// @Router /api/backup/agent-binaries [get]
func swaggerBackupAgentBinaries() {}

// swaggerBackupAgentBinaryUpload documents POST /api/backup/agent-binary.
// @Summary 上传 Agent 二进制
// @Tags Agent 管理
// @Accept octet-stream
// @Produce json
// @Param platform query string true "平台，取值 linux 或 windows"
// @Param variant query string false "通道，取值 standard 或 legacy；缺省按 Go 版本推导（Windows amd64 且 Go < 1.21 归入 legacy）"
// @Param version query string true "版本号"
// @Param file body string true "二进制原始字节（服务端校验 ELF/PE）"
// @Success 200 {object} swaggerAgentBinaryMeta
// @Failure 400 {object} swaggerErrorResponse
// @Description 上传指定平台与通道的 Agent 二进制并写入审计，支持 amd64 与 arm64（服务端按文件头解析架构、按 buildinfo 读取 Go 版本），单次不超过 128MB（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/agent-binary [post]
func swaggerBackupAgentBinaryUpload() {}

// swaggerBackupAgentPush documents PUT /api/backup/agent-update/push.
// @Summary 推送 Agent 自更新
// @Tags Agent 管理
// @Accept json
// @Produce json
// @Param request body swaggerAgentBinaryPushRequest true "平台与目标主机（全部或 IP 条件）"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 按平台与 IP 条件逐台选择匹配的槽位（架构 + Windows 版本）下发自更新并写入审计（权限 backup.agents.manage）
// @Security BearerAuth
// @Router /api/backup/agent-update/push [put]
func swaggerBackupAgentPush() {}

// swaggerSettingsPermissions documents GET /api/settings/permissions.
// @Summary 获取权限目录
// @Tags 系统设置
// @Produce json
// @Success 200 {array} swaggerGenericObject
// @Description 返回权限目录全量清单（Key、名称、描述、分类与隐含关系）
// @Security BearerAuth
// @Router /api/settings/permissions [get]
func swaggerSettingsPermissions() {}

// swaggerSettingsEmailTest documents POST /api/settings/email/test.
// @Summary 发送测试邮件
// @Tags 系统设置
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body swaggerGenericObject true "测试收件人，字段 to"
// @Success 200 {object} swaggerGenericObject
// @Failure 502 {object} swaggerErrorResponse
// @Description 按当前邮件配置向指定收件人发送测试邮件并写入审计（权限 settings.notifications.manage）
// @Router /api/settings/email/test [post]
func swaggerSettingsEmailTest() {}

// swaggerSettingsAuthProviderTest documents POST /api/settings/auth-provider/test.
// @Summary 测试 LDAP 连通性
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerGenericObject true "待测试的 LDAP 连接参数"
// @Success 200 {object} swaggerGenericObject
// @Failure 502 {object} swaggerErrorResponse
// @Description 按请求体或当前配置发起 LDAP 连通性测试（权限 settings.auth.manage）
// @Security BearerAuth
// @Router /api/settings/auth-provider/test [post]
func swaggerSettingsAuthProviderTest() {}

// swaggerHistoryExport documents POST /api/history/export.
// @Summary 记录数据导出
// @Tags 审计日志
// @Accept json
// @Produce json
// @Param request body swaggerHistoryExportRequest true "导出模块与记录条数"
// @Success 200 {object} swaggerOKResponse
// @Failure 400 {object} swaggerErrorResponse
// @Description 各页面导出完成后回写一条审计，模块取值 hosts/tasks/executions/alerts/audit
// @Security BearerAuth
// @Router /api/history/export [post]
func swaggerHistoryExport() {}

// swaggerAgentReport documents POST /api/ingest/report.
// @Summary Agent 上报数据
// @Tags Agent 通道
// @Accept json
// @Produce json
// @Param X-Agent-Token header string false "Agent 访问令牌，启用校验时必填"
// @Param request body swaggerGenericObject true "主机、任务与执行记录"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 接收 Agent 上报的主机、任务与执行记录，按 task+session 幂等更新；协议见 docs/agent-protocol.md
// @Router /api/ingest/report [post]
func swaggerAgentReport() {}

// swaggerAgentConfig documents GET /api/agent/config.
// @Summary 拉取 Agent 配置
// @Tags Agent 通道
// @Produce json
// @Param hostname query string true "主机名"
// @Param platform query string false "平台，取值 linux 或 windows"
// @Param X-Agent-Token header string false "Agent 访问令牌，启用校验时必填"
// @Success 200 {object} swaggerGenericObject
// @Failure 400 {object} swaggerErrorResponse
// @Description 返回全局参数、采集配置与二进制更新信息供 Agent 应用
// @Router /api/agent/config [get]
func swaggerAgentConfig() {}

// swaggerAgentWS documents GET /api/agent/ws.
// @Summary Agent 心跳长连接
// @Tags Agent 通道
// @Produce json
// @Param hostname query string true "主机名"
// @Param nodeId query string false "稳定机器标识"
// @Param platform query string false "平台，取值 linux 或 windows"
// @Param X-Agent-Token header string false "Agent 访问令牌，启用校验时必填"
// @Success 200 {string} string "协议升级为 WebSocket"
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 建立 WebSocket 长连接，承载心跳、配置变更与 set_cron/update 命令回执
// @Router /api/agent/ws [get]
func swaggerAgentWS() {}

// swaggerAgentDownload documents GET /api/agent/download.
// @Summary 下载托管 Agent 二进制
// @Tags Agent 通道
// @Produce octet-stream
// @Param platform query string true "平台，取值 linux 或 windows"
// @Param arch query string false "CPU 架构，取值 amd64 或 arm64；省略时按 osBuild 自动选包"
// @Param variant query string false "通道，取值 standard 或 legacy；与 arch 同时指定时精确取槽位"
// @Param osBuild query int false "Windows 构建号，用于在未精确指定槽位时自动选包"
// @Param X-Agent-Token header string false "Agent 访问令牌，启用校验时必填"
// @Success 200 {file} file "二进制文件"
// @Failure 404 {object} swaggerErrorResponse
// @Description 下载匹配主机架构与系统版本的托管 Agent 二进制；精确指定 arch+variant 时取对应槽位，否则按 osBuild 自动选包
// @Router /api/agent/download [get]
func swaggerAgentDownload() {}

// swaggerAgentBinaryLatest documents GET /api/agent/binary/latest.
// @Summary 查询最新 Agent 版本
// @Tags Agent 通道
// @Produce json
// @Param platform query string true "平台，取值 linux 或 windows"
// @Param arch query string false "CPU 架构，取值 amd64 或 arm64；省略时按 amd64 处理"
// @Param osBuild query int false "Windows 构建号，用于自动选包"
// @Param X-Agent-Token header string false "Agent 访问令牌，启用校验时必填"
// @Success 200 {object} swaggerAgentBinaryLatestResponse
// @Failure 404 {object} swaggerErrorResponse
// @Description 按平台、架构与系统版本选择槽位，返回对应安装包的版本号、大小、SHA-256 与通道
// @Router /api/agent/binary/latest [get]
func swaggerAgentBinaryLatest() {}

// swaggerPublicAgentDownload documents GET /download/backup-agent.
// @Summary 公开下载 Agent 二进制
// @Tags Agent 通道
// @Produce octet-stream
// @Param platform query string true "平台，取值 linux 或 windows"
// @Param arch query string false "CPU 架构，取值 amd64 或 arm64"
// @Param variant query string false "通道，取值 standard 或 legacy"
// @Param osBuild query int false "Windows 构建号，用于自动选包"
// @Success 200 {file} file "二进制文件，响应头含版本与 SHA-256"
// @Failure 404 {object} swaggerErrorResponse
// @Description 无需令牌的公开下载入口，按平台、架构与系统版本选择槽位，供首次部署直接拉取托管二进制
// @Router /download/backup-agent [get]
func swaggerPublicAgentDownload() {}
