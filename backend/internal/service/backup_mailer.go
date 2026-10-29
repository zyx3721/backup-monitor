package service

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/domain"
	"backup-monitor-backend/internal/security"
)

// sendMail 使用「邮件设置」中的 SMTP 配置发送通知邮件。
func (s *BackupService) sendMail(recipients []string, subject, body string) error {
	var raw string
	if err := s.repo.DB().QueryRow(`SELECT config FROM pwd_recover_config WHERE name='email'`).Scan(&raw); err != nil {
		return fmt.Errorf("读取 SMTP 配置失败")
	}
	config := domain.ParseJSONObject(raw)
	host := textConfig(config, "smtpHost")
	if host == "" {
		return fmt.Errorf("SMTP 主机未配置")
	}
	port := firstNonEmpty(textConfig(config, "smtpPort"), "25")
	from := textConfig(config, "from")
	if from == "" {
		from = textConfig(config, "username")
	}
	if from == "" {
		return fmt.Errorf("发件人未配置")
	}
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("发件人地址无效")
	}
	validRecipients := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		address, err := mail.ParseAddress(strings.TrimSpace(recipient))
		if err != nil {
			return fmt.Errorf("收件人地址无效：%s", recipient)
		}
		validRecipients = append(validRecipients, address.Address)
	}
	if len(validRecipients) == 0 {
		return fmt.Errorf("收件人为空")
	}

	password := textConfig(config, "password")
	if strings.HasPrefix(password, "enc:v1:") {
		decrypted, err := security.DecryptSettingsSecret(password, "")
		if err != nil {
			return fmt.Errorf("SMTP 密码解密失败")
		}
		password = decrypted
	}
	username := textConfig(config, "username")
	fromName := firstNonEmpty(textConfig(config, "fromName"), "Backup Monitor")
	useTLS := port == "465" || boolConfig(config, "useTLS")
	startTLS := !useTLS && boolConfig(config, "startTLS")
	address := net.JoinHostPort(host, port)

	var conn net.Conn
	if useTLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", address,
			&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: boolConfig(config, "insecureSkipVerify")})
	} else {
		conn, err = (&net.Dialer{Timeout: 10 * time.Second}).Dial("tcp", address)
	}
	if err != nil {
		return fmt.Errorf("SMTP 连接失败：%v", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SMTP 握手失败：%v", err)
	}
	defer client.Close()
	if startTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP 服务不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, InsecureSkipVerify: boolConfig(config, "insecureSkipVerify")}); err != nil {
			return fmt.Errorf("STARTTLS 失败：%v", err)
		}
	}
	if username != "" && password != "" {
		plainConnection := !useTLS && !startTLS
		insecureAuth := plainConnection && boolConfig(config, "allowInsecureAuth")
		if plainConnection && !insecureAuth {
			return fmt.Errorf("SMTP 明文认证未启用")
		}
		if ok, _ := client.Extension("AUTH"); !ok && !insecureAuth {
			return fmt.Errorf("SMTP 服务不支持认证")
		}
		auth := smtp.PlainAuth("", username, password, host)
		if insecureAuth {
			auth = appcommon.InsecurePlainAuth(username, password)
		}
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败：%v", err)
		}
	}
	if err := client.Mail(fromAddress.Address); err != nil {
		return fmt.Errorf("SMTP 发件人校验失败：%v", err)
	}
	for _, recipient := range validRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP 收件人校验失败：%v", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP 数据通道失败：%v", err)
	}
	message := buildMailMessage(fromName, fromAddress.Address, validRecipients, subject, body)
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP 邮件内容写入失败：%v", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP 邮件发送失败：%v", err)
	}
	_ = client.Quit()
	return nil
}

func buildMailMessage(fromName, from string, recipients []string, subject, body string) string {
	headers := []string{
		"From: " + mime.QEncoding.Encode("UTF-8", fromName) + " <" + from + ">",
		"To: " + strings.Join(recipients, ", "),
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: base64",
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	lines := make([]string, 0, len(encoded)/76+1)
	for len(encoded) > 76 {
		lines = append(lines, encoded[:76])
		encoded = encoded[76:]
	}
	lines = append(lines, encoded)
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + strings.Join(lines, "\r\n") + "\r\n"
}

func textConfig(config map[string]any, key string) string {
	if value, ok := config[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func boolConfig(config map[string]any, key string) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return false
}
