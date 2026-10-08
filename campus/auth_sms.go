package campus

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/usememos/memos/store"
)

// ============================================================
// 校园注册 / 找回密码（手机号 + 短信验证码）
//
// 短信通道抽象为 Provider：
//   - 开发模式（默认）：验证码写入日志并在响应中回显 devCode，便于演示与试点；
//   - Webhook：设置环境变量 CAMPUS_SMS_WEBHOOK 后，向该地址 POST
//     {"phone","code","purpose"}，可对接阿里云/腾讯云短信网关或学校统一消息平台。
// ============================================================

const (
	smsPurposeRegister = "register"
	smsPurposeReset    = "reset"
	smsCodeTTL         = 10 * time.Minute
)

// sendSMSCode 发送验证码。配置 webhook 时走真实通道，否则走开发模式。
// 返回 devCode（仅开发模式非空）。
func (s *Service) sendSMSCode(phone, code, purpose string) (devCode string, err error) {
	webhook := os.Getenv("CAMPUS_SMS_WEBHOOK")
	if webhook == "" {
		slog.Info("campus sms dev-mode: verification code generated",
			slog.String("phone", phone), slog.String("purpose", purpose), slog.String("code", code))
		return code, nil
	}
	body := fmt.Sprintf(`{"phone":%q,"code":%q,"purpose":%q}`, phone, code, purpose)
	resp, err := http.Post(webhook, "application/json", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("sms webhook returned status %d", resp.StatusCode)
	}
	return "", nil
}

func generateSMSCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

type smsRequestBody struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose"` // register / reset
}

func validPhone(p string) bool {
	if len(p) != 11 {
		return false
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// handleSendSMS 发送短信验证码（注册 / 找回密码共用）。
func (s *Service) handleSendSMS(c *echo.Context) error {
	req := &smsRequestBody{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if !validPhone(req.Phone) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid phone number")
	}
	if req.Purpose != smsPurposeRegister && req.Purpose != smsPurposeReset {
		return echo.NewHTTPError(http.StatusBadRequest, "purpose must be register or reset")
	}
	ctx := c.Request().Context()
	code := generateSMSCode()
	expire := time.Now().Add(smsCodeTTL).Unix()
	// 同手机号同用途覆盖旧验证码（upsert）。
	if _, err := s.exec(ctx, `DELETE FROM campus_sms_code WHERE phone = ? AND purpose = ?`, req.Phone, req.Purpose); err != nil {
		return err
	}
	if _, err := s.exec(ctx, `INSERT INTO campus_sms_code (phone, code, purpose, expire_ts, used) VALUES (?, ?, ?, ?, 0)`,
		req.Phone, code, req.Purpose, expire); err != nil {
		return err
	}
	devCode, err := s.sendSMSCode(req.Phone, code, req.Purpose)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "failed to send sms: "+err.Error())
	}
	resp := map[string]any{"sent": true}
	if devCode != "" {
		// 开发模式回显，便于非技术同学演示；配置 CAMPUS_SMS_WEBHOOK 后该字段为空。
		resp["devCode"] = devCode
	}
	return c.JSON(http.StatusOK, resp)
}

// verifySMSCode 校验并消费验证码（一次性）。
func (s *Service) verifySMSCode(c *echo.Context, phone, code, purpose string) error {
	ctx := c.Request().Context()
	var storedCode string
	var expireTs int64
	var used int
	err := s.queryRow(ctx, `SELECT code, expire_ts, used FROM campus_sms_code WHERE phone = ? AND purpose = ?`, phone, purpose).
		Scan(&storedCode, &expireTs, &used)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "请先获取验证码")
	}
	if used != 0 || time.Now().Unix() > expireTs || storedCode != strings.TrimSpace(code) {
		return echo.NewHTTPError(http.StatusBadRequest, "验证码错误或已过期")
	}
	if _, err := s.exec(ctx, `UPDATE campus_sms_code SET used = 1 WHERE phone = ? AND purpose = ?`, phone, purpose); err != nil {
		return err
	}
	return nil
}

type registerBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Code     string `json:"code"`
}

// handleRegister 简化注册：用户名（唯一）+ 密码 + 手机号 + 短信验证。
func (s *Service) handleRegister(c *echo.Context) error {
	req := &registerBody{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 || !validPhone(req.Phone) {
		return echo.NewHTTPError(http.StatusBadRequest, "用户名不能为空，密码至少 6 位，手机号 11 位")
	}
	if err := s.verifySMSCode(c, req.Phone, req.Code, smsPurposeRegister); err != nil {
		return err
	}
	ctx := c.Request().Context()
	// 用户名不可重复（store 层也有唯一约束兜底）。
	existing, err := s.Store.GetUser(ctx, &store.FindUser{Username: &req.Username})
	if err != nil {
		return err
	}
	if existing != nil {
		return echo.NewHTTPError(http.StatusConflict, "用户名已被使用")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user, err := s.Store.CreateUser(ctx, &store.User{
		Username:     req.Username,
		Nickname:     req.Username,
		PasswordHash: string(hash),
	})
	if err != nil {
		return err
	}
	// 手机号存入校园设置，用于找回密码校验。
	if err := s.putSetting(ctx, user.ID, "phone", req.Phone); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{
		"ok":       true,
		"username": user.Username,
	})
}

type resetPasswordBody struct {
	Username    string `json:"username"`
	Phone       string `json:"phone"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// handleResetPassword 找回密码：用户名 + 手机号 + 短信验证 → 重置密码。
func (s *Service) handleResetPassword(c *echo.Context) error {
	req := &resetPasswordBody{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if len(req.NewPassword) < 6 {
		return echo.NewHTTPError(http.StatusBadRequest, "密码至少 6 位")
	}
	ctx := c.Request().Context()
	user, err := s.Store.GetUser(ctx, &store.FindUser{Username: &req.Username})
	if err != nil {
		return err
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "用户不存在")
	}
	// 校验手机号与注册时一致。
	phone, err := s.getSetting(ctx, user.ID, "phone")
	if err != nil {
		return err
	}
	if phone == "" || phone != req.Phone {
		return echo.NewHTTPError(http.StatusBadRequest, "手机号与注册信息不一致")
	}
	if err := s.verifySMSCode(c, req.Phone, req.Code, smsPurposeReset); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.Store.UpdateUser(ctx, &store.UpdateUser{ID: user.ID, PasswordHash: ptrOf(string(hash))}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}
