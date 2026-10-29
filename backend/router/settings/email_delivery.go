package settings

import (
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/security"
)

func SendSettingsTestEmail(config map[string]any, to string) error {
	host := settingsString(config, "smtpHost")
	port := settingsString(config, "smtpPort")
	if host == "" {
		return fmt.Errorf("SMTP 主机不能为空")
	}
	if port == "" {
		port = "25"
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("测试收件人不能为空")
	}
	from := settingsString(config, "from")
	if from == "" {
		return fmt.Errorf("发件人不能为空")
	}
	if err := validateSMTPAddress("发件人", from); err != nil {
		return err
	}
	if err := validateSMTPAddress("收件人", to); err != nil {
		return err
	}
	fromAddress, _ := mail.ParseAddress(from)
	toAddress, _ := mail.ParseAddress(to)
	fromName := settingsString(config, "fromName")
	if fromName == "" {
		fromName = "Backup Monitor"
	}
	password := settingsString(config, "password")
	if strings.HasPrefix(password, "enc:v1:") {
		decrypted, err := security.DecryptSettingsSecret(password, "")
		if err != nil {
			return fmt.Errorf("SMTP 密码解密失败")
		}
		password = decrypted
	}
	username := settingsString(config, "username")
	useTLS := port == "465" || settingsBool(config, "useTLS")
	startTLS := !useTLS && settingsBool(config, "startTLS")
	if settingsBool(config, "useTLS") && settingsBool(config, "startTLS") {
		return fmt.Errorf("TLS 与 STARTTLS 不能同时启用")
	}
	if username != "" && !useTLS && !startTLS && !settingsBool(config, "allowInsecureAuth") {
		return fmt.Errorf("SMTP 明文认证未启用")
	}
	address := net.JoinHostPort(host, port)
	var conn net.Conn
	var err error
	if useTLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 8 * time.Second}, "tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: settingsBool(config, "insecureSkipVerify")})
	} else {
		conn, err = (&net.Dialer{Timeout: 8 * time.Second}).Dial("tcp", address)
	}
	if err != nil {
		return fmt.Errorf("SMTP 连接失败")
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SMTP 握手失败")
	}
	defer client.Close()
	if startTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP 服务不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: settingsBool(config, "insecureSkipVerify")}); err != nil {
			return fmt.Errorf("SMTP STARTTLS 失败")
		}
	}
	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if !useTLS && !startTLS && settingsBool(config, "allowInsecureAuth") {
			auth = appcommon.InsecurePlainAuth(username, password)
		}
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败：%v", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP 发件人校验失败")
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP 收件人校验失败")
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP 数据通道失败")
	}
	subject := "Backup Monitor 邮件配置测试"
	var body []byte
	if code := settingsString(config, "resetCode"); code != "" {
		brand := settingsString(config, "resetBrand")
		if brand == "" {
			brand = "Backup Monitor"
		}
		subject = brand + " 密码找回验证码"
		body = buildSettingsEmailMessage(fromAddress, toAddress, fromName, subject, passwordResetEmailHTML(brand, settingsString(config, "resetUsername"), code, settingsString(config, "resetExpiresAt"), settingsString(config, "resetRequestIP")), "text/html")
	} else {
		body = buildSettingsEmailMessage(fromAddress, toAddress, fromName, subject, "这是一封 Backup Monitor 邮件配置测试邮件", "text/plain")
	}
	if _, err = writer.Write([]byte(body)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP 邮件内容写入失败")
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("SMTP 邮件发送失败")
	}
	return client.Quit()
}

func settingsString(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}
func settingsBool(config map[string]any, key string) bool {
	value, ok := config[key].(bool)
	if ok {
		return value
	}
	text := settingsString(config, key)
	parsed, _ := strconv.ParseBool(text)
	return parsed
}

// validateSMTPAddress 对发件人/收件人地址做防注入与格式校验；
// field 为“发件人”“收件人”等字段名，错误信息具体化以便定位是哪个地址、为何无效
func validateSMTPAddress(field, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s地址不能包含换行符", field)
	}
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "<") {
		return fmt.Errorf("%s地址格式不正确", field)
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("%s地址格式不正确", field)
	}
	atPos := strings.LastIndex(parsed.Address, "@")
	if atPos <= 0 || atPos == len(parsed.Address)-1 {
		return fmt.Errorf("%s地址缺少有效的用户名或域名部分", field)
	}
	return nil
}

func buildSettingsEmailMessage(from, to *mail.Address, fromName, subject, bodyText, contentType string) []byte {
	if strings.TrimSpace(fromName) == "" {
		fromName = "Backup Monitor"
	}
	fromHeader := (&mail.Address{Name: fromName, Address: from.Address}).String()
	return []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s\r\n", fromHeader, to.String(), subject, contentType, bodyText))
}

// passwordResetEmailHTML 找回密码邮件 HTML 模板，配色采用默认品牌色
func passwordResetEmailHTML(brand, username, code, expiresAt, requestIP string) string {
	username = html.EscapeString(strings.TrimSpace(username))
	if username == "" {
		username = "当前账号"
	}
	requestIP = html.EscapeString(strings.TrimSpace(requestIP))
	if requestIP == "" {
		requestIP = "未知"
	}
	brand = html.EscapeString(strings.TrimSpace(brand))
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#f5f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;color:#172033;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f5f7fb;padding:32px 12px;"><tr><td align="center">
<table role="presentation" width="560" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #e4e9f2;border-radius:14px;overflow:hidden;">
<tr><td style="padding:28px 32px 18px;background:#2563eb;color:#ffffff;"><div style="font-size:20px;font-weight:700;">%s 密码找回</div><div style="margin-top:8px;font-size:13px;opacity:.86;">请使用以下验证码完成密码重置</div></td></tr>
<tr><td style="padding:30px 32px;"><div style="font-size:14px;color:#526071;">账号</div><div style="margin-top:6px;font-size:18px;font-weight:700;color:#172033;">%s</div><div style="margin-top:24px;padding:18px 20px;border-radius:12px;background:#eff6ff;border:1px solid #bfdbfe;text-align:center;"><div style="font-size:13px;color:#1d4ed8;">验证码</div><div style="margin-top:8px;font-size:34px;letter-spacing:8px;font-weight:800;color:#1e40af;">%s</div></div><div style="margin-top:22px;font-size:14px;line-height:1.8;color:#526071;">有效期至：<strong style="color:#172033;">%s</strong><br>请求来源：<strong style="color:#172033;">%s</strong></div><div style="margin-top:24px;padding:14px 16px;border-radius:10px;background:#fff7ed;border:1px solid #fed7aa;color:#9a3412;font-size:13px;line-height:1.7;">如果不是您本人操作，请忽略本邮件并检查平台账号安全。</div></td></tr>
</table></td></tr></table></body></html>`, brand, username, code, expiresAt, requestIP)
}
