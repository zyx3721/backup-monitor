package auth

import (
	appcommon "backup-monitor-backend/internal/common"
	"backup-monitor-backend/internal/service"
	"backup-monitor-backend/router/common"
	"backup-monitor-backend/router/settings"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backup-monitor-backend/internal/security"
)

// defaultResetRateLimitMax 统计窗口内同一邮箱最多发送次数。
const defaultResetRateLimitMax = 5

// resetRateLimitedError 频率限制错误，Minutes 为按统计窗口推算的剩余等待分钟数
type resetRateLimitedError struct {
	Minutes int64
}

func (e resetRateLimitedError) Error() string {
	return fmt.Sprintf("验证码请求过于频繁，请于 %d 分钟后再试", e.Minutes)
}

type passwordResetVerifyBody struct {
	Username      string `json:"username"`
	CaptchaToken  string `json:"captchaToken"`
	CaptchaAnswer string `json:"captchaAnswer"`
}
type passwordResetSendBody struct {
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	Channel           string `json:"channel"`
	VerifyEmail       string `json:"verifyEmail"`
}
type passwordResetConfirmBody struct {
	Username          string `json:"username"`
	VerificationToken string `json:"verificationToken"`
	Code              string `json:"code"`
	NewPassword       string `json:"newPassword"`
	ConfirmPassword   string `json:"confirmPassword"`
}

func (a *Router) handlePasswordResetCaptcha(w http.ResponseWriter, r *http.Request) {
	var first, second int
	if err := readRandomInt(&first, 1, 9); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "生成验证码失败")
		return
	}
	if err := readRandomInt(&second, 1, 9); err != nil {
		common.WriteError(w, http.StatusInternalServerError, "生成验证码失败")
		return
	}
	token, err := randomToken()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "生成验证码失败")
		return
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(settings.LoadSystemBaseConfig(r.Context(), a.db).ResetCaptchaTTLMinutes) * time.Minute)
	_, _ = a.db.ExecContext(r.Context(), "DELETE FROM reset_captchas WHERE expires_at < ?", appcommon.FormatTimestamp(now))
	_, err = a.db.ExecContext(r.Context(), "INSERT INTO reset_captchas(token_hash,question,answer_hash,expires_at,created_at) VALUES(?,?,?,?,?)", hashToken(token), fmt.Sprintf("%d + %d = ?", first, second), hashToken(strconv.Itoa(first+second)), appcommon.FormatTimestamp(expiresAt), appcommon.FormatTimestamp(now))
	if err != nil {
		common.WriteError(w, 500, "生成验证码失败")
		return
	}
	common.WriteJSON(w, 200, map[string]string{"token": token, "question": fmt.Sprintf("%d + %d = ?", first, second), "expiresAt": appcommon.FormatTimestamp(expiresAt)})
}

func (a *Router) handlePasswordResetVerify(w http.ResponseWriter, r *http.Request) {
	var body passwordResetVerifyBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	var answerHash string
	var expires string
	if err := a.db.QueryRowContext(r.Context(), "SELECT answer_hash,expires_at FROM reset_captchas WHERE token_hash=?", hashToken(body.CaptchaToken)).Scan(&answerHash, &expires); err != nil || expires < appcommon.NowTimestamp() || answerHash != hashToken(strings.TrimSpace(body.CaptchaAnswer)) {
		common.WriteError(w, 400, "验证码不正确")
		return
	}
	_, _ = a.db.ExecContext(r.Context(), "DELETE FROM reset_captchas WHERE token_hash=?", hashToken(body.CaptchaToken))
	var id int64
	var email string
	var disabled int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT id,COALESCE(email,''),disabled FROM users WHERE LOWER(username)=LOWER(?)", strings.TrimSpace(body.Username)).Scan(&id, &email, &disabled); err != nil || disabled != 0 || strings.TrimSpace(email) == "" {
		common.WriteError(w, 400, "当前账号无法找回密码")
		return
	}
	verification, err := randomToken()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "校验失败，请稍后重试")
		return
	}
	now := time.Now()
	_, err = a.db.ExecContext(r.Context(), "INSERT INTO reset_requests(token_hash,user_id,email,expires_at,created_at) VALUES(?,?,?,?,?)", hashToken(verification), id, email, appcommon.FormatTimestamp(now.Add(10*time.Minute)), appcommon.FormatTimestamp(now))
	if err != nil {
		common.WriteError(w, 500, "校验失败，请稍后重试")
		return
	}
	common.WriteJSON(w, 200, map[string]any{"verificationToken": verification, "channels": []map[string]any{{"id": "email", "name": "邮件", "maskedTo": maskEmail(email), "requiresTo": true}}})
}

func (a *Router) handlePasswordResetSend(w http.ResponseWriter, r *http.Request) {
	var body passwordResetSendBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	var userID int64
	var email string
	var expires, lastSent string
	var used int64
	base := settings.LoadSystemBaseConfig(r.Context(), a.db)
	cooldownSeconds := int64(base.PasswordResetSendCooldownMinutes * 60)
	if err := a.db.QueryRowContext(r.Context(), "SELECT user_id,email,expires_at,used,last_sent_at FROM reset_requests WHERE token_hash=?", hashToken(body.VerificationToken)).Scan(&userID, &email, &expires, &used, &lastSent); err != nil || used != 0 || expires < appcommon.NowTimestamp() {
		common.WriteError(w, 400, "发送验证码失败")
		return
	}
	if body.Channel != "email" || !strings.EqualFold(strings.TrimSpace(body.VerifyEmail), strings.TrimSpace(email)) {
		common.WriteError(w, 400, "验证邮箱与账号邮箱不一致")
		return
	}
	if lastSentTime, parseErr := appcommon.ParseTimestamp(lastSent); parseErr == nil {
		remaining := int64(lastSentTime.Add(time.Duration(cooldownSeconds) * time.Second).Sub(time.Now()).Seconds())
		if remaining > 0 {
			common.WriteError(w, http.StatusTooManyRequests, "验证码已发送，请于 "+strconv.FormatInt(remaining, 10)+" 秒后再试")
			return
		}
	}
	if err := a.ensureResetCodeSendAllowed(r.Context(), email, base.PasswordResetRateLimitMinutes); err != nil {
		var limited resetRateLimitedError
		if errors.As(err, &limited) {
			common.WriteError(w, http.StatusTooManyRequests, err.Error())
		} else {
			common.WriteError(w, 500, "发送验证码失败")
		}
		return
	}
	codeValue, err := randomCode()
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "发送验证码失败")
		return
	}
	code := fmt.Sprintf("%06d", codeValue)
	var raw string
	if err := a.db.QueryRowContext(r.Context(), "SELECT config FROM pwd_recover_config WHERE name='email'").Scan(&raw); err != nil {
		common.WriteError(w, 500, "发送验证码失败")
		return
	}
	config := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &config)
	config["resetCode"] = code
	var username string
	if err := a.db.QueryRowContext(r.Context(), "SELECT username FROM users WHERE id=?", userID).Scan(&username); err != nil {
		common.WriteError(w, 500, "发送验证码失败")
		return
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(base.ResetCodeTTLMinutes) * time.Minute)
	config["resetBrand"] = base.LoginName
	config["resetUsername"] = username
	config["resetExpiresAt"] = expiresAt.Format("2006-01-02 15:04:05")
	config["resetRequestIP"] = common.ClientIP(r)
	if err := settings.SendSettingsTestEmail(config, email); err != nil {
		common.WriteError(w, 502, "发送验证码失败")
		return
	}
	_, err = a.db.ExecContext(r.Context(), "UPDATE reset_requests SET code_hash=?,expires_at=?,last_sent_at=? WHERE token_hash=?", hashToken(code), appcommon.FormatTimestamp(expiresAt), appcommon.FormatTimestamp(now), hashToken(body.VerificationToken))
	if err != nil {
		common.WriteError(w, 500, "发送验证码失败")
		return
	}
	if err := a.recordResetCodeSent(r.Context(), email); err != nil {
		log.Printf("Record password reset send log failed: %s", err)
	}
	common.WriteJSON(w, 200, map[string]any{"status": "ok", "cooldownSeconds": int(cooldownSeconds)})
}

// ensureResetCodeSendAllowed 检查同一邮箱在统计窗口内的发送次数是否超限，并顺带清理过期记录。
// 超限时按窗口内最早一次发送时间推算剩余等待分钟数返回。
func (a *Router) ensureResetCodeSendAllowed(ctx context.Context, email string, windowMinutes int) error {
	now := time.Now()
	windowStart := appcommon.FormatTimestamp(now.Add(-time.Duration(windowMinutes) * time.Minute))
	normalized := strings.ToLower(strings.TrimSpace(email))
	var count int
	if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reset_send_log WHERE email=? AND sent_at>?", normalized, windowStart).Scan(&count); err != nil {
		return err
	}
	if count < defaultResetRateLimitMax {
		_, err := a.db.ExecContext(ctx, "DELETE FROM reset_send_log WHERE sent_at<=?", windowStart)
		return err
	}
	minutes := int64(windowMinutes)
	var oldest string
	if err := a.db.QueryRowContext(ctx, "SELECT COALESCE(MIN(sent_at),'') FROM reset_send_log WHERE email=? AND sent_at>?", normalized, windowStart).Scan(&oldest); err != nil {
		return err
	}
	if oldest != "" {
		if oldestTime, parseErr := appcommon.ParseTimestamp(oldest); parseErr == nil {
			if remaining := oldestTime.Add(time.Duration(windowMinutes) * time.Minute).Sub(now); remaining > 0 {
				minutes = int64((remaining + time.Minute - 1) / time.Minute)
			}
		}
	}
	if minutes < 1 {
		minutes = 1
	}
	return resetRateLimitedError{Minutes: minutes}
}

func (a *Router) recordResetCodeSent(ctx context.Context, email string) error {
	now := time.Now()
	_, err := a.db.ExecContext(ctx, "INSERT INTO reset_send_log(email,sent_at) VALUES(?,?)", strings.ToLower(strings.TrimSpace(email)), appcommon.FormatTimestamp(now))
	_, _ = a.db.ExecContext(ctx, "DELETE FROM reset_send_log WHERE sent_at < ?", appcommon.FormatTimestamp(now.Add(-7*24*time.Hour)))
	return err
}

func (a *Router) handlePasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var body passwordResetConfirmBody
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	if len(body.NewPassword) < 6 || body.NewPassword != body.ConfirmPassword {
		common.WriteError(w, 400, "新密码至少 6 位且两次输入必须一致")
		return
	}
	var userID int64
	var email, codeHash string
	var expires string
	var used int64
	if err := a.db.QueryRowContext(r.Context(), "SELECT user_id,email,code_hash,expires_at,used FROM reset_requests WHERE token_hash=?", hashToken(body.VerificationToken)).Scan(&userID, &email, &codeHash, &expires, &used); err != nil || used != 0 || expires < appcommon.NowTimestamp() {
		common.WriteError(w, 400, "请重新完成用户名和图形验证码校验")
		return
	}
	if codeHash != hashToken(strings.TrimSpace(body.Code)) {
		common.WriteError(w, 400, "验证码不正确")
		return
	}
	hashed, err := security.HashPassword(body.NewPassword)
	if err != nil {
		common.WriteError(w, 500, "密码重置失败")
		return
	}
	var resetUsername string
	if err := a.db.QueryRowContext(r.Context(), "SELECT username FROM users WHERE id=?", userID).Scan(&resetUsername); err != nil {
		common.WriteError(w, 500, "密码重置失败")
		return
	}
	if _, err = a.db.ExecContext(r.Context(), "UPDATE users SET pass=? WHERE id=?", hashed, userID); err != nil {
		common.WriteError(w, 500, "密码重置失败")
		return
	}
	cleared, err := a.clearLoginFailures(r.Context(), resetUsername)
	if err != nil {
		log.Printf("Clear login failures failed: %s", err)
	}
	var revoked int64
	if result, revokeErr := a.db.ExecContext(r.Context(), "DELETE FROM sessions WHERE user_id=?", userID); revokeErr != nil {
		log.Printf("Revoke user sessions failed: %s", revokeErr)
	} else if n, rowErr := result.RowsAffected(); rowErr == nil {
		revoked = n
	}
	detail := "用户 " + resetUsername + " 已通过找回密码流程修改密码"
	var extras []string
	if cleared > 0 {
		extras = append(extras, "解除登录失败锁定")
	}
	if revoked > 0 {
		extras = append(extras, "退出该账号全部登录会话")
	}
	if len(extras) > 0 {
		detail += "并" + strings.Join(extras, "、")
	}
	a.recordAuditEvent(r.Context(), resetUsername, common.ClientIP(r), service.AuditModuleAuth, "重置密码", resetUsername, detail, service.AuditResultSuccess)
	_, _ = a.db.ExecContext(r.Context(), "UPDATE reset_requests SET used=1 WHERE token_hash=?", hashToken(body.VerificationToken))
	_ = email
	common.WriteJSON(w, 200, map[string]string{"status": "ok"})
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomCode() (int, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return int(uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3])) % 1000000, nil
}

func readRandomInt(target *int, min, max int) error {
	if min > max {
		return fmt.Errorf("invalid random range")
	}
	value, err := randomCode()
	if err != nil {
		return err
	}
	*target = min + value%(max-min+1)
	return nil
}
func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func maskEmail(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	name := parts[0]
	if len(name) <= 2 {
		name = "*"
	} else {
		name = name[:1] + "***" + name[len(name)-1:]
	}
	return name + "@" + parts[1]
}
