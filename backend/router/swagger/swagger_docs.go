package swagger

// swaggerHealthRoot documents GET /health.
// @Summary 健康检查
// @Tags 健康检查
// @Produce json
// @Success 200 {object} swaggerHealthResponse
// @Description 探测服务进程是否存活，无需认证
// @Router /health [get]
func swaggerHealthRoot() {}

// swaggerHealthAPI documents GET /api/health.
// @Summary API 健康检查
// @Tags 健康检查
// @Produce json
// @Success 200 {object} swaggerHealthResponse
// @Description 探测服务进程是否存活，无需认证
// @Router /api/health [get]
func swaggerHealthAPI() {}

// swaggerPublicAuthProviders documents GET /api/auth/providers.
// @Summary 获取公开认证方式
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerPublicAuthProvidersResponse
// @Description 获取登录页可用的认证方式与找回密码开关，无需认证
// @Router /api/auth/providers [get]
func swaggerPublicAuthProviders() {}

// swaggerLogin documents POST /api/auth/login.
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body authLoginRequest true "登录信息"
// @Success 200 {object} authLoginResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 429 {object} swaggerErrorResponse
// @Description 用户登录，支持本地密码与 AD/LDAP 两种方式，成功返回令牌与用户信息；同一账号密码连续错误达到阈值后锁定，锁定期间即使密码正确也返回 429 与剩余等待分钟（内置 admin 不受限，企业微信扫码不涉及）
// @Router /api/auth/login [post]
func swaggerLogin() {}

// swaggerWecomAuthorize documents GET /api/auth/wecom/authorize.
// @Summary 获取企业微信扫码登录地址
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerWecomAuthorizeResponse
// @Failure 400 {object} swaggerErrorResponse
// @Description 生成企业微信 Web 扫码登录页地址（含防伪 state）与内嵌二维码渲染参数（iframe_url、callback_path）；直连模式内嵌扫码回跳 /wecom-qr-callback，统一认证中心模式回跳 /login。登录页默认内嵌渲染二维码，配置异常时自动回退整页跳转；需已启用企业微信认证，无需认证
// @Router /api/auth/wecom/authorize [get]
func swaggerWecomAuthorize() {}

// swaggerWecomCallback documents POST /api/auth/wecom/callback.
// @Summary 企业微信扫码登录回调
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerWecomCallbackRequest true "扫码回调载荷"
// @Success 200 {object} authLoginResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 校验 state 后用授权码换取企业微信成员身份，按绑定关系登录并返回令牌与用户信息；未绑定时返回 401
// @Router /api/auth/wecom/callback [post]
func swaggerWecomCallback() {}

// swaggerWecomBindURL documents GET /api/auth/wecom/bind-url.
// @Summary 获取企业微信绑定扫码地址
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerWecomAuthorizeResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 为当前登录用户生成企业微信绑定扫码地址，state 绑定当前用户；需已启用企业微信认证
// @Security BearerAuth
// @Router /api/auth/wecom/bind-url [get]
func swaggerWecomBindURL() {}

// swaggerWecomBind documents POST /api/auth/wecom/bind.
// @Summary 绑定企业微信账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerWecomCallbackRequest true "扫码回调载荷"
// @Success 200 {object} swaggerWecomBindResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Description 校验绑定 state 后用授权码换取企业微信成员 userid，与当前用户建立绑定；该企微已绑定其他用户时返回 409
// @Security BearerAuth
// @Router /api/auth/wecom/bind [post]
func swaggerWecomBind() {}

// swaggerWecomUnbind documents DELETE /api/auth/wecom/bind.
// @Summary 解绑企业微信账号
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerOKResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 解除当前登录用户的企业微信绑定
// @Security BearerAuth
// @Router /api/auth/wecom/bind [delete]
func swaggerWecomUnbind() {}

// swaggerWecomSSOCallback documents POST /api/auth/wecom/sso/callback.
// @Summary 统一认证中心扫码登录回调
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerWecomSSORequest true "认证中心回跳凭证"
// @Success 200 {object} authLoginResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 企业微信统一认证（SSO）模式：校验认证中心回跳的一次性 ticket 后按绑定关系登录并返回令牌与用户信息；需已启用统一认证模式
// @Router /api/auth/wecom/sso/callback [post]
func swaggerWecomSSOCallback() {}

// swaggerWecomSSOBind documents POST /api/auth/wecom/sso/bind.
// @Summary 统一认证中心扫码绑定
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerWecomSSORequest true "认证中心回跳凭证"
// @Success 200 {object} swaggerWecomBindResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Description 企业微信统一认证（SSO）模式：校验认证中心一次性 ticket 后与当前登录用户建立绑定；该企微已绑定其他用户时返回 409
// @Security BearerAuth
// @Router /api/auth/wecom/sso/bind [post]
func swaggerWecomSSOBind() {}

// swaggerMe documents GET /api/auth/me.
// @Summary 获取当前用户
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerGenericObject
// @Failure 401 {object} swaggerErrorResponse
// @Description 获取当前登录用户的资料、直接角色、有效角色与权限清单
// @Security BearerAuth
// @Router /api/auth/me [get]
func swaggerMe() {}

// swaggerLogout documents POST /api/auth/logout.
// @Summary 登出当前会话
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerOKResponse
// @Description 退出当前会话，并写入用户注销审计
// @Security BearerAuth
// @Router /api/auth/logout [post]
func swaggerLogout() {}

// swaggerHistory documents GET /api/history.
// @Summary 获取审计日志
// @Tags 审计日志
// @Produce json
// @Param limit query int false "返回条数；默认返回全部"
// @Success 200 {object} swaggerAuditHistoryResponse
// @Description 获取审计日志；导入类事件的目标仅返回前两个名称，完整清单通过 targetTitle 返回供悬浮提示
// @Security BearerAuth
// @Router /api/history [get]
func swaggerHistory() {}

// swaggerHistoryClear documents POST /api/history/clear.
// @Summary 清空审计日志
// @Tags 审计日志
// @Accept json
// @Produce json
// @Param body body swaggerHistoryClearRequest true "待清空的审计日志编号清单"
// @Success 200 {object} swaggerHistoryClearResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Description 按前端筛选结果批量清空审计日志；清空成功后自动补记一条「清空审计日志」审计事件（归属「审计日志」模块，目标为清除条数）
// @Security BearerAuth
// @Router /api/history/clear [post]
func swaggerHistoryClear() {}

// swaggerSettingsBaseGet documents GET /api/settings/base.
// @Summary 获取系统基础配置
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSystemBaseConfig
// @Failure 403 {object} swaggerErrorResponse
// @Description 获取系统基础配置
// @Security BearerAuth
// @Router /api/settings/base [get]
func swaggerSettingsBaseGet() {}

// swaggerSettingsBaseUpdate documents PUT /api/settings/base.
// @Summary 更新系统基础配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param body body swaggerSystemBaseConfig true "系统基础配置"
// @Success 200 {object} swaggerSystemBaseConfig
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Description 更新系统基础配置；携带 section（brand/security/backup）时按区块局部更新，审计目标细分为品牌标识、安全时效、数据备份
// @Security BearerAuth
// @Router /api/settings/base [put]
func swaggerSettingsBaseUpdate() {}

// swaggerPublicBaseGet documents GET /api/public/base.
// @Summary 获取公开品牌标识
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerPublicBrandingResponse
// @Description 获取登录页与启动屏使用的品牌标识，无需认证
// @Router /api/public/base [get]
func swaggerPublicBaseGet() {}

// swaggerSettingsUsersList documents GET /api/settings/users.
// @Summary 获取设置用户列表
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSettingsUserListResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 获取用户列表及有效角色与权限
// @Security BearerAuth
// @Router /api/settings/users [get]
func swaggerSettingsUsersList() {}

// swaggerSettingsRoles documents GET /api/settings/roles.
// @Summary 获取设置角色列表
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSettingsRoleListResponse
// @Description 获取角色列表及权限集合
// @Security BearerAuth
// @Router /api/settings/roles [get]
func swaggerSettingsRoles() {}

// swaggerSettingsGroups documents GET /api/settings/user-groups.
// @Summary 获取设置用户组列表
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSettingsGroupListResponse
// @Description 获取用户群组列表及成员与角色
// @Security BearerAuth
// @Router /api/settings/user-groups [get]
func swaggerSettingsGroups() {}

// swaggerSettingsAuthProvider documents GET /api/settings/auth-provider.
// @Summary 获取认证配置
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSettingsAuthProviderResponse
// @Description 获取 AD/LDAP 认证配置（敏感字段脱敏）
// @Security BearerAuth
// @Router /api/settings/auth-provider [get]
func swaggerSettingsAuthProvider() {}

// swaggerSettingsEmail documents GET /api/settings/email.
// @Summary 获取邮件配置
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerSettingsEmailResponse
// @Description 获取邮件配置（敏感字段脱敏）
// @Security BearerAuth
// @Router /api/settings/email [get]
func swaggerSettingsEmail() {}

// swaggerChangePassword documents POST /api/auth/change-password.
// @Summary 修改当前用户密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body changePasswordBody true "密码修改请求"
// @Success 200 {object} swaggerMessageResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Description 修改当前登录用户的密码，旧密码错误返回 400，成功后需重新登录
// @Security BearerAuth
// @Router /api/auth/change-password [post]
func swaggerChangePassword() {}

// swaggerPasswordResetCaptcha documents GET /api/auth/password-reset/captcha.
// @Summary 获取找回密码图形验证码
// @Tags 认证
// @Produce json
// @Success 200 {object} swaggerPasswordResetCaptchaResponse
// @Description 获取找回密码图形验证码，无需认证
// @Router /api/auth/password-reset/captcha [get]
func swaggerPasswordResetCaptcha() {}

// swaggerPasswordResetVerify documents POST /api/auth/password-reset/verify.
// @Summary 校验找回密码身份
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerPasswordResetVerifyRequest true "找回密码身份校验"
// @Success 200 {object} swaggerPasswordResetVerifyResponse
// @Description 校验用户名与图形验证码，换取找回密码流程令牌
// @Router /api/auth/password-reset/verify [post]
func swaggerPasswordResetVerify() {}

// swaggerPasswordResetSend documents POST /api/auth/password-reset/send.
// @Summary 发送找回密码验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerPasswordResetSendRequest true "找回密码验证码发送"
// @Success 200 {object} swaggerPasswordResetSendResponse
// @Description 向校验通过的邮箱发送找回密码验证码，受发送冷却与限流窗口约束
// @Router /api/auth/password-reset/send [post]
func swaggerPasswordResetSend() {}

// swaggerPasswordResetConfirm documents POST /api/auth/password-reset/confirm.
// @Summary 确认找回密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body swaggerPasswordResetConfirmRequest true "找回密码确认"
// @Success 200 {object} swaggerMessageResponse
// @Description 凭邮箱验证码完成找回密码，重置账号密码并写入审计
// @Router /api/auth/password-reset/confirm [post]
func swaggerPasswordResetConfirm() {}

// swaggerSettingsUsersPost documents POST /api/settings/users.
// @Summary 创建用户
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerSettingsUserRequest true "用户信息"
// @Success 201 {object} swaggerSettingsUser
// @Failure 400 {object} swaggerErrorResponse
// @Failure 409 {object} swaggerErrorResponse
// @Security BearerAuth
// @Router /api/settings/users [post]
func swaggerSettingsUsersPost() {}

// swaggerSettingsUsersPut documents PUT /api/settings/users/{id}.
// @Summary 更新用户
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "用户 ID"
// @Param request body swaggerSettingsUserRequest true "用户信息"
// @Success 200 {object} swaggerSettingsUser
// @Failure 400 {object} swaggerErrorResponse
// @Security BearerAuth
// @Router /api/settings/users/{id} [put]
func swaggerSettingsUsersPut() {}

// swaggerSettingsUsersDelete documents DELETE /api/settings/users/{id}.
// @Summary 删除用户
// @Tags 系统设置
// @Produce json
// @Param id path string true "用户 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 409 {object} swaggerErrorResponse
// @Security BearerAuth
// @Router /api/settings/users/{id} [delete]
func swaggerSettingsUsersDelete() {}

// swaggerSettingsUsersDisabled documents POST /api/settings/users/{id}/disabled.
// @Summary 启用或禁用用户
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "用户 ID"
// @Param request body swaggerSettingsDisabledRequest true "启用或禁用"
// @Success 200 {object} swaggerSettingsUser
// @Security BearerAuth
// @Router /api/settings/users/{id}/disabled [post]
func swaggerSettingsUsersDisabled() {}

// swaggerSettingsRolesPost documents POST /api/settings/roles.
// @Summary 创建角色
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerSettingsRoleRequest true "角色信息"
// @Success 201 {object} swaggerSettingsRole
// @Failure 409 {object} swaggerErrorResponse
// @Security BearerAuth
// @Router /api/settings/roles [post]
func swaggerSettingsRolesPost() {}

// swaggerSettingsRolesPut documents PUT /api/settings/roles/{id}.
// @Summary 更新角色
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "角色 ID"
// @Param request body swaggerSettingsRoleRequest true "角色信息"
// @Success 200 {object} swaggerSettingsRole
// @Security BearerAuth
// @Router /api/settings/roles/{id} [put]
func swaggerSettingsRolesPut() {}

// swaggerSettingsRolesDelete documents DELETE /api/settings/roles/{id}.
// @Summary 删除角色
// @Tags 系统设置
// @Produce json
// @Param id path string true "角色 ID"
// @Success 200 {object} swaggerOKResponse
// @Security BearerAuth
// @Router /api/settings/roles/{id} [delete]
func swaggerSettingsRolesDelete() {}

// swaggerSettingsRolesDisabled documents POST /api/settings/roles/{id}/disabled.
// @Summary 启用或禁用角色
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "角色 ID"
// @Param request body swaggerSettingsDisabledRequest true "启用或禁用"
// @Success 200 {object} swaggerSettingsRole
// @Security BearerAuth
// @Router /api/settings/roles/{id}/disabled [post]
func swaggerSettingsRolesDisabled() {}

// swaggerSettingsGroupsPost documents POST /api/settings/user-groups.
// @Summary 创建用户群组
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerSettingsGroupRequest true "群组信息"
// @Success 200 {object} swaggerSettingsGroup
// @Security BearerAuth
// @Router /api/settings/user-groups [post]
func swaggerSettingsGroupsPost() {}

// swaggerSettingsGroupsPut documents PUT /api/settings/user-groups/{id}.
// @Summary 更新用户群组
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param id path string true "群组 ID"
// @Param request body swaggerSettingsGroupRequest true "群组信息"
// @Success 200 {object} swaggerSettingsGroup
// @Security BearerAuth
// @Router /api/settings/user-groups/{id} [put]
func swaggerSettingsGroupsPut() {}

// swaggerSettingsGroupsDelete documents DELETE /api/settings/user-groups/{id}.
// @Summary 删除用户群组
// @Tags 系统设置
// @Produce json
// @Param id path string true "群组 ID"
// @Success 200 {object} swaggerOKResponse
// @Security BearerAuth
// @Router /api/settings/user-groups/{id} [delete]
func swaggerSettingsGroupsDelete() {}

// swaggerSettingsAuthProviderPut documents PUT /api/settings/auth-provider.
// @Summary 保存 AD/LDAP 认证配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerSettingsAuthProviderRequest true "AD/LDAP 认证配置"
// @Success 200 {object} swaggerSettingsAuthProviderResponse
// @Security BearerAuth
// @Router /api/settings/auth-provider [put]
func swaggerSettingsAuthProviderPut() {}

// swaggerSettingsWecomProvider documents GET /api/settings/auth/wecom.
// @Summary 获取企业微信认证配置
// @Tags 系统设置
// @Produce json
// @Success 200 {object} swaggerWecomProvider
// @Description 获取企业微信认证配置（Secret 不回显，仅返回是否已配置）
// @Security BearerAuth
// @Router /api/settings/auth/wecom [get]
func swaggerSettingsWecomProvider() {}

// swaggerSettingsWecomProviderPut documents PUT /api/settings/auth/wecom.
// @Summary 保存企业微信认证配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param body body swaggerWecomProviderRequest true "企业微信认证配置"
// @Success 200 {object} swaggerWecomProvider
// @Failure 400 {object} swaggerErrorResponse
// @Description 保存企业微信认证配置；认证方式 direct（直连）启用时要求企业 ID、AgentID 与 Secret 完整，sso（统一认证中心）启用时要求认证中心地址、应用标识与应用密钥完整，密钥加密存储不回显
// @Security BearerAuth
// @Router /api/settings/auth/wecom [put]
func swaggerSettingsWecomProviderPut() {}

// swaggerSettingsEmailPut documents PUT /api/settings/email.
// @Summary 保存邮件配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body swaggerSettingsEmailRequest true "邮件配置"
// @Success 200 {object} swaggerSettingsEmailResponse
// @Security BearerAuth
// @Router /api/settings/email [put]
func swaggerSettingsEmailPut() {}
