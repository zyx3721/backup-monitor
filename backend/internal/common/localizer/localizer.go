package localizer

import (
	"strings"
	"unicode"
)

// exactMessageCN 精确映射保留功能（认证、用户、设置、审计）实际产生的英文错误消息
var exactMessageCN = map[string]string{
	"invalid json":                     "请求体 JSON 格式无效",
	"username is required":             "用户名不能为空",
	"invalid username or password":     "用户名或密码错误",
	"invalid current password":         "当前密码错误",
	"unauthenticated":                  "未登录或登录已失效",
	"missing authorization header":     "缺少 Authorization 请求头",
	"invalid authorization header":     "Authorization 请求头格式无效",
	"invalid token":                    "登录令牌无效或已过期",
	"session expired or revoked":       "登录会话已过期或已被注销，请重新登录",
	"invalid id":                       "ID 参数无效",
	"user not found":                   "用户不存在",
	"username already exists":          "用户名已存在",
	"password is required":             "密码不能为空",
	"cannot remove default admin user": "不能删除默认管理员账号",
	"invalid user context":             "用户上下文无效",
	"missing user":                     "缺少用户上下文",
	"database bootstrap sql is empty":  "数据库初始化 SQL 为空",
}

func localizeMessage(message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "未知错误"
	}

	lower := strings.ToLower(msg)
	if translated, ok := exactMessageCN[lower]; ok {
		return translated
	}

	if strings.HasPrefix(lower, "hash default admin password failed:") {
		return "初始化默认管理员密码失败：" + strings.TrimSpace(msg[len("hash default admin password failed:"):])
	}
	if strings.HasPrefix(lower, "insert default admin failed:") {
		return "写入默认管理员账号失败：" + strings.TrimSpace(msg[len("insert default admin failed:"):])
	}
	if strings.HasPrefix(lower, "execute bootstrap sql #") {
		return "执行数据库初始化语句失败：" + msg
	}
	if strings.Contains(lower, "only one usage of each socket address") || strings.Contains(lower, "address already in use") {
		return "端口已被占用，请更换端口或关闭占用进程"
	}
	if strings.Contains(lower, "no such file or directory") {
		return "文件或目录不存在"
	}
	if strings.Contains(lower, "permission denied") {
		return "权限不足"
	}
	if strings.Contains(lower, "database is locked") {
		return "数据库被锁定，请稍后重试"
	}

	if strings.HasPrefix(lower, "invalid ") {
		return "参数无效：" + strings.TrimSpace(msg[len("invalid "):])
	}
	if strings.HasPrefix(lower, "missing ") {
		return "缺少必要参数：" + strings.TrimSpace(msg[len("missing "):])
	}
	if strings.HasSuffix(lower, " is required") {
		return strings.TrimSpace(msg[:len(msg)-len(" is required")]) + " 不能为空"
	}
	if strings.HasPrefix(lower, "cannot ") {
		return "操作不允许：" + msg
	}
	if strings.HasPrefix(lower, "unsupported ") {
		return "不支持的操作：" + strings.TrimSpace(msg[len("unsupported "):])
	}

	replaced := msg
	replaced = strings.ReplaceAll(replaced, "UNIQUE constraint failed", "唯一约束冲突")
	replaced = strings.ReplaceAll(replaced, "FOREIGN KEY constraint failed", "外键约束冲突")
	replaced = strings.ReplaceAll(replaced, "NOT NULL constraint failed", "非空约束冲突")
	replaced = strings.ReplaceAll(replaced, "CHECK constraint failed", "检查约束冲突")
	replaced = strings.ReplaceAll(replaced, "SQLITE_BUSY", "数据库忙")
	replaced = strings.ReplaceAll(replaced, "SQLITE_CONSTRAINT", "数据库约束冲突")
	replaced = strings.ReplaceAll(replaced, "SQL logic error", "SQL 逻辑错误")
	replaced = strings.ReplaceAll(replaced, "syntax error", "语法错误")
	replaced = strings.ReplaceAll(replaced, "no such table", "不存在的数据表")
	replaced = strings.ReplaceAll(replaced, "no such column", "不存在的字段")
	replaced = strings.ReplaceAll(replaced, "constraint failed", "约束冲突")
	replaced = strings.ReplaceAll(replaced, "database schema has changed", "数据库结构已变更")
	replaced = strings.ReplaceAll(replaced, "cannot start a transaction within a transaction", "事务开启失败：当前已存在事务")
	replaced = strings.ReplaceAll(replaced, "foreign key mismatch", "外键不匹配")
	replaced = strings.ReplaceAll(replaced, "datatype mismatch", "数据类型不匹配")
	replaced = strings.ReplaceAll(replaced, "busy timeout", "数据库忙超时")

	if containsChinese(replaced) {
		return replaced
	}

	if containsLatin(replaced) {
		return "系统错误，请稍后重试"
	}

	return replaced
}

func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func containsLatin(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && r <= unicode.MaxASCII {
			return true
		}
	}
	return false
}

func LocalizeMessage(message string) string {
	return localizeMessage(message)
}
