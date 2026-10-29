package settings

import (
	"backup-monitor-backend/internal/security"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// ldapUserMessage 将 LDAP 连接与查询错误映射为用户可读的中文提示
func ldapUserMessage(err error) string {
	if err == nil {
		return ""
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return "LDAP TLS 证书不受信任，请导入可信证书或勾选跳过证书校验"
	}
	var hostError x509.HostnameError
	if errors.As(err, &hostError) {
		return "LDAP TLS 证书域名与服务器地址不匹配，请检查证书或勾选跳过证书校验"
	}
	var certInvalidError x509.CertificateInvalidError
	if errors.As(err, &certInvalidError) {
		return "LDAP TLS 证书无效或已过期，请检查证书配置"
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		return "LDAP 服务连接超时，请检查服务器地址、端口和网络连通性"
	}
	if errors.Is(err, io.EOF) {
		return "LDAP 服务提前断开连接，请确认端口协议是否匹配"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "connection refused") {
		return "LDAP 服务拒绝连接，请检查端口是否开放"
	}
	if strings.Contains(message, "connection reset") {
		return "LDAP 连接被重置，请确认端口协议和 TLS 配置是否匹配"
	}
	if strings.Contains(message, "first record does not look like a tls handshake") {
		return "当前端口不是 LDAPS 服务，请改用 389 或关闭 LDAPS"
	}
	if strings.Contains(message, "unsupported protocol version") || strings.Contains(message, "protocol version 301") {
		return "LDAP TLS 版本过低，请启用 TLS 1.2+"
	}
	if strings.Contains(message, "start tls") || strings.Contains(message, "starttls") {
		return "LDAP 服务不支持 StartTLS 或 StartTLS 握手失败，请检查服务端配置"
	}
	if strings.Contains(message, "invalid credentials") {
		return "LDAP 绑定账号或密码不正确"
	}
	var filterErr ldapFilterSearchError
	if errors.As(err, &filterErr) && strings.Contains(message, "filter compile error") {
		return filterErr.Source + "格式不正确，请填写完整 LDAP 过滤器"
	}
	if strings.Contains(message, "filter compile error") {
		return "LDAP 过滤器格式不正确，请检查用户过滤器或用户组过滤器"
	}
	return "认证服务连接测试失败：" + err.Error()
}

const defaultLDAPLoginAttr = "sAMAccountName"

type ldapSettings struct {
	Enabled            int64
	Server             string
	BaseDN             string
	BindDN             string
	BindPassword       string
	UserFilter         string
	GroupFilter        string
	Port               int64
	UseTLS             bool
	StartTLS           bool
	InsecureSkipVerify bool
	TimeoutSeconds     int64
}

// loadLDAPSettings 从 auth_config 读取并解密 LDAP 认证配置
func loadLDAPSettings(db *sql.DB, jwtSecret string) (ldapSettings, error) {
	var cfg ldapSettings
	var enabled int64
	var raw string
	if err := db.QueryRow(`SELECT enabled, COALESCE(config,'{}') FROM auth_config WHERE name='ldap'`).Scan(&enabled, &raw); err != nil {
		return ldapSettings{}, err
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		stored = map[string]any{}
	}
	options := ldapProviderOptionsFromConfig(raw)
	cfg = ldapSettings{
		Enabled:            enabled,
		Server:             stringConfig(stored, "host"),
		BaseDN:             stringConfig(stored, "baseDN"),
		BindDN:             stringConfig(stored, "bindDN"),
		UserFilter:         stringConfig(stored, "userFilter"),
		GroupFilter:        stringConfig(stored, "groupFilter"),
		Port:               options.Port,
		UseTLS:             options.UseTLS,
		StartTLS:           options.StartTLS,
		InsecureSkipVerify: options.InsecureSkipVerify,
		TimeoutSeconds:     options.TimeoutSeconds,
	}
	password, err := security.DecryptSettingsSecret(stringConfig(stored, "bindPassword"), jwtSecret)
	if err != nil {
		return ldapSettings{}, err
	}
	cfg.BindPassword = password
	return cfg, nil
}

// ldapProviderOptions 从认证配置 JSON 解析的连接选项，零值表示沿用默认行为
type ldapProviderOptions struct {
	Port               int64
	UseTLS             bool
	StartTLS           bool
	InsecureSkipVerify bool
	TimeoutSeconds     int64
}

// ldapProviderOptionsFromConfig 从认证配置 JSON 读取连接选项，缺失或非法时为零值
func ldapProviderOptionsFromConfig(raw string) ldapProviderOptions {
	var options ldapProviderOptions
	if strings.TrimSpace(raw) == "" {
		return options
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return options
	}
	options.Port = parseLDAPNumber(config["port"])
	options.TimeoutSeconds = parseLDAPNumber(config["timeoutSeconds"])
	options.UseTLS = parseLDAPBool(config["useTLS"])
	options.StartTLS = parseLDAPBool(config["startTLS"])
	options.InsecureSkipVerify = parseLDAPBool(config["insecureSkipVerify"])
	return options
}

func parseLDAPNumber(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed
	}
	return 0
}

func parseLDAPBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	}
	return false
}

func AuthenticateLDAPUser(db *sql.DB, jwtSecret, username, password string) error {
	cfg, err := loadLDAPSettings(db, jwtSecret)
	if err != nil {
		return err
	}
	if cfg.Enabled != 1 {
		return errors.New("LDAP 未启用")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}

	conn, err := dialAndBindLDAP(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	userDN, err := searchLDAPUserDN(conn, cfg, username)
	if err != nil {
		return err
	}

	if err := conn.Bind(userDN, password); err != nil {
		if IsLDAPInvalidCredentialsError(err) {
			return errors.New("invalid username or password")
		}
		return err
	}
	return nil
}

// dialAndBindLDAP 按配置的端口与 TLS 选项建立连接并绑定
func dialAndBindLDAP(cfg ldapSettings) (*ldap.Conn, error) {
	host := strings.TrimSpace(cfg.Server)
	if host == "" {
		return nil, errors.New("ldap server is required")
	}
	if strings.TrimSpace(cfg.BaseDN) == "" {
		return nil, errors.New("ldap base DN is required")
	}
	useTLS := cfg.UseTLS
	if len(host) > 8 && strings.EqualFold(host[:8], "ldaps://") {
		host = strings.TrimSpace(host[8:])
		useTLS = true
	} else if len(host) > 7 && strings.EqualFold(host[:7], "ldap://") {
		host = strings.TrimSpace(host[7:])
	}
	port := cfg.Port
	address := net.JoinHostPort(host, strconv.FormatInt(port, 10))
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	tlsConfig := &tls.Config{ServerName: host, InsecureSkipVerify: cfg.InsecureSkipVerify}
	var conn *ldap.Conn
	var err error
	if useTLS {
		dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: timeout}, Config: tlsConfig}
		rawConn, dialErr := dialer.DialContext(context.Background(), "tcp", address)
		if dialErr != nil {
			return nil, dialErr
		}
		conn = ldap.NewConn(rawConn, true)
		conn.Start()
	} else {
		dialer := &net.Dialer{Timeout: timeout}
		rawConn, dialErr := dialer.DialContext(context.Background(), "tcp", address)
		if dialErr != nil {
			return nil, dialErr
		}
		conn = ldap.NewConn(rawConn, false)
		conn.Start()
		if cfg.StartTLS {
			err = conn.StartTLS(tlsConfig)
		}
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.SetTimeout(timeout)
	bindDN := strings.TrimSpace(cfg.BindDN)
	bindPassword := cfg.BindPassword
	if bindDN == "" {
		if strings.TrimSpace(bindPassword) != "" {
			conn.Close()
			return nil, errors.New("ldap bind DN is required")
		}
		return conn, nil
	}
	if err := conn.Bind(bindDN, bindPassword); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func searchLDAPUserDN(conn *ldap.Conn, cfg ldapSettings, username string) (string, error) {
	searchBase := strings.TrimSpace(cfg.BaseDN)
	if searchBase == "" {
		return "", errors.New("ldap base DN is required")
	}

	filter, err := buildLDAPUserFilter(cfg.UserFilter, cfg.GroupFilter, username)
	if err != nil {
		return "", err
	}

	req := ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		2,
		5,
		false,
		filter,
		[]string{"dn"},
		nil,
	)

	result, err := conn.Search(req)
	if err != nil {
		return "", err
	}
	if len(result.Entries) == 0 {
		return "", errors.New("invalid username or password")
	}
	if len(result.Entries) > 1 {
		return "", errors.New("LDAP 用户查询结果不唯一")
	}
	return result.Entries[0].DN, nil
}

func buildLDAPUserFilter(queryTemplate, extraFilter, username string) (string, error) {
	escapedUser := ldap.EscapeFilter(strings.TrimSpace(username))
	loginAttr := defaultLDAPLoginAttr
	primaryTemplate := strings.TrimSpace(queryTemplate)

	switch {
	case primaryTemplate == "":
		primaryTemplate = fmt.Sprintf("(%s=%%{user})", loginAttr)
	case looksLikeLDAPAttributeName(primaryTemplate):
		loginAttr = primaryTemplate
		primaryTemplate = fmt.Sprintf("(%s=%%{user})", loginAttr)
	}

	primary, err := expandLDAPFilter(primaryTemplate, loginAttr, escapedUser)
	if err != nil {
		return "", err
	}

	extra := strings.TrimSpace(extraFilter)
	if extra == "" {
		return primary, nil
	}
	extraExpanded, err := expandLDAPFilter(ldapNormalizedGroupFilter(extra), loginAttr, escapedUser)
	if err != nil {
		return "", err
	}
	return "(&" + primary + extraExpanded + ")", nil
}

func expandLDAPFilter(raw, loginAttr, escapedUser string) (string, error) {
	filter := strings.TrimSpace(raw)
	if filter == "" {
		return "", errors.New("ldap filter is required")
	}

	attrName := sanitizeLDAPAttributeName(loginAttr)
	if attrName == "" {
		attrName = defaultLDAPLoginAttr
	}

	filter = strings.ReplaceAll(filter, "%{attr}", attrName)
	filter = strings.ReplaceAll(filter, "%{user}", escapedUser)
	filter = strings.ReplaceAll(filter, "{username}", escapedUser)
	if !strings.HasPrefix(filter, "(") {
		filter = "(" + filter + ")"
	}
	return filter, nil
}

func looksLikeLDAPAttributeName(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "()=%{}&|!<>~ ") {
		return false
	}
	return sanitizeLDAPAttributeName(value) == value
}

func sanitizeLDAPAttributeName(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func IsLDAPInvalidCredentialsError(err error) bool {
	if err == nil {
		return false
	}

	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) && ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid username or password") ||
		strings.Contains(message, "invalid credentials") ||
		strings.Contains(message, "ldap result code 49")
}

// ldapFilterSearchError 标识搜索阶段由过滤器引发的错误，Source 为过滤器名称（用户过滤器/用户组过滤器）
type ldapFilterSearchError struct {
	Source string
	Err    error
}

func (e ldapFilterSearchError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "ldap filter error"
}

func (e ldapFilterSearchError) Unwrap() error {
	return e.Err
}

// ldapTestUserFilter 测试连接使用的过滤器：配置了用户组过滤器时按组统计
func ldapTestUserFilter(cfg ldapSettings) (string, string) {
	if groupFilter := strings.TrimSpace(cfg.GroupFilter); groupFilter != "" {
		return ldapNormalizedGroupFilter(groupFilter), "用户组过滤器"
	}
	filter := strings.TrimSpace(cfg.UserFilter)
	filter = strings.ReplaceAll(filter, "%{user}", "*")
	filter = strings.ReplaceAll(filter, "{username}", "*")
	return filter, "用户过滤器"
}

func ldapNormalizedGroupFilter(groupFilter string) string {
	groupFilter = strings.TrimSpace(groupFilter)
	if groupFilter == "" {
		return ""
	}
	if strings.HasPrefix(groupFilter, "(") {
		return groupFilter
	}
	return "(memberOf=" + ldap.EscapeFilter(groupFilter) + ")"
}

func countLDAPUsers(conn *ldap.Conn, cfg ldapSettings) (int, error) {
	base := strings.TrimSpace(cfg.BaseDN)
	if base == "" {
		return 0, errors.New("ldap base DN is required")
	}
	filter, source := ldapTestUserFilter(cfg)
	request := ldap.NewSearchRequest(base, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, int(cfg.TimeoutSeconds), false, filter, []string{"dn"}, nil)
	result, err := conn.SearchWithPaging(request, 500)
	if err != nil {
		return 0, ldapFilterSearchError{Source: source, Err: err}
	}
	return len(result.Entries), nil
}
