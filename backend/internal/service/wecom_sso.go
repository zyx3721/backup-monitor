package service

// 企业微信统一认证中心对接：跳转 /login 扫码，再以 HMAC-SHA256 签名调用 /api/verify 换取成员 userid

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// WecomSSOLoginURL 构造统一认证中心扫码登录入口地址，认证中心自行登记 state 与回跳地址
func (p *WecomProvider) WecomSSOLoginURL() string {
	return strings.TrimRight(p.SSOBaseURL, "/") + "/login?app=" + url.QueryEscape(p.SSOAppID)
}

// WecomSSOVerifyTicket 携带签名调用认证中心 /api/verify，用一次性 ticket 换取企业微信成员 userid
func (p *WecomProvider) WecomSSOVerifyTicket(ctx context.Context, ticket string) (string, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return "", errors.New("登录凭证缺失，请重新发起登录")
	}
	ts := wecomNowFunc().Unix()
	mac := hmac.New(sha256.New, []byte(p.SSOAppSecret))
	fmt.Fprintf(mac, "%s\n%s\n%d", p.SSOAppID, ticket, ts)
	body, err := json.Marshal(struct {
		App    string `json:"app"`
		Ticket string `json:"ticket"`
		TS     int64  `json:"ts"`
		Sign   string `json:"sign"`
	}{App: p.SSOAppID, Ticket: ticket, TS: ts, Sign: hex.EncodeToString(mac.Sum(nil))})
	if err != nil {
		return "", fmt.Errorf("构造统一认证中心请求失败: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.SSOBaseURL, "/")+"/api/verify", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造统一认证中心请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := wecomHTTPClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("连接统一认证中心失败: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("读取统一认证中心响应失败: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &failure)
		return "", errors.New(wecomSSOErrorMessage(failure.Error))
	}
	var result struct {
		UserID string `json:"userid"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", errors.New("解析统一认证中心响应失败")
	}
	if strings.TrimSpace(result.UserID) == "" {
		return "", errors.New("统一认证中心未返回成员身份，请确认账号为企业成员")
	}
	return result.UserID, nil
}

// wecomSSOErrorMessage 将认证中心 verify 错误码转译为用户可读提示
func wecomSSOErrorMessage(code string) string {
	switch code {
	case "invalid_app":
		return "统一认证中心未注册本系统的应用标识，请检查应用标识配置"
	case "invalid_sign":
		return "统一认证中心签名校验失败，请检查应用密钥配置"
	case "expired_ts":
		return "与统一认证中心服务器时间偏差过大，请校准系统时钟后重试"
	case "invalid_ticket":
		return "登录凭证无效或已过期，请重新发起登录"
	default:
		return "统一认证中心登录校验失败"
	}
}
