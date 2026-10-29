package swagger

import "backup-monitor-backend/internal/domain"

// swag 注解引用的载荷类型别名，实际定义见 internal/domain
type authLoginRequest = domain.AuthLoginRequest
type authLoginResponse = domain.AuthLoginResponse
type changePasswordBody = domain.ChangePasswordBody
