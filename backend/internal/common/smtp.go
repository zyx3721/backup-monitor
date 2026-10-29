package common

import "net/smtp"

// insecurePlainAuth 允许在未加密连接上完成 PLAIN 认证，仅在显式开启明文认证时使用。
type insecurePlainAuth string

// InsecurePlainAuth 构造明文连接可用的 PLAIN 认证，绕过标准库 smtp.PlainAuth 的加密校验。
func InsecurePlainAuth(username string, password string) smtp.Auth {
	return insecurePlainAuth("\x00" + username + "\x00" + password)
}

func (a insecurePlainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte(a), nil
}

func (a insecurePlainAuth) Next([]byte, bool) ([]byte, error) {
	return nil, nil
}
